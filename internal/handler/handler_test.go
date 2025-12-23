package handler

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/g123udini/gofemart/internal/repository"
	"github.com/g123udini/gofemart/internal/service"
)

func init() {
	sql.Register("handler_test_driver", handlerTestDriver{})
}

type handlerTestDriver struct{}

func (d handlerTestDriver) Open(name string) (driver.Conn, error) {
	return &handlerTestConn{mode: name}, nil
}

type handlerTestConn struct {
	mode string
}

func (c *handlerTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}
func (c *handlerTestConn) Close() error              { return nil }
func (c *handlerTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not supported") }

func (c *handlerTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (c *handlerTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "FROM users") && strings.Contains(query, "WHERE login = $1") {
		if c.mode == "user_ok" {
			return &handlerTestRows{
				cols: []string{"id", "login", "password", "current", "withdrawn"},
				data: [][]driver.Value{
					{int64(1), "u1", "hash", int64(100), int64(7)},
				},
			}, nil
		}
	}

	return &handlerTestRows{cols: []string{"x"}, data: nil}, nil
}

type handlerTestRows struct {
	cols []string
	data [][]driver.Value
	i    int
}

func (r *handlerTestRows) Columns() []string { return r.cols }
func (r *handlerTestRows) Close() error      { return nil }
func (r *handlerTestRows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return ioEOF()
	}
	row := r.data[r.i]
	r.i++
	for i := range dest {
		dest[i] = row[i]
	}
	return nil
}

func ioEOF() error {
	type eof interface{ EOF() }
	return errors.New("EOF")
}

func TestHandler_Test(t *testing.T) {
	h := NewHandler(&repository.Repo{}, service.NewMemStorage())

	req := httptest.NewRequest(http.MethodGet, "/api/user/test", nil)
	rr := httptest.NewRecorder()

	h.Test(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}
	if strings.TrimSpace(rr.Body.String()) != "ok" {
		t.Fatalf("body=%q want=%q", rr.Body.String(), "ok")
	}
}

func TestSessionAuth_NoCookie(t *testing.T) {
	h := NewHandler(&repository.Repo{}, service.NewMemStorage())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)

	h.SessionAuth(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}
}

func TestSessionAuth_InvalidSession(t *testing.T) {
	h := NewHandler(&repository.Repo{}, service.NewMemStorage())

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "nope"})

	h.SessionAuth(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}
}

func TestSessionAuth_ValidSession(t *testing.T) {
	ms := service.NewMemStorage()
	ms.AddSession("sid1", "u1")
	h := NewHandler(&repository.Repo{}, ms)

	nextCalled := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled++
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid1"})

	h.SessionAuth(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusOK)
	}
	if nextCalled != 1 {
		t.Fatalf("nextCalled=%d want=1", nextCalled)
	}
}

func TestGetBalance_Unauthorized(t *testing.T) {
	db, _ := sql.Open("handler_test_driver", "user_ok")
	repo := &repository.Repo{DB: db}
	h := NewHandler(repo, service.NewMemStorage())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/user/balance/", nil)

	h.GetBalance(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}
}

func TestGetBalance_OK(t *testing.T) {
	db, _ := sql.Open("handler_test_driver", "user_ok")
	repo := &repository.Repo{DB: db}

	ms := service.NewMemStorage()
	ms.AddSession("sid1", "u1")

	h := NewHandler(repo, ms)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/user/balance/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid1"})

	h.GetBalance(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%q", rr.Code, http.StatusOK, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type=%q", ct)
	}

	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v body=%q", err, rr.Body.String())
	}
	if got["current"] == nil {
		t.Fatalf("expected current in response, got=%v", got)
	}
	if got["withdraw"] == nil {
		t.Fatalf("expected withdraw in response, got=%v", got)
	}
}

func TestAddOrder_InvalidLuhn(t *testing.T) {
	h := NewHandler(&repository.Repo{}, service.NewMemStorage())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("123"))
	req.Header.Set("Content-Type", "text/plain")

	h.AddOrder(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestAddOrder_Unauthorized(t *testing.T) {
	h := NewHandler(&repository.Repo{}, service.NewMemStorage())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("79927398713"))
	req.Header.Set("Content-Type", "text/plain")

	h.AddOrder(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}
}

func TestWithdraw_BadJSON(t *testing.T) {
	h := NewHandler(&repository.Repo{}, service.NewMemStorage())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")

	h.Withdraw(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusBadRequest)
	}
}

func TestWithdraw_InvalidLuhn(t *testing.T) {
	h := NewHandler(&repository.Repo{}, service.NewMemStorage())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"123","sum":10}`))
	req.Header.Set("Content-Type", "application/json")

	h.Withdraw(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestWithdraw_Unauthorized(t *testing.T) {
	h := NewHandler(&repository.Repo{}, service.NewMemStorage())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"79927398713","sum":10}`))
	req.Header.Set("Content-Type", "application/json")

	h.Withdraw(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rr.Code, http.StatusUnauthorized)
	}
}

func TestWithdraw_InsufficientBalance(t *testing.T) {
	db, _ := sql.Open("handler_test_driver", "user_ok")
	repo := &repository.Repo{DB: db}

	ms := service.NewMemStorage()
	ms.AddSession("sid1", "u1")

	h := NewHandler(repo, ms)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", strings.NewReader(`{"order":"79927398713","sum":999999}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sid1"})

	h.Withdraw(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("status=%d want=%d body=%q", rr.Code, http.StatusPaymentRequired, rr.Body.String())
	}
}
