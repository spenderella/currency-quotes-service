package frankfurter

import (
	"context"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

type Provider struct {
	httpClient *http.Client
	baseURL    string
	providers  string
	maxRetries int
	baseDelay  time.Duration
}

func NewProvider(httpClient *http.Client, baseURL, providers string, maxRetries int, baseDelay time.Duration) *Provider {
	return &Provider{
		httpClient: httpClient,
		baseURL:    baseURL,
		providers:  providers,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

func (p *Provider) GetRate(ctx context.Context, baseCurrency, quoteCurrency string) (decimal.Decimal, time.Time, error) {
	var resp *response
	err := withRetry(ctx, p.maxRetries, p.baseDelay, func() error {
		r, err := p.fetchRates(ctx, baseCurrency, quoteCurrency)
		if err != nil {
			return err
		}
		resp = r
		return nil
	})
	if err != nil {
		return decimal.Decimal{}, time.Time{}, err
	}
	return resp.rate, resp.providerTime, nil
}
