package store

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pxc1984/nnkl-backend/store/models"
	"gorm.io/gorm"
)

type InMemoryStore struct {
	mu              sync.RWMutex
	users           map[string]models.User
	byEmail         map[string]string
	sessions        map[string]models.Session
	byHash          map[string]string
	blobs           map[string]models.Blob
	uploads         map[string]models.Upload
	querySessions   map[string]models.QuerySession
	auditLogs       []models.AuditLog
	auditLogCounter uint
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		users:         make(map[string]models.User),
		byEmail:       make(map[string]string),
		sessions:      make(map[string]models.Session),
		byHash:        make(map[string]string),
		blobs:         make(map[string]models.Blob),
		uploads:       make(map[string]models.Upload),
		querySessions: make(map[string]models.QuerySession),
		auditLogs:     make([]models.AuditLog, 0),
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

func (s *InMemoryStore) CreateUser(_ context.Context, params models.CreateUserParams) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byEmail[params.Email]; exists {
		return nil, gorm.ErrDuplicatedKey
	}
	now := time.Now().UTC()
	user := models.User{
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

func (s *InMemoryStore) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
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

func (s *InMemoryStore) GetUserByID(_ context.Context, userID string) (*models.User, error) {
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

func (s *InMemoryStore) UpdateUser(_ context.Context, userID string, params models.UpdateUserParams) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	if params.Name != nil {
		user.Name = *params.Name
	}
	if params.AvatarData != nil {
		user.AvatarData = params.AvatarData
	}
	if params.AvatarURL != nil {
		user.AvatarURL = params.AvatarURL
	}
	user.UpdatedAt = now
	s.users[userID] = user
	return cloneUser(user), nil
}

func (s *InMemoryStore) CreateSession(_ context.Context, params models.CreateSessionParams) (*models.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[params.UserID]; !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if _, exists := s.byHash[params.RefreshTokenHash]; exists {
		return nil, gorm.ErrDuplicatedKey
	}
	now := params.LastUsedAt
	session := models.Session{
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

func (s *InMemoryStore) GetSessionByID(_ context.Context, sessionID string) (*models.Session, error) {
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

func (s *InMemoryStore) GetSessionByRefreshTokenHash(_ context.Context, hash string) (*models.Session, error) {
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

func (s *InMemoryStore) UpdateSessionToken(_ context.Context, params models.UpdateSessionTokenParams) (*models.Session, error) {
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

func (s *InMemoryStore) ListUserSessions(_ context.Context, userID string) ([]models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessions := make([]models.Session, 0)
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

func (s *InMemoryStore) CreateBlob(_ context.Context, params models.CreateBlobParams) (*models.Blob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	blob := models.Blob{
		ID:          uuid.NewString(),
		Filename:    params.Filename,
		FileType:    params.FileType,
		ContentType: params.ContentType,
		SizeBytes:   params.SizeBytes,
		SHA256:      params.SHA256,
		Content:     append([]byte(nil), params.Content...),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.blobs[blob.ID] = blob
	return cloneBlob(blob), nil
}

func (s *InMemoryStore) GetBlobByID(_ context.Context, id string) (*models.Blob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	blob, ok := s.blobs[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return cloneBlob(blob), nil
}

func (s *InMemoryStore) GetBlobBySHA256(_ context.Context, sha256 string) (*models.Blob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, blob := range s.blobs {
		if blob.SHA256 != nil && *blob.SHA256 == sha256 {
			return cloneBlob(blob), nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *InMemoryStore) CreateUpload(_ context.Context, params models.CreateUploadParams) (*models.Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.blobs[params.InputBlobID]; !ok {
		return nil, gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	uploadID := params.ID
	if uploadID == "" {
		uploadID = uuid.NewString()
	}
	upload := models.Upload{
		ID:           uploadID,
		InputBlobID:  params.InputBlobID,
		Status:       params.Status,
		OutputFormat: params.OutputFormat,
		Language:     params.Language,
		Error:        params.Error,
		CreatedAt:    now,
		UpdatedAt:    now,
		InputBlob:    *cloneBlob(s.blobs[params.InputBlobID]),
	}
	s.uploads[upload.ID] = upload
	return cloneUpload(upload), nil
}

func (s *InMemoryStore) ListUploads(_ context.Context, params models.ListUploadsParams) ([]models.Upload, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]models.Upload, 0)
	for _, upload := range s.uploads {
		blob, ok := s.blobs[upload.InputBlobID]
		if !ok {
			continue
		}
		upload.InputBlob = *cloneBlob(blob)
		if upload.OutputBlobID != nil {
			if outputBlob, ok := s.blobs[*upload.OutputBlobID]; ok {
				upload.OutputBlob = cloneBlob(outputBlob)
			}
		}
		if params.Query != "" && !strings.Contains(strings.ToLower(blob.Filename), strings.ToLower(params.Query)) {
			continue
		}
		if params.FileType != "" && blob.FileType != strings.ToLower(params.FileType) {
			continue
		}
		if params.Status != "" && upload.Status != params.Status {
			continue
		}
		filtered = append(filtered, upload)
	}

	slices.SortFunc(filtered, func(a, b models.Upload) int {
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
		return []models.Upload{}, int64(len(filtered)), nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}

	items := make([]models.Upload, 0, end-start)
	for _, upload := range filtered[start:end] {
		items = append(items, *cloneUpload(upload))
	}
	return items, int64(len(filtered)), nil
}

func (s *InMemoryStore) GetUploadByID(_ context.Context, id string) (*models.Upload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	upload, ok := s.uploads[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	inputBlob, ok := s.blobs[upload.InputBlobID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	upload.InputBlob = *cloneBlob(inputBlob)
	if upload.OutputBlobID != nil {
		if outputBlob, ok := s.blobs[*upload.OutputBlobID]; ok {
			upload.OutputBlob = cloneBlob(outputBlob)
		}
	}
	return cloneUpload(upload), nil
}

func (s *InMemoryStore) UpdateUpload(_ context.Context, id string, params models.UpdateUploadParams) (*models.Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	upload, ok := s.uploads[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	if params.InputBlobID != nil {
		upload.InputBlobID = *params.InputBlobID
	}
	if params.OutputBlobID != nil {
		upload.OutputBlobID = params.OutputBlobID
	}
	if params.ClearOutputBlob {
		upload.OutputBlobID = nil
		upload.OutputBlob = nil
	}
	if params.Status != nil {
		upload.Status = *params.Status
	}
	if params.OutputFormat != nil {
		upload.OutputFormat = *params.OutputFormat
	}
	if params.Language != nil {
		upload.Language = *params.Language
	}
	if params.Error != nil {
		upload.Error = params.Error
	}
	if params.Attempts != nil {
		upload.Attempts = *params.Attempts
	}
	if params.ClaimedAt != nil {
		upload.ClaimedAt = params.ClaimedAt
	}
	if params.LeaseExpiresAt != nil {
		upload.LeaseExpiresAt = params.LeaseExpiresAt
	}
	if params.WorkerID != nil {
		upload.WorkerID = params.WorkerID
	}
	if params.ClearClaim {
		upload.ClaimedAt = nil
		upload.LeaseExpiresAt = nil
		upload.WorkerID = nil
	}
	upload.UpdatedAt = time.Now().UTC()
	s.uploads[id] = upload
	return cloneUpload(upload), nil
}

func (s *InMemoryStore) ClaimNextUploadJob(_ context.Context, workerID string, leaseDuration time.Duration) (*models.Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	var selected *models.Upload
	for _, upload := range s.uploads {
		if upload.Status != "pending" {
			continue
		}
		candidate := upload
		if selected == nil || candidate.CreatedAt.Before(selected.CreatedAt) {
			selected = &candidate
		}
	}
	if selected == nil {
		return nil, nil
	}

	claimedAt := now
	leaseExpiresAt := now.Add(leaseDuration)
	upload := s.uploads[selected.ID]
	upload.Status = "processing"
	upload.Attempts++
	upload.ClaimedAt = &claimedAt
	upload.LeaseExpiresAt = &leaseExpiresAt
	upload.WorkerID = &workerID
	emptyErr := ""
	upload.Error = &emptyErr
	upload.UpdatedAt = now
	if inputBlob, ok := s.blobs[upload.InputBlobID]; ok {
		upload.InputBlob = *cloneBlob(inputBlob)
	}
	if upload.OutputBlobID != nil {
		if outputBlob, ok := s.blobs[*upload.OutputBlobID]; ok {
			upload.OutputBlob = cloneBlob(outputBlob)
		}
	}
	s.uploads[upload.ID] = upload
	return cloneUpload(upload), nil
}

func (s *InMemoryStore) ReconcileUploadJobs(_ context.Context, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now = now.UTC()
	for id, upload := range s.uploads {
		if inputBlob, ok := s.blobs[upload.InputBlobID]; ok {
			upload.InputBlob = *cloneBlob(inputBlob)
			if inputBlob.FileType == "markdown" {
				upload.OutputBlobID = &upload.InputBlobID
				upload.Status = "completed"
				upload.ClaimedAt = nil
				upload.LeaseExpiresAt = nil
				upload.WorkerID = nil
				emptyErr := ""
				upload.Error = &emptyErr
			}
		}
		if upload.OutputBlobID != nil {
			if outputBlob, ok := s.blobs[*upload.OutputBlobID]; ok {
				upload.OutputBlob = cloneBlob(outputBlob)
				upload.Status = "completed"
				upload.ClaimedAt = nil
				upload.LeaseExpiresAt = nil
				upload.WorkerID = nil
				emptyErr := ""
				upload.Error = &emptyErr
			}
		}
		if upload.OutputBlobID == nil && upload.Status == "processing" && (upload.LeaseExpiresAt == nil || !upload.LeaseExpiresAt.After(now)) {
			upload.Status = "pending"
			upload.ClaimedAt = nil
			upload.LeaseExpiresAt = nil
			upload.WorkerID = nil
		}
		if upload.OutputBlobID == nil && upload.Status == "completed" {
			upload.Status = "pending"
			upload.ClaimedAt = nil
			upload.LeaseExpiresAt = nil
			upload.WorkerID = nil
		}
		upload.UpdatedAt = now
		s.uploads[id] = upload
	}
	return nil
}

func (s *InMemoryStore) DeleteUploadByID(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	upload, ok := s.uploads[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	delete(s.uploads, id)
	s.deleteBlobIfUnreferenced(upload.InputBlobID)
	if upload.OutputBlobID != nil {
		s.deleteBlobIfUnreferenced(*upload.OutputBlobID)
	}
	return nil
}

func (s *InMemoryStore) CreateAuditLog(_ context.Context, entry *models.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auditLogCounter++
	entry.ID = s.auditLogCounter
	entry.Timedate = entry.Timedate.UTC()
	entry.CreatedAt = time.Now().UTC()
	s.auditLogs = append(s.auditLogs, *entry)
	return nil
}

func (s *InMemoryStore) CreateQuerySession(_ context.Context, params models.CreateQuerySessionParams) (*models.QuerySession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	session := models.QuerySession{
		ID:         uuid.NewString(),
		UserID:     params.UserID,
		Query:      params.Query,
		Mode:       params.Mode,
		Response:   params.Response,
		References: params.References,
		CreatedAt:  now,
	}
	s.querySessions[session.ID] = session
	return cloneQuerySession(session), nil
}

func (s *InMemoryStore) GetQuerySessionByID(_ context.Context, id string) (*models.QuerySession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.querySessions[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return cloneQuerySession(session), nil
}

func (s *InMemoryStore) ListQuerySessions(_ context.Context, userID string, params models.ListQuerySessionsParams) ([]models.QuerySession, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]models.QuerySession, 0)
	for _, session := range s.querySessions {
		if session.UserID == userID {
			filtered = append(filtered, session)
		}
	}

	slices.SortFunc(filtered, func(a, b models.QuerySession) int {
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
	total := int64(len(filtered))
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return []models.QuerySession{}, total, nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	items := make([]models.QuerySession, 0, end-start)
	for _, session := range filtered[start:end] {
		items = append(items, *cloneQuerySession(session))
	}
	return items, total, nil
}

func cloneUser(user models.User) *models.User {
	clone := user
	clone.AvatarData = append([]byte(nil), user.AvatarData...)
	return &clone
}

func cloneSession(session models.Session) *models.Session {
	clone := session
	return &clone
}

func cloneBlob(blob models.Blob) *models.Blob {
	clone := blob
	clone.Content = append([]byte(nil), blob.Content...)
	return &clone
}

func cloneUpload(upload models.Upload) *models.Upload {
	clone := upload
	clone.InputBlob = *cloneBlob(upload.InputBlob)
	if upload.OutputBlob != nil {
		clone.OutputBlob = cloneBlob(*upload.OutputBlob)
	}
	return &clone
}

func cloneQuerySession(session models.QuerySession) *models.QuerySession {
	clone := session
	return &clone
}

func (s *InMemoryStore) deleteBlobIfUnreferenced(blobID string) {
	for _, upload := range s.uploads {
		if upload.InputBlobID == blobID {
			return
		}
		if upload.OutputBlobID != nil && *upload.OutputBlobID == blobID {
			return
		}
	}
	delete(s.blobs, blobID)
}
