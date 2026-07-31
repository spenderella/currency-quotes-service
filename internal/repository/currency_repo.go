package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type CurrencyRepository struct {
	db *sql.DB
}

func NewCurrencyRepository(db *sql.DB) *CurrencyRepository {
	return &CurrencyRepository{db: db}
}

func (r *CurrencyRepository) GetEnabledCurrencies(ctx context.Context) ([]string, error) {
	query := `
        SELECT code
        FROM currencies
        WHERE enabled = true
    `

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository: get enabled currencies: %w", err)
	}
	defer rows.Close()

	codes := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("repository: get enabled currency: %w", err)
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: get enabled currencies: %w", err)
	}

	return codes, nil
}
