package main

import (
	"fmt"
	"github.com/g123udini/gofemart/internal/handler"
	"github.com/g123udini/gofemart/internal/repository"
	"github.com/g123udini/gofemart/internal/router"
	"github.com/g123udini/gofemart/internal/service"
	"go.uber.org/zap"
	"log"
	"net"
	"net/http"
)

func main() {
	f := parseFlags()
	repo := repository.NewRepository(f.Dsn)
	logger := service.NewLogger()

	err := run(repo, logger, f)

	if err != nil {
		log.Fatal(err.Error())
	}
}

func run(repo *repository.Repo, logger *zap.SugaredLogger, f *flags) error {
	fmt.Println("Running server on", f.RunAddr)

	normalizeHost(f.RunAddr)

	h := handler.NewHandler(logger, repo)
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
