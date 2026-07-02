package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/pxc1984/backend-rest-go/utils"
)

type Store interface {
	Backend() string
	Ping(context.Context) error
	Close() error
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
