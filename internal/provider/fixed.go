package provider

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// Fixed is a temporary RatesProvider stand-in that always returns the same
// rate. Lets the worker pool be wired and tested end-to-end before the real
// frankfurter.dev client exists; swapped out without touching worker/service code.
type Fixed struct {
	Rate decimal.Decimal
}

func (p Fixed) GetRate(ctx context.Context, baseCurrency, quoteCurrency string) (decimal.Decimal, time.Time, error) {
	return p.Rate, time.Now().UTC(), nil
}
