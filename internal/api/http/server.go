package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/spenderella/currency-quotes-service/internal/config"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// New initializes the HTTP server, router, and all dependencies
func New(ctx context.Context, conf config.HTTPServerConfig, logger *slog.Logger) (*Server, error) {

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /quotes", createQuoteHandler)
	mux.HandleFunc("GET /quotes/{id}", getQuoteByIDHandler)
	mux.HandleFunc("GET /quotes/latest", getLatestQuoteHandler)

	server := Server{logger: logger}

	server.httpServer = &http.Server{
		Addr:         conf.Address,
		Handler:      mux,
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
