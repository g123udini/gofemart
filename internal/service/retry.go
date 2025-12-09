package service

import (
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"time"
)

func RetryDB[T any](attempts int, base, step time.Duration, fn func() (T, error)) (T, error) {
	var (
		res T
		err error
	)

	for i := 1; i < attempts; i++ {
		res, err = fn()

		if err == nil {
			return res, nil
		}

		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			return res, err
		}

		if pgErr.SQLState() != pgerrcode.ConnectionFailure &&
			pgErr.SQLState() != pgerrcode.TooManyConnections &&
			pgErr.SQLState() != pgerrcode.DeadlockDetected {
			return res, err
		}

		delay := base + step*time.Duration(i-1)
		time.Sleep(delay)
	}

	return res, fmt.Errorf("after %d attempts, last error: %w", attempts, err)
}
