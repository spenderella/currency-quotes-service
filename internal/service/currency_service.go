package service

import (
	"context"
	"fmt"
)

//go:generate mockgen -source=currency_service.go -destination=mocks/mock_currency_repository.go -package=mocks

type ICurrencyRepository interface {
	GetEnabledCurrencies(ctx context.Context) ([]string, error)
}

type CurrencyService struct {
	repo       ICurrencyRepository
	currencies map[string]struct{}
}

func NewCurrencyService(repo ICurrencyRepository) *CurrencyService {
	return &CurrencyService{repo: repo}
}

// LoadCurrencies pulls the enabled currency whitelist from the repository
// into memory as a set. Meant to be called once at startup.
func (s *CurrencyService) LoadCurrencies(ctx context.Context) error {
	codes, err := s.repo.GetEnabledCurrencies(ctx)
	if err != nil {
		return fmt.Errorf("service: load currencies: %w", err)
	}

	currencies := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		currencies[code] = struct{}{}
	}
	s.currencies = currencies

	return nil
}

// IsSupported reports whether a currency code is in the enabled whitelist.
func (s *CurrencyService) IsSupported(code string) bool {
	_, ok := s.currencies[code]
	return ok
}
