package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/g123udini/gofemart/internal/handler"
	"github.com/g123udini/gofemart/internal/repository"
	"github.com/g123udini/gofemart/internal/router"
	"github.com/g123udini/gofemart/internal/service"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"net"
	"net/http"
)

func main() {
	f := parseFlags()
	ms := service.NewMemStorage()
	repo, err := repository.NewRepository(f.Dsn)
	if err != nil {
		log.Fatal(err.Error())
	}

	//initMigrations(repo.DB)

	err = run(repo, ms, f)

	if err != nil {
		log.Fatal(err.Error())
	}
}

func run(repo *repository.Repo, ms *service.MemSessionStorage, f *flags) error {
	fmt.Println("Running server on", f.RunAddr)

	normalizeHost(f.RunAddr)

	h := handler.NewHandler(repo, ms)
	r := router.NewRouter(h)

	return http.ListenAndServe(f.RunAddr, r)
}

func normalizeHost(host string) string {
	if h, p, err := net.SplitHostPort(host); err == nil {
		if h == "" {
			host = fmt.Sprintf("localhost:%s", p)
		}
	}
	return host
}

func initMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("postgres driver error: ", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatal("migrate init error: ", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrate up error: %v", err)
	}
}
