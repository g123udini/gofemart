package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/g123udini/gofemart/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
)

func init() {
	sql.Register("repo_test_driver", repoTestDriver{})
}

type repoTestDriver struct{}

func (d repoTestDriver) Open(name string) (driver.Conn, error) {
	return &repoTestConn{mode: name}, nil
}

type repoTestConn struct {
	mode string
}

func (c *repoTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}
func (c *repoTestConn) Close() error              { return nil }
func (c *repoTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not supported") }

func (c *repoTestConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	switch c.mode {
	case "exec_ok":
		return driver.RowsAffected(1), nil
	case "exec_uniq":
		return nil, &pgconn.PgError{Code: "23505", Message: "unique violation"}
	case "exec_err":
		return nil, errors.New("exec failed")
	default:
		return nil, errors.New("unknown mode")
	}
}

func (c *repoTestConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch c.mode {
	case "query_norows":
		return &repoTestRowsNoRows{}, nil
	default:
		return nil, errors.New("unknown mode")
	}
}

type repoTestRowsNoRows struct{}

func (r *repoTestRowsNoRows) Columns() []string { return []string{"c1"} }
func (r *repoTestRowsNoRows) Close() error      { return nil }
func (r *repoTestRowsNoRows) Next(dest []driver.Value) error {
	return io.EOF
}

func TestIsValidDSN(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"   ", false},
		{"postgres://", false},
		{"postgres://user:pass@localhost", false},
		{"postgres://user:pass@localhost/", false},
		{"postgres://user:pass@localhost:5432/db", true},
		{"postgres://localhost:5432/db", true},
	}

	for _, tt := range tests {
		if got := isValidDSN(tt.in); got != tt.want {
			t.Fatalf("isValidDSN(%q)=%v want=%v", tt.in, got, tt.want)
		}
	}
}

func TestSaveDB_UniqueConstraint(t *testing.T) {
	db, err := sql.Open("repo_test_driver", "exec_uniq")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := &Repo{DB: db}

	err = repo.SaveDB("INSERT INTO users(login,password) VALUES($1,$2)", "a", "b")
	if !errors.Is(err, ErrUniqConstrait) {
		t.Fatalf("err=%v want ErrUniqConstrait", err)
	}
}

func TestSaveDB_GenericError(t *testing.T) {
	db, err := sql.Open("repo_test_driver", "exec_err")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := &Repo{DB: db}

	err = repo.SaveDB("INSERT INTO users(login,password) VALUES($1,$2)", "a", "b")
	if err == nil || errors.Is(err, ErrUniqConstrait) {
		t.Fatalf("err=%v want generic error (not ErrUniqConstrait)", err)
	}
	if err.Error() != "exec failed" {
		t.Fatalf("err=%q want %q", err.Error(), "exec failed")
	}
}

func TestGetModel_NoRows_ReturnsErrNotFound(t *testing.T) {
	db, err := sql.Open("repo_test_driver", "query_norows")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := &Repo{DB: db}

	u := model.User{}
	err = repo.getModel(&u, "SELECT ... WHERE login=$1", "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v want ErrNotFound", err)
	}
}

func TestGetUserByLogin_NoRows_ReturnsNilNil(t *testing.T) {
	db, err := sql.Open("repo_test_driver", "query_norows")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	repo := &Repo{DB: db}

	u, err := repo.GetUserByLogin("nope")
	if err != nil {
		t.Fatalf("err=%v want nil", err)
	}
	if u != nil {
		t.Fatalf("user=%v want nil", u)
	}
}
