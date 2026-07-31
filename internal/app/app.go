package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	httpserver "github.com/spenderella/currency-quotes-service/internal/api/http"
	"github.com/spenderella/currency-quotes-service/internal/config"
	"github.com/spenderella/currency-quotes-service/internal/db/postgres"
	"github.com/spenderella/currency-quotes-service/internal/repository"
	"github.com/spenderella/currency-quotes-service/internal/service"
)

type Application struct {
	conf            *config.Configuration
	logger          *slog.Logger
	httpServer      *httpserver.Server
	postgres        *sql.DB
	quoteRepo       *repository.QuoteRepository
	currencyRepo    *repository.CurrencyRepository
	currencyService *service.CurrencyService
	quoteService    *service.QuoteService
}

func New(ctx context.Context, logger *slog.Logger) (*Application, error) {
	app := &Application{
		logger: logger,
	}
	var err error
	defer func() {
		if err != nil {
			app.Close(ctx)
		}
	}()

	if err = app.setConfig(); err != nil {
		return nil, fmt.Errorf("set configuration: %w", err)
	}

	if err = app.setDatabase(app.conf.Postgres); err != nil {
		return nil, fmt.Errorf("set database: %w", err)
	}

	if err = app.setRepositories(); err != nil {
		return nil, fmt.Errorf("set repositories: %w", err)
	}

	if err = app.setServices(ctx); err != nil {
		return nil, fmt.Errorf("set services: %w", err)
	}

	if err = app.setServer(ctx, app.conf.HTTPServer); err != nil {
		return nil, fmt.Errorf("set server: %w", err)
	}

	return app, nil
}

func (a *Application) setConfig() error {
	conf, err := config.New()
	if err != nil {
		return fmt.Errorf("new configuration: %w", err)
	}
	a.conf = conf
	return nil
}

func (a *Application) setServer(ctx context.Context, conf config.HTTPServerConfig) error {
	srv, err := httpserver.New(ctx, conf, a.logger)
	if err != nil {
		return fmt.Errorf("create http server: %w", err)
	}
	a.httpServer = srv
	return nil
}

func (a *Application) setDatabase(conf config.PostgresConfig) error {
	postgresClient, err := postgres.Connect(conf)
	if err != nil {
		return fmt.Errorf("create postgres connection: %w", err)
	}
	a.postgres = postgresClient
	return nil
}

func (a *Application) setRepositories() error {
	a.quoteRepo = repository.NewQuoteRepository(a.postgres)
	a.currencyRepo = repository.NewCurrencyRepository(a.postgres)
	return nil
}

func (a *Application) setServices(ctx context.Context) error {
	a.currencyService = service.NewCurrencyService(a.currencyRepo)
	if err := a.currencyService.LoadCurrencies(ctx); err != nil {
		return fmt.Errorf("load currencies: %w", err)
	}

	a.quoteService = service.NewQuoteService(a.quoteRepo, a.currencyService)
	return nil
}

func (a *Application) Start(ctx context.Context) error {
	return a.httpServer.Start(ctx)
}

func (a *Application) Close(ctx context.Context) error {
	errs := []error{}
	if a.httpServer != nil {
		errs = append(errs, a.httpServer.Close(ctx))
	}
	if a.postgres != nil {
		errs = append(errs, a.postgres.Close())
	}
	return errors.Join(errs...)
}
