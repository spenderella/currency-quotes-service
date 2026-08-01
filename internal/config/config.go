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
	Worker     WorkerConfig
}

type HTTPServerConfig struct {
	Address             string `env:"HTTP_SERVER_ADDRESS"`
	ReadTimeoutSeconds  uint   `env:"HTTP_SERVER_READ_TIMEOUT" envDefault:"30"`
	WriteTimeoutSeconds uint   `env:"HTTP_SERVER_WRITE_TIMEOUT" envDefault:"30"`
}

type WorkerConfig struct {
	PoolSize            int  `env:"WORKER_POOL_SIZE" envDefault:"4"`
	TickIntervalSeconds uint `env:"WORKER_TICK_INTERVAL_SECONDS" envDefault:"5"`
	TaskTimeoutSeconds  uint `env:"WORKER_TASK_TIMEOUT_SECONDS" envDefault:"10"`
}

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST"`
	Port     string `env:"POSTGRES_PORT"`
	User     string `env:"POSTGRES_USER"`
	Password string `env:"POSTGRES_PASSWORD"`
	DBName   string `env:"POSTGRES_NAME"`
	SSLMode  string `env:"POSTGRES_SSLMODE"`
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
		errs = append(errs, errors.New("postgres: POSTGRES_HOST is required"))
	}
	if c.Postgres.Port == "" {
		errs = append(errs, errors.New("postgres: POSTGRES_PORT is required"))
	}
	if c.Postgres.User == "" {
		errs = append(errs, errors.New("postgres: POSTGRES_USER is required"))
	}
	if c.Postgres.Password == "" {
		errs = append(errs, errors.New("postgres: POSTGRES_PASSWORD is required"))
	}
	if c.Postgres.DBName == "" {
		errs = append(errs, errors.New("postgres: POSTGRES_NAME is required"))
	}
	if c.Postgres.SSLMode == "" {
		errs = append(errs, errors.New("postgres: POSTGRES_SSLMODE is required"))
	}
	if c.Worker.PoolSize <= 0 {
		errs = append(errs, errors.New("worker: WORKER_POOL_SIZE must be > 0"))
	}
	if c.Worker.TickIntervalSeconds == 0 {
		errs = append(errs, errors.New("worker: WORKER_TICK_INTERVAL_SECONDS must be > 0"))
	}
	if c.Worker.TaskTimeoutSeconds == 0 {
		errs = append(errs, errors.New("worker: WORKER_TASK_TIMEOUT_SECONDS must be > 0"))
	}

	return errors.Join(errs...)
}
