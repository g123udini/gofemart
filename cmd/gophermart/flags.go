package main

import (
	"flag"
	"github.com/caarlos0/env"
	"log"
)

type flags struct {
	RunAddr string `.env:"ADDRESS"`
	Dsn     string `.env:"DATABASE_DSN"`
}

func parseFlags() *flags {
	f := flags{
		RunAddr: ":8080",
		Dsn:     "postgres://dev:dev@localhost:5432/dev",
	}

	flag.StringVar(&f.RunAddr, "a", f.RunAddr, "address and port to run server")
	flag.StringVar(&f.Dsn, "d", f.Dsn, "database connection string")

	flag.Parse()

	err := env.Parse(&f)

	if err != nil {
		log.Fatal(err)
	}

	return &f
}
