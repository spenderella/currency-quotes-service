package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

var ErrNotFound = errors.New("repository: quote update not found")

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
		return uuid.Nil, fmt.Errorf("repository: create quoteUpdate: %w", err)
	}

	return id, nil
}

// quoteUpdateRow mirrors the nullable shape of the quote_updates table.
type quoteUpdateRow struct {
	id            uuid.UUID
	baseCurrency  string
	quoteCurrency string
	rate          decimal.NullDecimal
	status        string
	fetchedAt     sql.Null[time.Time]
}

func (row quoteUpdateRow) toDomain() domain.Quote {
	quote := domain.Quote{
		ID:            row.id,
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
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Quote{}, ErrNotFound
		}
		return domain.Quote{}, fmt.Errorf("repository: get quote update by id: %w", err)
	}
	row.id = id
	return row.toDomain(), nil
}

func (r *QuoteRepository) GetQuoteUpdateLatest(ctx context.Context, baseCurrency string, quoteCurrency string) (domain.Quote, error) {
	query := `
        SELECT
		id,
		base_currency,
		quote_currency,
		rate,
		status,
		fetched_at
		FROM quote_updates
		WHERE base_currency = $1 AND quote_currency = $2 AND status = 'done'
		ORDER BY fetched_at DESC
		LIMIT 1
    `

	var row quoteUpdateRow
	err := r.db.QueryRowContext(ctx, query, baseCurrency, quoteCurrency).Scan(
		&row.id,
		&row.baseCurrency,
		&row.quoteCurrency,
		&row.rate,
		&row.status,
		&row.fetchedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Quote{}, ErrNotFound
		}
		return domain.Quote{}, fmt.Errorf("repository: get quote update by id: %w", err)
	}

	return row.toDomain(), nil
}

func (r *QuoteRepository) MarkInProgress(ctx context.Context, id uuid.UUID) error {
	query := `
        UPDATE quote_updates
        SET status = 'in_progress', updated_at = now()
        WHERE id = $1
    `
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repository: mark in progress: %w", err)
	}
	return checkRowsAffected(res, "mark in progress")
}

func (r *QuoteRepository) MarkDone(ctx context.Context, id uuid.UUID, rate decimal.Decimal, fetchedAt time.Time) error {
	query := `
        UPDATE quote_updates
        SET status = 'done', rate = $1, fetched_at = $2, updated_at = now()
        WHERE id = $3
    `
	res, err := r.db.ExecContext(ctx, query, rate, fetchedAt, id)
	if err != nil {
		return fmt.Errorf("repository: mark done: %w", err)
	}
	return checkRowsAffected(res, "mark done")
}

func (r *QuoteRepository) MarkFailed(ctx context.Context, id uuid.UUID, reason string) error {
	query := `
        UPDATE quote_updates
        SET status = 'failed', error = $1, updated_at = now()
        WHERE id = $2
    `
	res, err := r.db.ExecContext(ctx, query, reason, id)
	if err != nil {
		return fmt.Errorf("repository: mark failed: %w", err)
	}
	return checkRowsAffected(res, "mark failed")
}

func checkRowsAffected(res sql.Result, op string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository: %s: %w", op, err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *QuoteRepository) GetPendingQuoteUpdates(ctx context.Context) ([]domain.Quote, error) {
	query := `
        SELECT
		id,
		base_currency,
		quote_currency
		FROM quote_updates
		WHERE status = 'pending' OR status = 'in_progress'
		ORDER BY created_at ASC
    `
	updates := []domain.Quote{}

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository: get pending quote updates: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var quote domain.Quote
		err := rows.Scan(
			&quote.ID,
			&quote.BaseCurrency,
			&quote.QuoteCurrency,
		)
		if err != nil {
			return nil, fmt.Errorf("repository: get pending quote update: %w", err)
		}
		updates = append(updates, quote)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("repository: get pending quote updates: %w", err)
	}

	return updates, nil
}
