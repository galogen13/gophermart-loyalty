package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"go.uber.org/zap"
)

const (
	maxAttempts = 3
	firstDelay  = 1
)

func Do(ctx context.Context, op func() error, classifier ErrorClassifier) error {
	var err error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err = op()
		if err == nil {
			return nil
		}

		if attempt == maxAttempts {
			break
		}

		classification := classifier.Classify(err)
		if classification == NonRetriable {
			return fmt.Errorf("non retriable error: %w", err)
		}

		delay := time.Second * time.Duration(firstDelay+attempt*2)

		logger.Log.Info("retriable error, operaion delayed", zap.Duration("delay", delay), zap.Error(err))

		select {
		case <-time.After(delay):
			// Продолжаем попытки
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled: %w", ctx.Err())
		}
	}

	return fmt.Errorf("aborted after %d attempts: %w", maxAttempts+1, err)
}

func DoWithResult[T any](ctx context.Context, op func() (T, error), classifier ErrorClassifier) (T, error) {
	var zero T
	var err error
	var result T

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err = op()
		if err == nil {
			return result, nil
		}

		// после последней попытки не делаем delay, возвращаем что получилось
		if attempt == maxAttempts {
			break
		}

		classification := classifier.Classify(err)
		if classification == NonRetriable {
			return zero, fmt.Errorf("non retriable error: %w", err)
		}

		delay := time.Second * time.Duration(firstDelay+attempt*2)
		logger.Log.Info("retriable error, operaion delayed", zap.Duration("delay", delay), zap.Error(err))

		select {
		case <-time.After(delay):
			// Продолжаем попытки
		case <-ctx.Done():
			return zero, fmt.Errorf("operation cancelled: %w", ctx.Err())
		}
	}

	return zero, fmt.Errorf("aborted after %d attempts: %w", maxAttempts+1, err)
}
