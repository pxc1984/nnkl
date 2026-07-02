package utils

import (
	"github.com/kelseyhightower/envconfig"
)

type settings struct {
	Host             string `envconfig:"HOST" default:"0.0.0.0"`
	Port             int    `envconfig:"PORT" default:"8080"`
	LogLevel         string `envconfig:"LOG_LEVEL" default:"INFO"`
	StoreBackend     string `envconfig:"STORE_BACKEND" default:"memory"`
	PostgresHost     string `envconfig:"POSTGRES_HOST" default:"db"`
	PostgresPort     int    `envconfig:"POSTGRES_PORT" default:"5432"`
	PostgresUser     string `envconfig:"POSTGRES_USER" default:"admin"`
	PostgresPassword string `envconfig:"POSTGRES_PASSWORD" default:"admin"`
	PostgresDB       string `envconfig:"POSTGRES_DB" default:"db"`
	PostgresSSLMode  string `envconfig:"POSTGRES_SSLMODE" default:"disable"`
}

var Settings settings

func InitSettings() {
	if err := envconfig.Process("", &Settings); err != nil {
		panic("failed to load utils from env: " + err.Error())
	}
}
