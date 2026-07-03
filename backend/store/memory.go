package store

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type InMemoryStore struct {
	mu       sync.RWMutex
	users    map[string]User
	byEmail  map[string]string
	sessions map[string]Session
	byHash   map[string]string
	blobs    map[string]InputBlob
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		users:    make(map[string]User),
		byEmail:  make(map[string]string),
		sessions: make(map[string]Session),
		byHash:   make(map[string]string),
		blobs:    make(map[string]InputBlob),
	}
}

func (s *InMemoryStore) Backend() string {
	return "memory"
}

func (s *InMemoryStore) Ping(context.Context) error {
	return nil
}

func (s *InMemoryStore) Close() error {
	return nil
}

func (s *InMemoryStore) CountUsers(context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.users)), nil
}

func (s *InMemoryStore) CreateUser(_ context.Context, params CreateUserParams) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byEmail[params.Email]; exists {
		return nil, gorm.ErrDuplicatedKey
	}
	now := time.Now().UTC()
	user := User{
		ID:            uuid.NewString(),
		Email:         params.Email,
		Name:          params.Name,
		Role:          params.Role,
		PasswordHash:  params.PasswordHash,
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.users[user.ID] = user
	s.byEmail[user.Email] = user.ID
	return cloneUser(user), nil
}

func (s *InMemoryStore) GetUserByEmail(_ context.Context, email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userID, ok := s.byEmail[email]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	user, ok := s.users[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return cloneUser(user), nil
}

func (s *InMemoryStore) GetUserByID(_ context.Context, userID string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return cloneUser(user), nil
}

func (s *InMemoryStore) UpdateUserLastLogin(_ context.Context, userID string, lastLoginAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	user.LastLoginAt = &lastLoginAt
	user.UpdatedAt = lastLoginAt
	s.users[userID] = user
	return nil
}

func (s *InMemoryStore) CreateSession(_ context.Context, params CreateSessionParams) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[params.UserID]; !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if _, exists := s.byHash[params.RefreshTokenHash]; exists {
		return nil, gorm.ErrDuplicatedKey
	}
	now := params.LastUsedAt
	session := Session{
		ID:               uuid.NewString(),
		UserID:           params.UserID,
		RefreshTokenHash: params.RefreshTokenHash,
		IP:               params.IP,
		UserAgent:        params.UserAgent,
		CreatedAt:        now,
		LastUsedAt:       params.LastUsedAt,
		ExpiresAt:        params.ExpiresAt,
	}
	s.sessions[session.ID] = session
	s.byHash[session.RefreshTokenHash] = session.ID
	return cloneSession(session), nil
}

func (s *InMemoryStore) GetSessionByID(_ context.Context, sessionID string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if user, ok := s.users[session.UserID]; ok {
		session.User = user
	}
	return cloneSession(session), nil
}

func (s *InMemoryStore) GetSessionByRefreshTokenHash(_ context.Context, hash string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessionID, ok := s.byHash[hash]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if user, ok := s.users[session.UserID]; ok {
		session.User = user
	}
	return cloneSession(session), nil
}

func (s *InMemoryStore) UpdateSessionToken(_ context.Context, params UpdateSessionTokenParams) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[params.SessionID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if existingSessionID, exists := s.byHash[params.RefreshTokenHash]; exists && existingSessionID != params.SessionID {
		return nil, gorm.ErrDuplicatedKey
	}
	delete(s.byHash, session.RefreshTokenHash)
	session.RefreshTokenHash = params.RefreshTokenHash
	session.ExpiresAt = params.ExpiresAt
	session.LastUsedAt = params.LastUsedAt
	s.sessions[params.SessionID] = session
	s.byHash[params.RefreshTokenHash] = params.SessionID
	if user, ok := s.users[session.UserID]; ok {
		session.User = user
	}
	return cloneSession(session), nil
}

func (s *InMemoryStore) TouchSession(_ context.Context, sessionID string, lastUsedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	session.LastUsedAt = lastUsedAt
	s.sessions[sessionID] = session
	return nil
}

func (s *InMemoryStore) ListUserSessions(_ context.Context, userID string) ([]Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessions := make([]Session, 0)
	for _, session := range s.sessions {
		if session.UserID == userID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (s *InMemoryStore) DeleteSessionByID(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	delete(s.sessions, sessionID)
	delete(s.byHash, session.RefreshTokenHash)
	return nil
}

func (s *InMemoryStore) DeleteSessionByUserAndHash(_ context.Context, userID, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionID, ok := s.byHash[hash]
	if !ok {
		return nil
	}
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}
	if session.UserID != userID {
		return errors.New("session does not belong to user")
	}
	delete(s.sessions, sessionID)
	delete(s.byHash, hash)
	return nil
}

func (s *InMemoryStore) DeleteUserSessions(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sessionID, session := range s.sessions {
		if session.UserID != userID {
			continue
		}
		delete(s.byHash, session.RefreshTokenHash)
		delete(s.sessions, sessionID)
	}
	return nil
}

func (s *InMemoryStore) CreateInputBlob(_ context.Context, params CreateInputBlobParams) (*InputBlob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	blob := InputBlob{
		ID:          uuid.NewString(),
		Filename:    params.Filename,
		FileType:    params.FileType,
		ContentType: params.ContentType,
		Tags:        pq.StringArray(append([]string(nil), params.Tags...)),
		SizeBytes:   params.SizeBytes,
		SHA256:      params.SHA256,
		Content:     append([]byte(nil), params.Content...),
		CreatedAt:   now,
	}
	s.blobs[blob.ID] = blob
	return cloneInputBlob(blob), nil
}

func (s *InMemoryStore) ListInputBlobs(_ context.Context, params ListInputBlobsParams) ([]InputBlob, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]InputBlob, 0)
	for _, blob := range s.blobs {
		if params.Query != "" && !strings.Contains(strings.ToLower(blob.Filename), strings.ToLower(params.Query)) {
			continue
		}
		if params.FileType != "" && blob.FileType != strings.ToLower(params.FileType) {
			continue
		}
		if !containsAllTags(blob.Tags, params.Tags) {
			continue
		}
		filtered = append(filtered, blob)
	}

	slices.SortFunc(filtered, func(a, b InputBlob) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})

	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return []InputBlob{}, int64(len(filtered)), nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	items := make([]InputBlob, 0, end-start)
	for _, blob := range filtered[start:end] {
		items = append(items, *cloneInputBlob(blob))
	}
	return items, int64(len(filtered)), nil
}

func cloneUser(user User) *User {
	clone := user
	return &clone
}

func cloneSession(session Session) *Session {
	clone := session
	return &clone
}

func cloneInputBlob(blob InputBlob) *InputBlob {
	clone := blob
	clone.Tags = pq.StringArray(append([]string(nil), blob.Tags...))
	clone.Content = append([]byte(nil), blob.Content...)
	return &clone
}

func containsAllTags(blobTags, filterTags []string) bool {
	if len(filterTags) == 0 {
		return true
	}
	available := make(map[string]struct{}, len(blobTags))
	for _, tag := range blobTags {
		available[strings.ToLower(tag)] = struct{}{}
	}
	for _, tag := range filterTags {
		if _, ok := available[strings.ToLower(tag)]; !ok {
			return false
		}
	}
	return true
}
