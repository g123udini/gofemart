package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_GetOrder_OK_WithAccrual(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order":   "123",
			"status":  "PROCESSED",
			"accrual": 500.5,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())

	oi, err := c.GetOrder(context.Background(), "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if oi.Order != "123" {
		t.Fatalf("order mismatch")
	}
	if oi.Status != StatusProcessed {
		t.Fatalf("status mismatch: %s", oi.Status)
	}
	if oi.Accrual == nil || *oi.Accrual != 500.5 {
		t.Fatalf("accrual mismatch: %+v", oi.Accrual)
	}
}

func TestClient_GetOrder_OK_WithoutAccrual(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order":  "123",
			"status": "PROCESSING",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())

	oi, err := c.GetOrder(context.Background(), "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if oi.Accrual != nil {
		t.Fatalf("expected nil accrual, got %+v", oi.Accrual)
	}
}

func TestClient_GetOrder_204_NotRegistered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())

	_, err := c.GetOrder(context.Background(), "123")
	if !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("expected ErrNotRegistered, got %v", err)
	}
}

func TestClient_GetOrder_429_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "42")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())

	_, err := c.GetOrder(context.Background(), "123")

	rl, ok := err.(RateLimitError)
	if !ok {
		t.Fatalf("expected RateLimitError, got %T (%v)", err, err)
	}

	if rl.RetryAfter != 42*time.Second {
		t.Fatalf("unexpected RetryAfter: %s", rl.RetryAfter)
	}
}

func TestClient_GetOrder_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())

	_, err := c.GetOrder(context.Background(), "123")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestClient_GetOrder_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, srv.Client())

	_, err := c.GetOrder(context.Background(), "123")
	if err == nil {
		t.Fatalf("expected JSON decode error")
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want time.Duration
	}{
		{"valid", "10", 10 * time.Second},
		{"spaces", " 5 ", 5 * time.Second},
		{"empty", "", 60 * time.Second},
		{"invalid", "abc", 60 * time.Second},
		{"zero", "0", 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseRetryAfter(tt.in); got != tt.want {
				t.Fatalf("got %s want %s", got, tt.want)
			}
		})
	}
}
