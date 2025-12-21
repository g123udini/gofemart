package accrual

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetAccrual_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/123" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"order":"123","status":"PROCEEDED","accrual":10}`))
	}))
	defer srv.Close()

	c := New(srv.URL)

	res, err := c.GetAccrual(context.Background(), 123)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if res == nil {
		t.Fatalf("nil result")
	}
}

func TestGetAccrual_OrderNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL)

	_, err := c.GetAccrual(context.Background(), 1)
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("err=%v want ErrOrderNotFound", err)
	}
}

func TestGetAccrual_TooManyRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := New(srv.URL)

	now := time.Now()
	_, err := c.GetAccrual(context.Background(), 1)

	var tm ErrTooManyRequests
	if !errors.As(err, &tm) {
		t.Fatalf("err=%T %v want ErrTooManyRequests", err, err)
	}

	d := tm.RetryAfterTime.Sub(now)
	if d < 4*time.Second || d > 7*time.Second {
		t.Fatalf("retry-after delta=%v want about 5s", d)
	}
}

func TestGetAccrual_InternalServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL)

	_, err := c.GetAccrual(context.Background(), 1)
	if !errors.Is(err, ErrInternalServerError) {
		t.Fatalf("err=%v want ErrInternalServerError", err)
	}
}

func TestGetAccrual_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	c := New(srv.URL)

	_, err := c.GetAccrual(context.Background(), 1)

	var us ErrUnexpectedStatus
	if !errors.As(err, &us) {
		t.Fatalf("err=%T %v want ErrUnexpectedStatus", err, err)
	}
	if us.Status != http.StatusTeapot {
		t.Fatalf("status=%d want=%d", us.Status, http.StatusTeapot)
	}
}
