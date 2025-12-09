package repository

import (
	"database/sql"
	"errors"
	"github.com/g123udini/gofemart/internal/model"
	"github.com/g123udini/gofemart/internal/service"
	"github.com/jackc/pgx/v5/pgconn"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	ErrUserAlreadyExists = errors.New("login already exists")
	ErrNotFoundUser      = errors.New("user not found")
)

type Repo struct {
	Db *sql.DB
	mu sync.RWMutex
}

func NewRepository(DSN string) (*Repo, error) {
	if !isValidDSN(DSN) {
		return nil, errors.New("invalid DSN")
	}
	db, err := sql.Open("pgx", DSN)

	if err != nil {
		log.Fatal(err)
	}

	return &Repo{Db: db}, nil
}

func (repo *Repo) GetUserByLogin(login string) (model.User, error) {
	repo.mu.RLock()
	repo.mu.RUnlock()

	u := model.User{}

	_, err := service.RetryDB(3, 1*time.Second, 2*time.Second, func() (sql.Result, error) {
		return nil, repo.Db.QueryRow("SELECT login, password FROM users WHERE login = $1", login).Scan(&u.Login, &u.Password)
	})

	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, ErrNotFoundUser
	}

	if err != nil {
		return u, err
	}

	return u, nil
}

func (repo *Repo) SaveUser(user model.User) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	_, err := service.RetryDB(3, 1*time.Second, 2*time.Second, func() (sql.Result, error) {
		return repo.Db.Exec("INSERT INTO users (login, password) VALUES ($1, $2)", user.Login, user.Password)
	})

	var pgErr *pgconn.PgError
	if err != nil && errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return ErrUserAlreadyExists
		}
	}

	if err != nil {
		return err
	}

	return nil
}

func isValidDSN(dsn string) bool {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return false
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return false
	}

	if u.Host == "" {
		return false
	}

	if u.Path == "" || u.Path == "/" {
		return false
	}

	return true
}
