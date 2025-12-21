package service

import (
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"strconv"
	"time"
)

func RetryDB[T any](attempts int, base, step time.Duration, fn func() (T, error)) (T, error) {
	var (
		res T
		err error
	)

	if attempts <= 0 {
		return res, fmt.Errorf("attempts must be > 0")
	}

	for i := 1; i <= attempts; i++ {
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

		// если это была последняя попытка — выходим без sleep
		if i == attempts {
			break
		}

		delay := base + step*time.Duration(i-1)
		time.Sleep(delay)
	}

	return res, fmt.Errorf("after %d attempts, last error: %w", attempts, err)
}

func Retry(
	baseDelay,
	maxDelay time.Duration,
	retries,
	multiplier uint64,
	function func() error,
	filter func(err error) bool,
) error {
	var err error
	retry := uint64(0)

	for {
		if retry > retries {
			return err
		}

		if err = function(); err == nil {
			return nil
		}

		if filter != nil && !filter(err) {
			return err
		}

		time.Sleep(calculateDelay(baseDelay, maxDelay, retry, multiplier))
		retry++
	}
}

func pow(x, y uint64) uint64 {
	if y == 0 {
		return 1
	}

	if y == 1 {
		return x
	}

	result := x
	for i := uint64(2); i <= y; i++ {
		result *= x
	}
	return result
}

func calculateDelay(baseDelay time.Duration, maxDelay time.Duration, attempt uint64, multiplier uint64) time.Duration {
	if attempt == 0 {
		return min(baseDelay, maxDelay)
	} else {
		return min(baseDelay*time.Duration(pow(multiplier, attempt)), maxDelay)
	}
}

func Parse(retryAfter string, defaultValue time.Duration) time.Duration {
	if duration, err := parseSeconds(retryAfter); err == nil {
		return duration
	}

	if dateTime, err := parseHTTPDate(retryAfter); err == nil {
		duration := time.Until(dateTime)

		if duration < 0 {
			return 0
		}

		return duration
	}

	return defaultValue
}

func parseSeconds(retryAfter string) (time.Duration, error) {
	seconds, err := strconv.ParseUint(retryAfter, 10, 64)
	if err != nil {
		return time.Duration(0), err
	}

	return time.Second * time.Duration(seconds), nil
}

func parseHTTPDate(retryAfter string) (time.Time, error) {
	dateTime, err := time.Parse(time.RFC1123, retryAfter)
	if err != nil {
		return time.Time{}, err
	}

	return dateTime, nil
}
