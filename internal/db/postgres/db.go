package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/spenderella/currency-quotes-service/internal/config"

	_ "github.com/lib/pq"
)

// Connect establishes a connection to the PostgreSQL database using the given configuration.
// It configures connection pool settings and verifies the connection with a ping.
func Connect(cfg config.PostgresConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
