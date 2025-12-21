package main

import (
	"flag"
	"github.com/caarlos0/env"
	"log"
)

type flags struct {
	RunAddr        string `env:"RUN_ADDRESS"`
	Dsn            string `env:"DATABASE_URI"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func parseFlags() *flags {
	f := flags{
		RunAddr:        ":8080",
		Dsn:            "postgres://dev:dev@localhost:5432/dev",
		AccrualAddress: "http://localhost:8080/accrual",
	}

	flag.StringVar(&f.RunAddr, "a", f.RunAddr, "address and port to run server")
	flag.StringVar(&f.Dsn, "d", f.Dsn, "database connection string")
	flag.StringVar(&f.AccrualAddress, "r", f.AccrualAddress, "accrual service connection string")

	flag.Parse()

	err := env.Parse(&f)

	if err != nil {
		log.Fatal(err)
	}

	return &f
}
