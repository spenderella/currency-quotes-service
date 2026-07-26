package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/spenderella/currency-quotes-service/internal/domain"
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

// quoteUpdateRow mirrors the nullable shape of the quote_updates table.
// rate/fetched_at are NULL until status reaches "done", which domain.Quote's
// non-nullable fields can't represent directly during a scan.
type quoteUpdateRow struct {
	baseCurrency  string
	quoteCurrency string
	rate          decimal.NullDecimal
	status        string
	fetchedAt     sql.Null[time.Time]
}

func (row quoteUpdateRow) toDomain(id uuid.UUID) domain.Quote {
	quote := domain.Quote{
		ID:            id,
		BaseCurrency:  row.baseCurrency,
		QuoteCurrency: row.quoteCurrency,
		Status:        row.status,
	}
	if row.rate.Valid {
		quote.Rate = row.rate.Decimal
	}
	if row.fetchedAt.Valid {
		quote.FetchedAt = row.fetchedAt.V
	}
	return quote
}

func (r *QuoteRepository) GetQuoteUpdateByID(ctx context.Context, id uuid.UUID) (domain.Quote, error) {
	query := `
        SELECT
		base_currency,
		quote_currency,
		rate,
		status,
		fetched_at
		FROM quote_updates
		WHERE id = $1
    `

	var row quoteUpdateRow
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&row.baseCurrency,
		&row.quoteCurrency,
		&row.rate,
		&row.status,
		&row.fetchedAt,
	)
	if err != nil {
		return domain.Quote{}, err
	}

	return row.toDomain(id), nil
}
