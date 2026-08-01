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
	//maxRetries int
	//baseDelay  time.Duration
}

func NewProvider(httpClient *http.Client, baseURL, providers string, maxRetries int, baseDelay time.Duration) *Provider {
	return &Provider{
		httpClient: httpClient,
		baseURL:    baseURL,
		providers:  providers,
	}
}

func (p *Provider) GetRate(ctx context.Context, baseCurrency, quoteCurrency string) (decimal.Decimal, time.Time, error)
