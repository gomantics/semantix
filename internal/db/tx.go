package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxRetries = 3
	retryDelay = 10 * time.Millisecond
)

// isRetryableError checks if an error is safe to retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check if pgx thinks it's safe to retry (connection errors before sending data)
	if pgconn.SafeToRetry(err) {
		return true
	}

	// Check for specific PostgreSQL error codes
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001": // serialization_failure
			return true
		case "40P01": // deadlock_detected
			return true
		case "08000", "08003", "08006": // connection errors
			return true
		}
	}

	return false
}

// Query runs fn with a Queries instance.
// Automatically retries on transient errors.
func Query(ctx context.Context, fn func(*Queries) error) error {
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			delay := retryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn(New(defaultPool))
		if err == nil {
			return nil
		}

		lastErr = err
		if !isRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("query failed after %d attempts: %w", maxRetries, lastErr)
}

// Query1 runs fn and returns a single result.
// Automatically retries on transient errors.
func Query1[T any](ctx context.Context, fn func(*Queries) (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			delay := retryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
			}
		}

		var err error
		result, err = fn(New(defaultPool))
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isRetryableError(err) {
			return result, err
		}
	}

	return result, fmt.Errorf("query failed after %d attempts: %w", maxRetries, lastErr)
}

// Tx runs fn within a transaction.
// Automatically retries on transient errors.
func Tx(ctx context.Context, fn func(*Queries) error) error {
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			delay := retryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		conn, err := defaultPool.Acquire(ctx)
		if err != nil {
			lastErr = err
			if isRetryableError(err) {
				continue
			}
			return err
		}

		err = executeTx(ctx, conn, fn)
		conn.Release()

		if err == nil {
			return nil
		}

		lastErr = err
		if !isRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("transaction failed after %d attempts: %w", maxRetries, lastErr)
}

// Tx1 runs fn within a transaction and returns a result.
// Automatically retries on transient errors.
func Tx1[T any](ctx context.Context, fn func(*Queries) (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			delay := retryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(delay):
			}
		}

		conn, err := defaultPool.Acquire(ctx)
		if err != nil {
			lastErr = err
			if isRetryableError(err) {
				continue
			}
			return result, err
		}

		result, err = executeTx1(ctx, conn, fn)
		conn.Release()

		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isRetryableError(err) {
			return result, err
		}
	}

	return result, fmt.Errorf("transaction failed after %d attempts: %w", maxRetries, lastErr)
}

// executeTx is a helper that executes the transaction logic
func executeTx(ctx context.Context, conn *pgxpool.Conn, fn func(*Queries) error) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(New(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// executeTx1 is a helper that executes the transaction logic and returns one value
func executeTx1[T any](ctx context.Context, conn *pgxpool.Conn, fn func(*Queries) (T, error)) (T, error) {
	var zero T

	tx, err := conn.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback(ctx)

	result, err := fn(New(tx))
	if err != nil {
		return zero, err
	}

	if err := tx.Commit(ctx); err != nil {
		return zero, err
	}

	return result, nil
}
