package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pxc1984/nnkl-backend/store/models"

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
	if err := store.db.AutoMigrate(&models.User{}, &models.Session{}, &models.Blob{}, &models.Upload{}, &models.AuditLog{}, &models.QuerySession{}); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("migrate postgres schema: %w", err)
	}
	if err := store.migrateLegacyBlobTables(context.Background()); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("migrate legacy blob tables: %w", err)
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
	err := s.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}

func (s *PostgresStore) CreateUser(ctx context.Context, params models.CreateUserParams) (*models.User, error) {
	user := &models.User{
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

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *PostgresStore) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *PostgresStore) UpdateUserLastLogin(ctx context.Context, userID string, lastLoginAt time.Time) error {
	return s.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"last_login_at": lastLoginAt, "updated_at": lastLoginAt}).Error
}

func (s *PostgresStore) UpdateUser(ctx context.Context, userID string, params models.UpdateUserParams) (*models.User, error) {
	updates := map[string]any{"updated_at": time.Now().UTC()}
	if params.Name != nil {
		updates["name"] = *params.Name
	}
	if params.AvatarData != nil {
		updates["avatar_data"] = params.AvatarData
	}
	if params.AvatarURL != nil {
		updates["avatar_url"] = *params.AvatarURL
	}
	if err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetUserByID(ctx, userID)
}

func (s *PostgresStore) CreateSession(ctx context.Context, params models.CreateSessionParams) (*models.Session, error) {
	session := &models.Session{
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

func (s *PostgresStore) GetSessionByID(ctx context.Context, sessionID string) (*models.Session, error) {
	var session models.Session
	if err := s.db.WithContext(ctx).Preload("User").First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *PostgresStore) GetSessionByRefreshTokenHash(ctx context.Context, hash string) (*models.Session, error) {
	var session models.Session
	if err := s.db.WithContext(ctx).Preload("User").Where("refresh_token_hash = ?", hash).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *PostgresStore) UpdateSessionToken(ctx context.Context, params models.UpdateSessionTokenParams) (*models.Session, error) {
	if err := s.db.WithContext(ctx).
		Model(&models.Session{}).
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
		Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("last_used_at", lastUsedAt).Error
}

func (s *PostgresStore) ListUserSessions(ctx context.Context, userID string) ([]models.Session, error) {
	var sessions []models.Session
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&sessions).Error
	return sessions, err
}

func (s *PostgresStore) DeleteSessionByID(ctx context.Context, sessionID string) error {
	return s.db.WithContext(ctx).Delete(&models.Session{}, "id = ?", sessionID).Error
}

func (s *PostgresStore) DeleteSessionByUserAndHash(ctx context.Context, userID, hash string) error {
	return s.db.WithContext(ctx).Delete(&models.Session{}, "user_id = ? AND refresh_token_hash = ?", userID, hash).Error
}

func (s *PostgresStore) DeleteUserSessions(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).Delete(&models.Session{}, "user_id = ?", userID).Error
}

func (s *PostgresStore) CreateBlob(ctx context.Context, params models.CreateBlobParams) (*models.Blob, error) {
	blob := &models.Blob{
		ID:          uuid.NewString(),
		Filename:    params.Filename,
		FileType:    strings.ToLower(params.FileType),
		ContentType: params.ContentType,
		SizeBytes:   params.SizeBytes,
		SHA256:      params.SHA256,
		Content:     params.Content,
	}
	if err := s.db.WithContext(ctx).Create(blob).Error; err != nil {
		return nil, err
	}
	return blob, nil
}

func (s *PostgresStore) GetBlobByID(ctx context.Context, id string) (*models.Blob, error) {
	var blob models.Blob
	if err := s.db.WithContext(ctx).First(&blob, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &blob, nil
}

func (s *PostgresStore) GetBlobBySHA256(ctx context.Context, sha256 string) (*models.Blob, error) {
	var blob models.Blob
	if err := s.db.WithContext(ctx).Where("sha256 = ?", sha256).First(&blob).Error; err != nil {
		return nil, err
	}
	return &blob, nil
}

func (s *PostgresStore) CreateUpload(ctx context.Context, params models.CreateUploadParams) (*models.Upload, error) {
	upload := &models.Upload{
		ID:          params.ID,
		InputBlobID: params.InputBlobID,
		Status:      params.Status,
		Language:    params.Language,
		Error:       params.Error,
	}
	if upload.ID == "" {
		upload.ID = uuid.NewString()
	}
	if err := s.db.WithContext(ctx).Create(upload).Error; err != nil {
		return nil, err
	}
	return s.GetUploadByID(ctx, upload.ID)
}

func (s *PostgresStore) ListUploads(ctx context.Context, params models.ListUploadsParams) ([]models.Upload, int64, error) {
	page := max(params.Page, 1)
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := s.db.WithContext(ctx).Model(&models.Upload{}).Joins("InputBlob")
	if params.Query != "" {
		query = query.Where(`"InputBlob"."filename" ILIKE ?`, "%"+params.Query+"%")
	}
	if params.FileType != "" {
		query = query.Where(`"InputBlob"."file_type" = ?`, strings.ToLower(params.FileType))
	}
	if params.Status != "" {
		query = query.Where(`"uploads"."status" = ?`, params.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var uploads []models.Upload
	err := query.Preload("InputBlob").Preload("OutputBlob").Order("uploads.created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&uploads).Error
	return uploads, total, err
}

func (s *PostgresStore) GetUploadByID(ctx context.Context, id string) (*models.Upload, error) {
	var upload models.Upload
	if err := s.db.WithContext(ctx).Preload("InputBlob").Preload("OutputBlob").First(&upload, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &upload, nil
}

func (s *PostgresStore) UpdateUpload(ctx context.Context, id string, params models.UpdateUploadParams) (*models.Upload, error) {
	updates := map[string]any{}
	if params.InputBlobID != nil {
		updates["input_blob"] = *params.InputBlobID
	}
	if params.OutputBlobID != nil {
		updates["output_blob"] = *params.OutputBlobID
	}
	if params.Status != nil {
		updates["status"] = *params.Status
	}
	if params.Language != nil {
		updates["language"] = *params.Language
	}
	if params.Error != nil {
		updates["error"] = *params.Error
	}
	if err := s.db.WithContext(ctx).Model(&models.Upload{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetUploadByID(ctx, id)
}

func (s *PostgresStore) DeleteUploadByID(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var upload models.Upload
		if err := tx.First(&upload, "id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Upload{}, "id = ?", id).Error; err != nil {
			return err
		}
		if err := deleteBlobIfUnreferenced(tx, upload.InputBlobID); err != nil {
			return err
		}
		if upload.OutputBlobID != nil {
			if err := deleteBlobIfUnreferenced(tx, *upload.OutputBlobID); err != nil {
				return err
			}
		}
		return nil
	})
}

func deleteBlobIfUnreferenced(tx *gorm.DB, blobID string) error {
	if blobID == "" {
		return nil
	}
	var count int64
	if err := tx.Model(&models.Upload{}).Where("input_blob = ? OR output_blob = ?", blobID, blobID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Delete(&models.Blob{}, "id = ?", blobID).Error
}

func (s *PostgresStore) migrateLegacyBlobTables(ctx context.Context) error {
	exists, err := s.tableExists(ctx, "input_blobs")
	if err != nil || !exists {
		return err
	}

	queries := []string{
		`INSERT INTO blobs (id, filename, file_type, content_type, size_bytes, sha256, content, created_at, updated_at)
		 SELECT id, filename, file_type, content_type, size_bytes, sha256, content, created_at, updated_at
		 FROM input_blobs
		 ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO blobs (id, filename, file_type, content_type, size_bytes, sha256, content, created_at, updated_at)
		 SELECT pr.id,
		        CONCAT(COALESCE(NULLIF(split_part(ib.filename, '.', 1), ''), 'upload'), '.md'),
		        'markdown',
		        COALESCE(pr.content_type, 'text/markdown'),
		        octet_length(convert_to(pr.content_text, 'UTF8')),
		        NULL,
		        convert_to(pr.content_text, 'UTF8'),
		        pr.created_at,
		        pr.updated_at
		 FROM parse_results pr
		 JOIN parse_jobs pj ON pj.id = pr.job_id
		 LEFT JOIN input_blobs ib ON ib.id = pj.input_blob_id
		 ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO uploads (id, input_blob, output_blob, status, language, error, created_at, updated_at)
		 SELECT pj.input_blob_id,
		        pj.input_blob_id,
		        pr.id,
		        pj.status,
		        pj.language,
		        pj.error,
		        pj.created_at,
		        pj.updated_at
		 FROM parse_jobs pj
		 LEFT JOIN parse_results pr ON pr.job_id = pj.id
		 ON CONFLICT (id) DO UPDATE SET
		   input_blob = EXCLUDED.input_blob,
		   output_blob = EXCLUDED.output_blob,
		   status = EXCLUDED.status,
		   language = EXCLUDED.language,
		   error = EXCLUDED.error,
		   updated_at = EXCLUDED.updated_at`,
	}

	for _, query := range queries {
		if err := s.db.WithContext(ctx).Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) tableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	err := s.db.WithContext(ctx).Raw(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?)`, tableName).Scan(&exists).Error
	return exists, err
}

func max(value, fallback int) int {
	if value > fallback {
		return value
	}
	return fallback
}

func (s *PostgresStore) CreateAuditLog(ctx context.Context, entry *models.AuditLog) error {
	entry.Timedate = entry.Timedate.UTC()
	return s.db.WithContext(ctx).Create(entry).Error
}

func (s *PostgresStore) CreateQuerySession(ctx context.Context, params models.CreateQuerySessionParams) (*models.QuerySession, error) {
	session := &models.QuerySession{
		ID:         uuid.NewString(),
		UserID:     params.UserID,
		Query:      params.Query,
		Mode:       params.Mode,
		Response:   params.Response,
		References: params.References,
	}
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

func (s *PostgresStore) GetQuerySessionByID(ctx context.Context, id string) (*models.QuerySession, error) {
	var session models.QuerySession
	if err := s.db.WithContext(ctx).First(&session, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *PostgresStore) ListQuerySessions(ctx context.Context, userID string, params models.ListQuerySessionsParams) ([]models.QuerySession, int64, error) {
	page := max(params.Page, 1)
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := s.db.WithContext(ctx).Model(&models.QuerySession{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var sessions []models.QuerySession
	err := query.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&sessions).Error
	return sessions, total, err
}
