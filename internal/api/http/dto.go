package http

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateQuoteRequest struct {
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
}

type CreateQuoteResponse struct {
	ID uuid.UUID `json:"id"`
}

type QuoteResponse struct {
	ID            uuid.UUID       `json:"id"`
	BaseCurrency  string          `json:"base_currency"`
	QuoteCurrency string          `json:"quote_currency"`
	Status        string          `json:"status"`
	Rate          decimal.Decimal `json:"rate,omitzero"`
	ProviderTime  time.Time       `json:"provider_time,omitzero"`
	FetchedAt     time.Time       `json:"fetched_at,omitzero"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
