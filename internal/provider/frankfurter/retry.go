package frankfurter

import (
	"context"
	"errors"
	"time"
)

// retryableError marks an error as transient — worth retrying.
type retryableError struct {
	err error
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	var re *retryableError
	return errors.As(err, &re)
}

func withRetry(ctx context.Context, maxAttempts int, baseDelay time.Duration, fn func() error) error {
	var err error
	for attempt := range maxAttempts {
		if err = fn(); err == nil || !isRetryable(err) {
			return err
		}
		if attempt == maxAttempts-1 {
			return err
		}
		delay := baseDelay * time.Duration(1<<attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return err
}
