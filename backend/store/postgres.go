package store

import (
	"context"
	"database/sql"
	"fmt"

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
