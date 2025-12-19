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
	ErrUniqConstrait = errors.New("already exists")
	ErrNotFound      = errors.New("user not found")
)

type Repo struct {
	DB *sql.DB
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

	return &Repo{DB: db}, nil
}

func (repo *Repo) GetUserByLogin(login string) (*model.User, error) {
	u := model.User{}

	err := repo.getModel(&u, "SELECT id, login, password, current, withdrawn FROM users WHERE login = $1", login)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

func (repo *Repo) GetOrderByNumberUser(number string, user *model.User) (*model.Order, error) {
	var order model.Order

	err := repo.getModel(
		&order,
		`SELECT number, status, accural, uploaded_at, user_id
		 FROM orders
		 WHERE number = $1 AND user_id = $2`,
		number,
		user.ID,
	)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &order, nil
}

func (repo *Repo) GetOrdersByUser(user *model.User) ([]model.Order, error) {
	rows, err := repo.DB.Query(
		`SELECT number, status, accural, uploaded_at, user_id
		 FROM orders
		 WHERE user_id = $1
		 ORDER BY uploaded_at ASC`,
		user.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.Order, 0)

	for rows.Next() {
		var order model.Order

		if err := rows.Scan(order.ScanFields()...); err != nil {
			return nil, err
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (repo *Repo) getModel(
	model model.Model,
	sqlString string,
	args ...any,
) error {

	err := repo.DB.
		QueryRow(sqlString, args...).
		Scan(model.ScanFields()...)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (repo *Repo) SaveUser(user *model.User) error {
	return repo.SaveDB("INSERT INTO users (login, password) VALUES ($1, $2)", user.Login, user.Password)
}

func (repo *Repo) UpdateUser(user *model.User) error {
	return repo.SaveDB("UPDATE users SET login = $1, password = $2, current = $3, withdrawn = $4 WHERE id = $5", user.Login, user.Password, user.Balance.Current, user.Balance.Withdrawn, user.ID)
}

func (repo *Repo) SaveWithdrawal(w *model.Withdrawal) error {
	return repo.SaveDB("INSERT INTO withdrawals (number, sum, user_id) VALUES ($1, $2, $3)", w.Number, w.Sum, w.UserID)
}

func (repo *Repo) SaveOrder(order *model.Order) error {
	return repo.
		SaveDB(
			"INSERT INTO orders (number, status, accural, uploaded_at, user_id) VALUES ($1, $2, $3, $4, $5)",
			order.Number, order.Status, order.Accrual, order.UploadedAt, order.UserID,
		)
}

func (repo *Repo) SaveDB(sqlString string, args ...any) error {
	_, err := service.RetryDB(
		3,
		1*time.Second,
		2*time.Second,
		func() (sql.Result, error) {
			return repo.DB.Exec(sqlString, args...)
		},
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUniqConstrait
		}
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
