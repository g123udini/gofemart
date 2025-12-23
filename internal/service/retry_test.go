package service

import (
	"errors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"math/rand/v2"
	"strings"
	"testing"
	"time"
)

func pgErr(code string) error {
	return &pgconn.PgError{
		Code:    code,
		Message: "db error",
	}
}

func TestRetryDB_SuccessFirstTry(t *testing.T) {
	t.Parallel()

	calls := 0
	fn := func() (int, error) {
		calls++
		return 42, nil
	}

	got, err := RetryDB[int](3, 0, 0, fn)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != 42 {
		t.Fatalf("got=%d want=42", got)
	}
	if calls != 1 {
		t.Fatalf("calls=%d want=1", calls)
	}
}

func TestRetryDB_RetryablePgErrorThenSuccess(t *testing.T) {
	t.Parallel()

	calls := 0
	fn := func() (int, error) {
		calls++
		if calls <= 2 {
			return 0, pgErr(pgerrcode.ConnectionFailure)
		}
		return 7, nil
	}

	got, err := RetryDB[int](3, 0, 0, fn) // 2 фейла + 1 успех
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != 7 {
		t.Fatalf("got=%d want=7", got)
	}
	if calls != 3 {
		t.Fatalf("calls=%d want=3", calls)
	}
}

func TestRetryDB_NonPgErrorStopsImmediately(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	calls := 0
	fn := func() (int, error) {
		calls++
		return 0, wantErr
	}

	_, err := RetryDB[int](5, 0, 0, fn)
	if err == nil {
		t.Fatalf("expected err, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("err=%v; want errors.Is(..., boom)=true", err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d want=1", calls)
	}
}

func TestRetryDB_NonRetryablePgErrorStopsImmediately(t *testing.T) {
	t.Parallel()

	calls := 0
	fn := func() (int, error) {
		calls++
		return 0, pgErr(pgerrcode.UniqueViolation) // не в whitelist
	}

	_, err := RetryDB[int](5, 0, 0, fn)
	if err == nil {
		t.Fatalf("expected err, got nil")
	}

	var pe *pgconn.PgError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PgError, got: %T (%v)", err, err)
	}
	if pe.SQLState() != pgerrcode.UniqueViolation {
		t.Fatalf("sqlstate=%s want=%s", pe.SQLState(), pgerrcode.UniqueViolation)
	}

	if calls != 1 {
		t.Fatalf("calls=%d want=1", calls)
	}
}

func TestRetryDB_ExhaustAttempts_ReturnsWrappedError(t *testing.T) {
	t.Parallel()

	calls := 0
	fn := func() (int, error) {
		calls++
		return 0, pgErr(pgerrcode.DeadlockDetected)
	}

	_, err := RetryDB[int](3, 0, 0, fn)
	if err == nil {
		t.Fatalf("expected err, got nil")
	}
	if calls != 3 {
		t.Fatalf("calls=%d want=3", calls)
	}
	if !strings.Contains(err.Error(), "after 3 attempts") {
		t.Fatalf("err=%q expected to contain %q", err.Error(), "after 3 attempts")
	}

	var pe *pgconn.PgError
	if !errors.As(err, &pe) {
		t.Fatalf("expected wrapped PgError, got: %T (%v)", err, err)
	}
	if pe.SQLState() != pgerrcode.DeadlockDetected {
		t.Fatalf("sqlstate=%s want=%s", pe.SQLState(), pgerrcode.DeadlockDetected)
	}
}

func TestRetryDB_Attempts1_CallsOnce(t *testing.T) {
	t.Parallel()

	calls := 0
	fn := func() (int, error) {
		calls++
		return 0, pgErr(pgerrcode.TooManyConnections)
	}

	_, err := RetryDB[int](1, 10*time.Millisecond, 10*time.Millisecond, fn)
	if err == nil {
		t.Fatalf("expected err, got nil")
	}
	if calls != 1 {
		t.Fatalf("calls=%d want=1", calls)
	}
}

func Test_pow(t *testing.T) {
	tests := []struct {
		name string
		x    uint64
		y    uint64
		want uint64
	}{
		{
			name: "2 pow 2",
			x:    2,
			y:    2,
			want: 4,
		},
		{
			name: "10 pow 10",
			x:    10,
			y:    10,
			want: 10000000000,
		},
		{
			name: "3 pow 17",
			x:    3,
			y:    17,
			want: 129140163,
		},
		{
			name: "17 pow 3",
			x:    17,
			y:    3,
			want: 4913,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, pow(tt.x, tt.y))
		})
	}
}

func Test_calculateDelay(t *testing.T) {
	tests := []struct {
		name       string
		baseDelay  time.Duration
		maxDelay   time.Duration
		attempt    uint64
		multiplier uint64
		want       time.Duration
	}{
		{
			name:       "1/1",
			baseDelay:  time.Second,
			maxDelay:   10 * time.Second,
			attempt:    0,
			multiplier: 2,
			want:       time.Second,
		},
		{
			name:       "1/2",
			baseDelay:  time.Second,
			maxDelay:   10 * time.Second,
			attempt:    1,
			multiplier: 2,
			want:       2 * time.Second,
		},
		{
			name:       "1/3",
			baseDelay:  time.Second,
			maxDelay:   10 * time.Second,
			attempt:    2,
			multiplier: 2,
			want:       4 * time.Second,
		},
		{
			name:       "1/4",
			baseDelay:  time.Second,
			maxDelay:   10 * time.Second,
			attempt:    3,
			multiplier: 2,
			want:       8 * time.Second,
		},
		{
			name:       "2/1",
			baseDelay:  2 * time.Second,
			maxDelay:   15 * time.Second,
			attempt:    0,
			multiplier: 3,
			want:       2 * time.Second,
		},
		{
			name:       "2/2",
			baseDelay:  2 * time.Second,
			maxDelay:   15 * time.Second,
			attempt:    1,
			multiplier: 3,
			want:       6 * time.Second,
		},
		{
			name:       "2/3",
			baseDelay:  2 * time.Second,
			maxDelay:   15 * time.Second,
			attempt:    2,
			multiplier: 3,
			want:       15 * time.Second,
		},
		{
			name:       "2/4",
			baseDelay:  2 * time.Second,
			maxDelay:   15 * time.Second,
			attempt:    3,
			multiplier: 3,
			want:       15 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, calculateDelay(tt.baseDelay, tt.maxDelay, tt.attempt, tt.multiplier))
		})
	}
}

func TestRetry(t *testing.T) {
	attempts := uint64(0)
	tests := []struct {
		name         string
		retries      uint64
		wantAttempts uint64
		function     func() error
		filter       func(err error) bool
		wantErr      bool
	}{
		{
			name:         "first attempt ok",
			retries:      1,
			wantAttempts: 1,
			function: func() error {
				return nil
			},
			filter: func(err error) bool {
				return true
			},
			wantErr: false,
		},
		{
			name:         "non-retryable error",
			retries:      3,
			wantAttempts: 1,
			function: func() error {
				return errors.New("test error")
			},
			filter: func(err error) bool {
				return err.Error() != "test error"
			},
			wantErr: true,
		},
		{
			name:         "retryable error fail",
			retries:      3,
			wantAttempts: 4,
			function: func() error {
				return errors.New("test error")
			},
			filter: func(err error) bool {
				return err.Error() == "test error"
			},
			wantErr: true,
		},
		{
			name:         "retryable error ok",
			retries:      3,
			wantAttempts: 2,
			function: func() error {
				if attempts > 1 {
					return nil
				}
				return errors.New("test error")
			},
			filter: func(err error) bool {
				return err.Error() == "test error"
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			function := func() error {
				attempts++
				return tt.function()
			}
			err := Retry(0, 0, tt.retries, 0, function, tt.filter)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.wantAttempts, attempts)
			attempts = 0
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		want       time.Duration
	}{
		{
			name:       "success date",
			retryAfter: time.Now().Round(time.Hour).Add(time.Hour * 10).Format(time.RFC1123),
			want:       time.Until(time.Now().Round(time.Hour).Add(time.Hour * 10)),
		},
		{
			name:       "success second",
			retryAfter: "30",
			want:       30 * time.Second,
		},
		{
			name:       "failure",
			retryAfter: "abc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultValue := time.Second * time.Duration(rand.Uint64N(1000))
			got := Parse(tt.retryAfter, defaultValue)
			if tt.want > 0 {
				assert.Equal(t, tt.want.Nanoseconds()/time.Second.Nanoseconds(), got.Nanoseconds()/time.Second.Nanoseconds())
			} else {
				assert.Equal(t, defaultValue, got)
			}
		})
	}
}

func Test_parseHTTPDate(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		want       time.Time
		wantErr    bool
	}{
		{
			name:       "success",
			retryAfter: time.Now().UTC().Format(time.RFC1123),
			want:       time.Now().UTC(),
		},
		{
			name:       "failure",
			retryAfter: "abc",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHTTPDate(tt.retryAfter)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want.Round(time.Second*2), got.Round(time.Second*2))
		})
	}
}

func Test_parseSeconds(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		want       time.Duration
		wantErr    bool
	}{
		{
			name:       "success",
			retryAfter: "30",
			want:       30 * time.Second,
		},
		{
			name:       "failure",
			retryAfter: "abc",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSeconds(tt.retryAfter)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
