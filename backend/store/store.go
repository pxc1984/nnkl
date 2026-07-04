package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pxc1984/nnkl-backend/store/models"
	"github.com/pxc1984/nnkl-backend/utils"
)

type Store interface {
	Backend() string
	Ping(context.Context) error
	Close() error
	CountUsers(context.Context) (int64, error)
	CreateUser(context.Context, models.CreateUserParams) (*models.User, error)
	GetUserByEmail(context.Context, string) (*models.User, error)
	GetUserByID(context.Context, string) (*models.User, error)
	UpdateUserLastLogin(context.Context, string, time.Time) error
	CreateSession(context.Context, models.CreateSessionParams) (*models.Session, error)
	GetSessionByID(context.Context, string) (*models.Session, error)
	GetSessionByRefreshTokenHash(context.Context, string) (*models.Session, error)
	UpdateSessionToken(context.Context, models.UpdateSessionTokenParams) (*models.Session, error)
	TouchSession(context.Context, string, time.Time) error
	ListUserSessions(context.Context, string) ([]models.Session, error)
	DeleteSessionByID(context.Context, string) error
	DeleteSessionByUserAndHash(context.Context, string, string) error
	DeleteUserSessions(context.Context, string) error
	CreateBlob(context.Context, models.CreateBlobParams) (*models.Blob, error)
	GetBlobByID(context.Context, string) (*models.Blob, error)
	GetBlobBySHA256(context.Context, string) (*models.Blob, error)
	CreateUpload(context.Context, models.CreateUploadParams) (*models.Upload, error)
	ListUploads(context.Context, models.ListUploadsParams) ([]models.Upload, int64, error)
	GetUploadByID(context.Context, string) (*models.Upload, error)
	UpdateUpload(context.Context, string, models.UpdateUploadParams) (*models.Upload, error)
	DeleteUploadByID(context.Context, string) error
	CreateAuditLog(context.Context, *models.AuditLog) error
	CreateQuerySession(context.Context, models.CreateQuerySessionParams) (*models.QuerySession, error)
	GetQuerySessionByID(context.Context, string) (*models.QuerySession, error)
	ListQuerySessions(context.Context, string, models.ListQuerySessionsParams) ([]models.QuerySession, int64, error)
}

var globalStore Store

func InitStore() (Store, error) {
	backend := getStoreType(strings.ToLower(utils.Settings.StoreBackend))
	return initStore(backend)
}

func initStore(backend Backend) (Store, error) {
	switch backend {
	case InMemory:
		globalStore = NewInMemoryStore()
		return globalStore, nil
	case Postgres:
		var err error
		globalStore, err = NewPostgresStore()
		return globalStore, err
	default:
		return nil, fmt.Errorf("unsupported STORE_BACKEND %q", utils.Settings.StoreBackend)
	}
}

type Backend string

const (
	InMemory  Backend = "memory"
	Postgres          = "postgres"
	Undefined         = "undef"
)

func getStoreType(backend string) Backend {
	switch backend {
	case "memory":
		return InMemory
	case "postgres":
		return Postgres
	default:
		return Undefined
	}
}

func GetStore() Store {
	return globalStore
}
