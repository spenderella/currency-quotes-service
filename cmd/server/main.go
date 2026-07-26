package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spenderella/currency-quotes-service/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	application, err := app.New(ctx, logger)
	if err != nil {
		logger.Error("error while inject dependencies", "error", err)
		os.Exit(1)
	}

	startErr := application.Start(ctx)
	if startErr != nil {
		logger.Error("running the app", "error", startErr)
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Close(closeCtx); err != nil {
		logger.Error("closing the app", "error", err)
	}
	logger.Info("application stopped")

	if startErr != nil {
		os.Exit(1)
	}
}
