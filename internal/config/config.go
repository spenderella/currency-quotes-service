package config

import (
	"errors"
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

const configPath = ".env"

type Configuration struct {
	HTTPServer HTTPServerConfig
	Postgres   PostgresConfig
}

type HTTPServerConfig struct {
	Address             string `env:"HTTP_SERVER_ADDRESS"`
	ReadTimeoutSeconds  uint   `env:"HTTP_SERVER_READ_TIMEOUT" envDefault:"30"`
	WriteTimeoutSeconds uint   `env:"HTTP_SERVER_WRITE_TIMEOUT" envDefault:"30"`
}

type PostgresConfig struct {
	Host     string `env:"DB_HOST"`
	Port     string `env:"DB_PORT"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	DBName   string `env:"DB_NAME"`
	SSLMode  string `env:"DB_SSLMODE"`
}

func New() (*Configuration, error) {
	return LoadAndParseConfig(configPath)
}

func LoadAndParseConfig(path string) (*Configuration, error) {
	if err := loadEnvFile(path); err != nil {
		return nil, err
	}

	cfg := &Configuration{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func loadEnvFile(path string) error {
	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("load env file %s: %w", path, err)
	}
	return nil
}

func (c *Configuration) validate() error {
	var errs []error

	if c.HTTPServer.Address == "" {
		errs = append(errs, errors.New("token: HTTP_SERVER_ADDRESS is required"))
	}
	if c.Postgres.Host == "" {
		errs = append(errs, errors.New("postgres: DB_HOST is required"))
	}
	if c.Postgres.Port == "" {
		errs = append(errs, errors.New("postgres: DB_PORT is required"))
	}
	if c.Postgres.User == "" {
		errs = append(errs, errors.New("postgres: DB_USER is required"))
	}
	if c.Postgres.Password == "" {
		errs = append(errs, errors.New("postgres: DB_PASSWORD is required"))
	}
	if c.Postgres.DBName == "" {
		errs = append(errs, errors.New("postgres: DB_NAME is required"))
	}
	if c.Postgres.SSLMode == "" {
		errs = append(errs, errors.New("postgres: DB_SSLMODE is required"))
	}

	return errors.Join(errs...)
}
