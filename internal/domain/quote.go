package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Quote struct {
	ID            uuid.UUID
	BaseCurrency  string
	QuoteCurrency string
	Rate          decimal.Decimal
	Status        string
	ProviderTime  time.Time
	FetchedAt     time.Time
}
