package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/spenderella/currency-quotes-service/internal/domain"
	"github.com/spenderella/currency-quotes-service/internal/repository"
	"github.com/spenderella/currency-quotes-service/internal/service"
	"github.com/spenderella/currency-quotes-service/internal/service/mocks"
)

// newQuoteService wires a QuoteService with fresh mocks for each test and
// returns them so the caller can set expectations.
func newQuoteService(t *testing.T) (*service.QuoteService, *mocks.MockIQuoteRepository, *mocks.MockICurrencyService, *mocks.MockIRatesProvider) {
	t.Helper()
	ctrl := gomock.NewController(t)
	quoteRepo := mocks.NewMockIQuoteRepository(ctrl)
	currencyService := mocks.NewMockICurrencyService(ctrl)
	provider := mocks.NewMockIRatesProvider(ctrl)

	svc := service.NewQuoteService(quoteRepo, currencyService, provider)
	return svc, quoteRepo, currencyService, provider
}

func TestQuoteService_CreateQuoteUpdate_Success(t *testing.T) {
	svc, quoteRepo, currencyService, _ := newQuoteService(t)
	ctx := t.Context()
	wantID := uuid.New()

	currencyService.EXPECT().IsSupported("EUR").Return(true)
	currencyService.EXPECT().IsSupported("MXN").Return(true)
	quoteRepo.EXPECT().
		CreateQuoteUpdate(ctx, "EUR", "MXN", "idem-key").
		Return(wantID, nil)

	gotID, err := svc.CreateQuoteUpdate(ctx, "EUR", "MXN", "idem-key")
	require.NoError(t, err)
	assert.Equal(t, wantID, gotID)
}

func TestQuoteService_CreateQuoteUpdate_UnsupportedBase(t *testing.T) {
	svc, quoteRepo, currencyService, _ := newQuoteService(t)
	ctx := t.Context()

	currencyService.EXPECT().IsSupported("XXX").Return(false)
	currencyService.EXPECT().IsSupported("MXN").Return(true)
	quoteRepo.EXPECT().CreateQuoteUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := svc.CreateQuoteUpdate(ctx, "XXX", "MXN", "idem-key")
	require.ErrorIs(t, err, service.ErrUnsupportedCurrency)
	assert.ErrorContains(t, err, "XXX")
}

func TestQuoteService_CreateQuoteUpdate_BothUnsupported(t *testing.T) {
	svc, quoteRepo, currencyService, _ := newQuoteService(t)
	ctx := t.Context()

	currencyService.EXPECT().IsSupported("XXX").Return(false)
	currencyService.EXPECT().IsSupported("YYY").Return(false)
	quoteRepo.EXPECT().CreateQuoteUpdate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := svc.CreateQuoteUpdate(ctx, "XXX", "YYY", "idem-key")
	require.ErrorIs(t, err, service.ErrUnsupportedCurrency)
	assert.ErrorContains(t, err, "XXX")
	assert.ErrorContains(t, err, "YYY")
}

func TestQuoteService_CreateQuoteUpdate_RepoError(t *testing.T) {
	svc, quoteRepo, currencyService, _ := newQuoteService(t)
	ctx := t.Context()
	repoErr := errors.New("insert failed")

	currencyService.EXPECT().IsSupported("EUR").Return(true)
	currencyService.EXPECT().IsSupported("MXN").Return(true)
	quoteRepo.EXPECT().
		CreateQuoteUpdate(ctx, "EUR", "MXN", "idem-key").
		Return(uuid.Nil, repoErr)

	_, err := svc.CreateQuoteUpdate(ctx, "EUR", "MXN", "idem-key")
	require.ErrorIs(t, err, repoErr)
}

func TestQuoteService_GetQuoteUpdateByID_Success(t *testing.T) {
	svc, quoteRepo, _, _ := newQuoteService(t)
	ctx := t.Context()
	id := uuid.New()
	want := domain.Quote{ID: id, BaseCurrency: "EUR", QuoteCurrency: "MXN", Status: "done"}

	quoteRepo.EXPECT().GetQuoteUpdateByID(ctx, id).Return(want, nil)

	got, err := svc.GetQuoteUpdateByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestQuoteService_GetQuoteUpdateByID_NotFound(t *testing.T) {
	svc, quoteRepo, _, _ := newQuoteService(t)
	ctx := t.Context()
	id := uuid.New()

	quoteRepo.EXPECT().GetQuoteUpdateByID(ctx, id).Return(domain.Quote{}, repository.ErrNotFound)

	_, err := svc.GetQuoteUpdateByID(ctx, id)
	require.ErrorIs(t, err, service.ErrNotFound)
}

func TestQuoteService_GetQuoteUpdateByID_OtherRepoError(t *testing.T) {
	svc, quoteRepo, _, _ := newQuoteService(t)
	ctx := t.Context()
	id := uuid.New()
	repoErr := errors.New("connection reset")

	quoteRepo.EXPECT().GetQuoteUpdateByID(ctx, id).Return(domain.Quote{}, repoErr)

	_, err := svc.GetQuoteUpdateByID(ctx, id)
	require.ErrorIs(t, err, repoErr)
	assert.NotErrorIs(t, err, service.ErrNotFound, "unrelated repo errors must not be mapped to ErrNotFound")
}

func TestQuoteService_GetQuoteUpdateLatest_Success(t *testing.T) {
	svc, quoteRepo, currencyService, _ := newQuoteService(t)
	ctx := t.Context()
	want := domain.Quote{BaseCurrency: "EUR", QuoteCurrency: "MXN", Status: "done"}

	currencyService.EXPECT().IsSupported("EUR").Return(true)
	currencyService.EXPECT().IsSupported("MXN").Return(true)
	quoteRepo.EXPECT().GetQuoteUpdateLatest(ctx, "EUR", "MXN").Return(want, nil)

	got, err := svc.GetQuoteUpdateLatest(ctx, "EUR", "MXN")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestQuoteService_GetQuoteUpdateLatest_UnsupportedPair(t *testing.T) {
	svc, quoteRepo, currencyService, _ := newQuoteService(t)
	ctx := t.Context()

	currencyService.EXPECT().IsSupported("XXX").Return(false)
	currencyService.EXPECT().IsSupported("MXN").Return(true)
	quoteRepo.EXPECT().GetQuoteUpdateLatest(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, err := svc.GetQuoteUpdateLatest(ctx, "XXX", "MXN")
	require.ErrorIs(t, err, service.ErrUnsupportedCurrency)
}

func TestQuoteService_GetQuoteUpdateLatest_NotFound(t *testing.T) {
	svc, quoteRepo, currencyService, _ := newQuoteService(t)
	ctx := t.Context()

	currencyService.EXPECT().IsSupported("EUR").Return(true)
	currencyService.EXPECT().IsSupported("MXN").Return(true)
	quoteRepo.EXPECT().GetQuoteUpdateLatest(ctx, "EUR", "MXN").Return(domain.Quote{}, repository.ErrNotFound)

	_, err := svc.GetQuoteUpdateLatest(ctx, "EUR", "MXN")
	require.ErrorIs(t, err, service.ErrNotFound)
}

func TestQuoteService_ClaimPendingQuoteUpdates_Delegates(t *testing.T) {
	svc, quoteRepo, _, _ := newQuoteService(t)
	ctx := t.Context()
	want := []domain.Quote{{ID: uuid.New()}, {ID: uuid.New()}}

	quoteRepo.EXPECT().ClaimPendingQuoteUpdates(ctx, 5).Return(want, nil)

	got, err := svc.ClaimPendingQuoteUpdates(ctx, 5)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestQuoteService_RecoverPendingUpdates_Delegates(t *testing.T) {
	svc, quoteRepo, _, _ := newQuoteService(t)
	ctx := t.Context()
	want := []domain.Quote{{ID: uuid.New(), Status: "in_progress"}}

	quoteRepo.EXPECT().GetUndoneQuoteUpdates(ctx).Return(want, nil)

	got, err := svc.RecoverPendingUpdates(ctx)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestQuoteService_ProcessClaimedTask_Success(t *testing.T) {
	svc, quoteRepo, _, provider := newQuoteService(t)
	ctx := t.Context()
	task := domain.Quote{ID: uuid.New(), BaseCurrency: "EUR", QuoteCurrency: "MXN"}
	rate := decimal.NewFromFloat(19.5)
	providerTime := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)

	provider.EXPECT().GetRate(ctx, "EUR", "MXN").Return(rate, providerTime, nil)

	before := time.Now().UTC()
	quoteRepo.EXPECT().
		MarkDone(ctx, task.ID, rate, providerTime, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ decimal.Decimal, _ time.Time, fetchedAt time.Time) error {
			assert.WithinRange(t, fetchedAt, before, time.Now().UTC())
			return nil
		})

	require.NoError(t, svc.ProcessClaimedTask(ctx, task))
}

func TestQuoteService_ProcessClaimedTask_ProviderError_MarkFailedSucceeds(t *testing.T) {
	svc, quoteRepo, _, provider := newQuoteService(t)
	ctx := t.Context()
	task := domain.Quote{ID: uuid.New(), BaseCurrency: "EUR", QuoteCurrency: "MXN"}
	providerErr := errors.New("provider timeout")

	provider.EXPECT().GetRate(ctx, "EUR", "MXN").Return(decimal.Decimal{}, time.Time{}, providerErr)
	// MarkFailed gets its own context (see markFailedTimeout), deliberately not ctx itself —
	// it must survive even when ctx is the thing that just expired.
	quoteRepo.EXPECT().MarkFailed(gomock.Any(), task.ID, providerErr.Error()).Return(nil)

	err := svc.ProcessClaimedTask(ctx, task)
	require.ErrorIs(t, err, providerErr)
}

func TestQuoteService_ProcessClaimedTask_ProviderError_MarkFailedAlsoErrors(t *testing.T) {
	svc, quoteRepo, _, provider := newQuoteService(t)
	ctx := t.Context()
	task := domain.Quote{ID: uuid.New(), BaseCurrency: "EUR", QuoteCurrency: "MXN"}
	providerErr := errors.New("provider timeout")
	markErr := errors.New("db write failed")

	provider.EXPECT().GetRate(ctx, "EUR", "MXN").Return(decimal.Decimal{}, time.Time{}, providerErr)
	quoteRepo.EXPECT().MarkFailed(gomock.Any(), task.ID, providerErr.Error()).Return(markErr)

	err := svc.ProcessClaimedTask(ctx, task)
	require.ErrorIs(t, err, providerErr)
	assert.ErrorIs(t, err, markErr)
}

func TestQuoteService_ProcessClaimedTask_MarkDoneError(t *testing.T) {
	svc, quoteRepo, _, provider := newQuoteService(t)
	ctx := t.Context()
	task := domain.Quote{ID: uuid.New(), BaseCurrency: "EUR", QuoteCurrency: "MXN"}
	rate := decimal.NewFromFloat(19.5)
	providerTime := time.Now().UTC()
	markDoneErr := errors.New("db write failed")

	provider.EXPECT().GetRate(ctx, "EUR", "MXN").Return(rate, providerTime, nil)
	quoteRepo.EXPECT().MarkDone(ctx, task.ID, rate, providerTime, gomock.Any()).Return(markDoneErr)

	err := svc.ProcessClaimedTask(ctx, task)
	require.ErrorIs(t, err, markDoneErr)
}
