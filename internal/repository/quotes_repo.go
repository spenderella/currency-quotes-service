package repository

import (
	"context"
	"database/sql"

	//"fmt"
	//"strings"
	//"github.com/spenderella/currency-quotes-service/internal/domain"
	"github.com/google/uuid"
)

type QuoteRepository struct {
	db *sql.DB
}

func NewQuoteRepository(db *sql.DB) *QuoteRepository {
	return &QuoteRepository{db: db}
}

func (r *QuoteRepository) CreateQuoteUpdate(ctx context.Context, baseCurrency string, quoteCurrency string, idempotencyKey string) (id uuid.UUID, err error) {
	query := `
        INSERT INTO quote_updates (base_currency, quote_currency, idempotency_key)
		VALUES ($1, $2, $3)
		ON CONFLICT (idempotency_key) DO UPDATE
    		SET idempotency_key = EXCLUDED.idempotency_key
		RETURNING id
    `

	err = r.db.QueryRowContext(ctx, query, baseCurrency, quoteCurrency, idempotencyKey).Scan(&id)

	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}
