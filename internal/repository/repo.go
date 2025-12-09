package repository

import (
	"database/sql"
	"log"
	"net/url"
	"strings"
)

type Repo struct {
	db *sql.DB
}

func NewRepository(DSN string) *Repo {
	if !isValidDSN(DSN) {
		return nil
	}
	db, err := sql.Open("pgx", DSN)

	if err != nil {
		log.Fatal(err)
	}

	return &Repo{db: db}
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
