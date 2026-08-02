package service_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/spenderella/currency-quotes-service/internal/service"
	"github.com/spenderella/currency-quotes-service/internal/service/mocks"
)

func TestCurrencyService_LoadCurrencies_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockICurrencyRepository(ctrl)
	repo.EXPECT().
		GetEnabledCurrencies(gomock.Any()).
		Return([]string{"USD", "EUR", "MXN"}, nil)

	svc := service.NewCurrencyService(repo)

	err := svc.LoadCurrencies(t.Context())
	require.NoError(t, err)

	for _, code := range []string{"USD", "EUR", "MXN"} {
		assert.Truef(t, svc.IsSupported(code), "IsSupported(%q) = false, want true", code)
	}
	assert.False(t, svc.IsSupported("GBP"), "GBP is not in the whitelist")
}

func TestCurrencyService_LoadCurrencies_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockICurrencyRepository(ctrl)
	repoErr := errors.New("db unreachable")
	repo.EXPECT().GetEnabledCurrencies(gomock.Any()).Return(nil, repoErr)

	svc := service.NewCurrencyService(repo)

	err := svc.LoadCurrencies(t.Context())
	require.ErrorIs(t, err, repoErr)
}

func TestCurrencyService_LoadCurrencies_EmptyWhitelist(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockICurrencyRepository(ctrl)
	repo.EXPECT().GetEnabledCurrencies(gomock.Any()).Return([]string{}, nil)

	svc := service.NewCurrencyService(repo)

	require.NoError(t, svc.LoadCurrencies(t.Context()))
	assert.False(t, svc.IsSupported("USD"), "whitelist is empty")
}

func TestCurrencyService_IsSupported_BeforeLoad(t *testing.T) {
	svc := service.NewCurrencyService(nil)

	// LoadCurrencies was never called: the internal set is a nil map.
	// Reading from a nil map is well-defined in Go and must not panic.
	assert.False(t, svc.IsSupported("USD"), "want false before LoadCurrencies is called")
}

func TestCurrencyService_IsSupported_CaseSensitive(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockICurrencyRepository(ctrl)
	repo.EXPECT().GetEnabledCurrencies(gomock.Any()).Return([]string{"USD"}, nil)

	svc := service.NewCurrencyService(repo)
	require.NoError(t, svc.LoadCurrencies(t.Context()))

	assert.False(t, svc.IsSupported("usd"), "whitelist matching is case-sensitive")
}
