package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/spenderella/currency-quotes-service/internal/config"
	"github.com/spenderella/currency-quotes-service/internal/domain"
)

// IQuoteService is the subset of *service.QuoteService the HTTP layer depends on.
type IQuoteService interface {
	CreateQuoteUpdate(ctx context.Context, baseCurrency, quoteCurrency, idempotencyKey string) (uuid.UUID, error)
	GetQuoteUpdateByID(ctx context.Context, id uuid.UUID) (domain.Quote, error)
	GetQuoteUpdateLatest(ctx context.Context, baseCurrency, quoteCurrency string) (domain.Quote, error)
}

type Server struct {
	httpServer   *http.Server
	logger       *slog.Logger
	quoteService IQuoteService
}

// New initializes the HTTP server, router, and all dependencies
func New(ctx context.Context, conf config.HTTPServerConfig, quoteService IQuoteService, logger *slog.Logger) (*Server, error) {
	server := Server{logger: logger, quoteService: quoteService}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /quotes", server.createQuoteHandler)
	mux.HandleFunc("GET /quotes/{id}", server.getQuoteByIDHandler)
	mux.HandleFunc("GET /quotes/latest", server.getLatestQuoteHandler)

	server.httpServer = &http.Server{
		Addr:         conf.Address,
		Handler:      loggingMiddleware(logger, mux),
		ReadTimeout:  time.Duration(conf.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(conf.WriteTimeoutSeconds) * time.Second,
	}

	return &server, nil
}

func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.InfoContext(ctx, "http server is listening", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		return nil
	}
}

func (s *Server) Close(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	s.logger.InfoContext(ctx, "http server is closed")
	return nil
}
