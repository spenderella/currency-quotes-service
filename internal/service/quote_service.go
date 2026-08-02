package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/spenderella/currency-quotes-service/internal/domain"
	"github.com/spenderella/currency-quotes-service/internal/repository"
)

// ErrUnsupportedCurrency means a currency code is not in the enabled whitelist.
var ErrUnsupportedCurrency = errors.New("service: unsupported currency")
var ErrNotFound = errors.New("service: quote update not found")

//go:generate mockgen -source=quote_service.go -destination=mocks/mock_quote_service.go -package=mocks

type IQuoteRepository interface {
	CreateQuoteUpdate(ctx context.Context, baseCurrency string, quoteCurrency string, idempotencyKey string) (id uuid.UUID, err error)
	GetQuoteUpdateByID(ctx context.Context, id uuid.UUID) (domain.Quote, error)
	GetQuoteUpdateLatest(ctx context.Context, baseCurrency string, quoteCurrency string) (domain.Quote, error)
	ClaimPendingQuoteUpdates(ctx context.Context, limit int) ([]domain.Quote, error)
	GetUndoneQuoteUpdates(ctx context.Context) ([]domain.Quote, error)
	MarkDone(ctx context.Context, id uuid.UUID, rate decimal.Decimal, providerTime, fetchedAt time.Time) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string) error
}

type ICurrencyService interface {
	IsSupported(code string) bool
}

// IRatesProvider fetches the current rate for a currency pair from an external source.
type IRatesProvider interface {
	GetRate(ctx context.Context, baseCurrency, quoteCurrency string) (rate decimal.Decimal, providerTime time.Time, err error)
}

type QuoteService struct {
	quoteRepo       IQuoteRepository
	currencyService ICurrencyService
	provider        IRatesProvider
}

func NewQuoteService(quoteRepo IQuoteRepository, currencyService ICurrencyService, provider IRatesProvider) *QuoteService {
	return &QuoteService{
		quoteRepo:       quoteRepo,
		currencyService: currencyService,
		provider:        provider,
	}
}

func (s *QuoteService) validatePair(baseCurrency, quoteCurrency string) error {
	var errs []error
	if !s.currencyService.IsSupported(baseCurrency) {
		errs = append(errs, fmt.Errorf("%w: %s", ErrUnsupportedCurrency, baseCurrency))
	}
	if !s.currencyService.IsSupported(quoteCurrency) {
		errs = append(errs, fmt.Errorf("%w: %s", ErrUnsupportedCurrency, quoteCurrency))
	}
	return errors.Join(errs...)
}

func (s *QuoteService) CreateQuoteUpdate(ctx context.Context, baseCurrency string, quoteCurrency string, idempotencyKey string) (id uuid.UUID, err error) {
	if err := s.validatePair(baseCurrency, quoteCurrency); err != nil {
		return uuid.Nil, err
	}
	return s.quoteRepo.CreateQuoteUpdate(ctx, baseCurrency, quoteCurrency, idempotencyKey)
}

func (s *QuoteService) GetQuoteUpdateByID(ctx context.Context, id uuid.UUID) (domain.Quote, error) {
	quote, err := s.quoteRepo.GetQuoteUpdateByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.Quote{}, ErrNotFound
		}
		return domain.Quote{}, err
	}
	return quote, nil
}

func (s *QuoteService) GetQuoteUpdateLatest(ctx context.Context, baseCurrency string, quoteCurrency string) (domain.Quote, error) {
	if err := s.validatePair(baseCurrency, quoteCurrency); err != nil {
		return domain.Quote{}, err
	}

	quote, err := s.quoteRepo.GetQuoteUpdateLatest(ctx, baseCurrency, quoteCurrency)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return domain.Quote{}, ErrNotFound
		}
		return domain.Quote{}, err
	}
	return quote, nil
}

func (s *QuoteService) ClaimPendingQuoteUpdates(ctx context.Context, limit int) ([]domain.Quote, error) {
	return s.quoteRepo.ClaimPendingQuoteUpdates(ctx, limit)
}

// RecoverPendingUpdates returns tasks left pending/in_progress after a crash,
// for the caller to feed into the worker pool.
func (s *QuoteService) RecoverPendingUpdates(ctx context.Context) ([]domain.Quote, error) {
	return s.quoteRepo.GetUndoneQuoteUpdates(ctx)
}

// ProcessClaimedTask resolves an already-claimed task: fetches the rate from
// the provider and persists the result.
func (s *QuoteService) ProcessClaimedTask(ctx context.Context, task domain.Quote) error {
	rate, providerTime, err := s.provider.GetRate(ctx, task.BaseCurrency, task.QuoteCurrency)
	if err != nil {
		if markErr := s.quoteRepo.MarkFailed(ctx, task.ID, err.Error()); markErr != nil {
			return errors.Join(fmt.Errorf("get rate: %w", err), markErr)
		}
		return fmt.Errorf("get rate: %w", err)
	}
	return s.quoteRepo.MarkDone(ctx, task.ID, rate, providerTime, time.Now().UTC())
}
