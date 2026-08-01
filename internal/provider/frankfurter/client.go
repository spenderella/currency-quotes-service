package frankfurter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/shopspring/decimal"
)

type response struct {
	rate         decimal.Decimal
	providerTime time.Time
}

// rateRecord mirrors one entry of the JSON array returned by the Frankfurter API.
type rateRecord struct {
	Date  string          `json:"date"`
	Base  string          `json:"base"`
	Quote string          `json:"quote"`
	Rate  decimal.Decimal `json:"rate"`
}

func (p *Provider) fetchRates(ctx context.Context, baseCurrency, quoteCurrency string) (*response, error) {
	base, err := url.Parse(p.baseURL)
	if err != nil {
		return nil, err
	}
	u := base.JoinPath("rates")
	q := u.Query()
	q.Set("base", baseCurrency)
	q.Set("quotes", quoteCurrency)
	q.Set("providers", p.providers)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &retryableError{err}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &retryableError{fmt.Errorf("frankfurter: read response: %w", err)}
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("frankfurter: unexpected status %d: %s", resp.StatusCode, data)
		if resp.StatusCode >= http.StatusInternalServerError {
			return nil, &retryableError{err}
		}
		return nil, err
	}

	var body []rateRecord
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("frankfurter: decode response: %w", err)
	}

	idx := slices.IndexFunc(body, func(r rateRecord) bool { return r.Quote == quoteCurrency })
	if idx == -1 {
		return nil, fmt.Errorf("frankfurter: rate for %s not found in response", quoteCurrency)
	}
	record := body[idx]

	providerTime, err := time.Parse("2006-01-02", record.Date)
	if err != nil {
		return nil, fmt.Errorf("frankfurter: parse date: %w", err)
	}

	return &response{rate: record.Rate, providerTime: providerTime}, nil
}
