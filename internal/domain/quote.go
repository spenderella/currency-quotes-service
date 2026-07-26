package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/ydb-platform/ydb-go-sdk/v3/pkg/decimal"
)

type Quote struct {
	ID            uuid.UUID
	BaseCurrency  string
	QuoteCurrency string
	Rate          decimal.Decimal
	FetchedAt     time.Time
}
