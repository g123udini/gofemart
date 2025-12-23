package router

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/g123udini/gofemart/internal/handler"
	"github.com/g123udini/gofemart/internal/repository"
	"github.com/g123udini/gofemart/internal/service"
	"github.com/go-chi/chi/v5"
)

func init() {
	sql.Register("router_test_driver", routerTestDriver{})
}

type routerTestDriver struct{}

func (d routerTestDriver) Open(name string) (driver.Conn, error) {
	return &routerTestConn{mode: name}, nil
}

type routerTestConn struct {
	mode string
}

func (c *routerTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}
func (c *routerTestConn) Close() error              { return nil }
func (c *routerTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not supported") }

func (c *routerTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	switch c.mode {
	case "exec_ok":
		return driver.RowsAffected(1), nil
	default:
		return driver.RowsAffected(1), nil
	}
}

func (c *routerTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return &routerNoRows{cols: []string{"c1"}}, nil
}

type routerNoRows struct {
	cols []string
}

func (r *routerNoRows) Columns() []string { return r.cols }
func (r *routerNoRows) Close() error      { return nil }
func (r *routerNoRows) Next(dest []driver.Value) error {
	return io.EOF
}

func newTestRouter(t *testing.T) chi.Router {
	t.Helper()

	db, err := sql.Open("router_test_driver", "exec_ok")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	repo := &repository.Repo{DB: db}
	ms := service.NewMemStorage()
	h := handler.NewHandler(repo, ms)

	return NewRouter(h)
}

func TestRouter_RoutesRegistered(t *testing.T) {
	r := newTestRouter(t)

	want := map[string]struct{}{
		"POST /api/user/register":         {},
		"POST /api/user/login":            {},
		"GET /api/user/test":              {},
		"POST /api/user/orders":           {},
		"GET /api/user/orders":            {},
		"GET /api/user/balance/":          {},
		"POST /api/user/balance/withdraw": {},
	}

	got := map[string]struct{}{}

	err := chi.Walk(r, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method != "" && route != "" {
			got[method+" "+route] = struct{}{}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	for k := range want {
		if _, ok := got[k]; !ok {
			t.Fatalf("missing route: %s", k)
		}
	}
}

func TestRouter_Test_UnauthorizedWithoutCookie(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/user/test", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d body=%q", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
}

func TestRouter_AddOrder_WrongContentType(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("79927398713"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusAccepted || rr.Code == http.StatusOK {
		t.Fatalf("status=%d unexpected body=%q", rr.Code, rr.Body.String())
	}
}

func TestRouter_AddOrder_CorrectContentTypeButUnauthorized(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", strings.NewReader("79927398713"))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d body=%q", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
}

func TestRouter_Register_OK_SetsCookie(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader(`{"login":"u1","password":"p1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%q", rr.Code, http.StatusOK, rr.Body.String())
	}

	res := rr.Result()
	defer res.Body.Close()

	found := false
	for _, c := range res.Cookies() {
		if c.Name == "session_id" && c.Value != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected session_id cookie to be set")
	}
}
