package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	httpserver "github.com/spenderella/currency-quotes-service/internal/api/http"
	"github.com/spenderella/currency-quotes-service/internal/config"
)

type Application struct {
	conf       *config.Configuration
	logger     *slog.Logger
	httpServer *httpserver.Server
	postgres   *sql.DB
}

func New(ctx context.Context, logger *slog.Logger) (*Application, error) {
	app := &Application{
		logger: logger,
	}
	if err := app.setConfig(); err != nil {
		return nil, fmt.Errorf("set configuration: %w", err)
	}

	if err := app.setServer(ctx, app.conf.HTTPServer); err != nil {
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

func (a *Application) Start(ctx context.Context) error {
	return a.httpServer.Start(ctx)
}

func (a *Application) Close(ctx context.Context) error {
	errs := []error{
		a.httpServer.Close(ctx),
	}
	return errors.Join(errs...)
}
