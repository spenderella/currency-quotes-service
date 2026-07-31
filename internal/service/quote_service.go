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

type IQuoteRepository interface {
	CreateQuoteUpdate(ctx context.Context, baseCurrency string, quoteCurrency string, idempotencyKey string) (id uuid.UUID, err error)
	GetQuoteUpdateByID(ctx context.Context, id uuid.UUID) (domain.Quote, error)
	GetQuoteUpdateLatest(ctx context.Context, baseCurrency string, quoteCurrency string) (domain.Quote, error)
	ClaimPendingQuoteUpdates(ctx context.Context, limit int) ([]domain.Quote, error)
	GetUndoneQuoteUpdates(ctx context.Context) ([]domain.Quote, error)
	MarkDone(ctx context.Context, id uuid.UUID, rate decimal.Decimal, fetchedAt time.Time) error
	MarkFailed(ctx context.Context, id uuid.UUID, reason string) error
}

type ICurrencyService interface {
	IsSupported(code string) bool
}

type QuoteService struct {
	quoteRepo       IQuoteRepository
	currencyService ICurrencyService
}

func NewQuoteService(quoteRepo IQuoteRepository, currencyService ICurrencyService) *QuoteService {
	return &QuoteService{
		quoteRepo:       quoteRepo,
		currencyService: currencyService,
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

func (s *QuoteService) RunWorkerPool(ctx context.Context, poolSize int) error {
	panic("not implemented")
}

// RecoverPendingUpdates returns tasks left pending/in_progress after a crash,
// for the caller to feed into the worker pool.
func (s *QuoteService) RecoverPendingUpdates(ctx context.Context) ([]domain.Quote, error) {
	return s.quoteRepo.GetUndoneQuoteUpdates(ctx)
}
