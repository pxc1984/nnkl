package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pxc1984/nnkl-backend/utils"
)

type Store interface {
	Backend() string
	Ping(context.Context) error
	Close() error
	CountUsers(context.Context) (int64, error)
	CreateUser(context.Context, CreateUserParams) (*User, error)
	GetUserByEmail(context.Context, string) (*User, error)
	GetUserByID(context.Context, string) (*User, error)
	UpdateUserLastLogin(context.Context, string, time.Time) error
	CreateSession(context.Context, CreateSessionParams) (*Session, error)
	GetSessionByID(context.Context, string) (*Session, error)
	GetSessionByRefreshTokenHash(context.Context, string) (*Session, error)
	UpdateSessionToken(context.Context, UpdateSessionTokenParams) (*Session, error)
	TouchSession(context.Context, string, time.Time) error
	ListUserSessions(context.Context, string) ([]Session, error)
	DeleteSessionByID(context.Context, string) error
	DeleteSessionByUserAndHash(context.Context, string, string) error
	DeleteUserSessions(context.Context, string) error
	CreateBlob(context.Context, CreateBlobParams) (*Blob, error)
	GetBlobByID(context.Context, string) (*Blob, error)
	GetBlobBySHA256(context.Context, string) (*Blob, error)
	CreateUpload(context.Context, CreateUploadParams) (*Upload, error)
	ListUploads(context.Context, ListUploadsParams) ([]Upload, int64, error)
	GetUploadByID(context.Context, string) (*Upload, error)
	UpdateUpload(context.Context, string, UpdateUploadParams) (*Upload, error)
	DeleteUploadByID(context.Context, string) error
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
