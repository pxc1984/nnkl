package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/pxc1984/nnkl-backend/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresStore struct {
	db    *gorm.DB
	sqldb *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		utils.Settings.PostgresHost,
		utils.Settings.PostgresPort,
		utils.Settings.PostgresUser,
		utils.Settings.PostgresPassword,
		utils.Settings.PostgresDB,
		utils.Settings.PostgresSSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	sqldb, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("extract postgres sql db: %w", err)
	}

	store := &PostgresStore{db: db, sqldb: sqldb}
	if err := store.db.AutoMigrate(&User{}, &Session{}, &InputBlob{}); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("migrate postgres schema: %w", err)
	}
	if err := store.Ping(context.Background()); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return store, nil
}

func (s *PostgresStore) Backend() string {
	return "postgres"
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.sqldb.PingContext(ctx)
}

func (s *PostgresStore) Close() error {
	return s.sqldb.Close()
}

func (s *PostgresStore) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&User{}).Count(&count).Error
	return count, err
}

func (s *PostgresStore) CreateUser(ctx context.Context, params CreateUserParams) (*User, error) {
	user := &User{
		ID:            uuid.NewString(),
		Email:         params.Email,
		Name:          params.Name,
		Role:          params.Role,
		PasswordHash:  params.PasswordHash,
		EmailVerified: true,
	}
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *PostgresStore) GetUserByID(ctx context.Context, userID string) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *PostgresStore) UpdateUserLastLogin(ctx context.Context, userID string, lastLoginAt time.Time) error {
	return s.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"last_login_at": lastLoginAt, "updated_at": lastLoginAt}).Error
}

func (s *PostgresStore) CreateSession(ctx context.Context, params CreateSessionParams) (*Session, error) {
	session := &Session{
		ID:               uuid.NewString(),
		UserID:           params.UserID,
		RefreshTokenHash: params.RefreshTokenHash,
		IP:               params.IP,
		UserAgent:        params.UserAgent,
		LastUsedAt:       params.LastUsedAt,
		ExpiresAt:        params.ExpiresAt,
	}
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func (s *PostgresStore) GetSessionByID(ctx context.Context, sessionID string) (*Session, error) {
	var session Session
	if err := s.db.WithContext(ctx).Preload("User").First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *PostgresStore) GetSessionByRefreshTokenHash(ctx context.Context, hash string) (*Session, error) {
	var session Session
	if err := s.db.WithContext(ctx).Preload("User").Where("refresh_token_hash = ?", hash).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *PostgresStore) UpdateSessionToken(ctx context.Context, params UpdateSessionTokenParams) (*Session, error) {
	if err := s.db.WithContext(ctx).
		Model(&Session{}).
		Where("id = ?", params.SessionID).
		Updates(map[string]any{
			"refresh_token_hash": params.RefreshTokenHash,
			"expires_at":         params.ExpiresAt,
			"last_used_at":       params.LastUsedAt,
		}).Error; err != nil {
		return nil, err
	}
	return s.GetSessionByID(ctx, params.SessionID)
}

func (s *PostgresStore) TouchSession(ctx context.Context, sessionID string, lastUsedAt time.Time) error {
	return s.db.WithContext(ctx).
		Model(&Session{}).
		Where("id = ?", sessionID).
		Update("last_used_at", lastUsedAt).Error
}

func (s *PostgresStore) ListUserSessions(ctx context.Context, userID string) ([]Session, error) {
	var sessions []Session
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&sessions).Error
	return sessions, err
}

func (s *PostgresStore) DeleteSessionByID(ctx context.Context, sessionID string) error {
	return s.db.WithContext(ctx).Delete(&Session{}, "id = ?", sessionID).Error
}

func (s *PostgresStore) DeleteSessionByUserAndHash(ctx context.Context, userID, hash string) error {
	return s.db.WithContext(ctx).Delete(&Session{}, "user_id = ? AND refresh_token_hash = ?", userID, hash).Error
}

func (s *PostgresStore) DeleteUserSessions(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).Delete(&Session{}, "user_id = ?", userID).Error
}

func (s *PostgresStore) CreateInputBlob(ctx context.Context, params CreateInputBlobParams) (*InputBlob, error) {
	blob := &InputBlob{
		ID:          uuid.NewString(),
		Filename:    params.Filename,
		FileType:    strings.ToLower(params.FileType),
		ContentType: params.ContentType,
		Tags:        pq.StringArray(params.Tags),
		SizeBytes:   params.SizeBytes,
		SHA256:      params.SHA256,
		Content:     params.Content,
	}
	if err := s.db.WithContext(ctx).Create(blob).Error; err != nil {
		return nil, err
	}
	return blob, nil
}

func (s *PostgresStore) ListInputBlobs(ctx context.Context, params ListInputBlobsParams) ([]InputBlob, int64, error) {
	page := max(params.Page, 1)
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := s.db.WithContext(ctx).Model(&InputBlob{})
	if params.Query != "" {
		query = query.Where("filename ILIKE ?", "%"+params.Query+"%")
	}
	if params.FileType != "" {
		query = query.Where("file_type = ?", strings.ToLower(params.FileType))
	}
	if len(params.Tags) > 0 {
		query = query.Where("tags @> ?", pq.Array(params.Tags))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var blobs []InputBlob
	err := query.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&blobs).Error
	return blobs, total, err
}

func max(value, fallback int) int {
	if value > fallback {
		return value
	}
	return fallback
}
