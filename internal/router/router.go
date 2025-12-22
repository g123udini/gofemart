package router

import (
	"github.com/g123udini/gofemart/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *handler.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	routeAPI(r, handler)

	return r
}

func routeAPI(router chi.Router, handler *handler.Handler) {
	router.Route("/api", func(r chi.Router) {
		routeUser(r, handler)
	})
}

func routeUser(router chi.Router, handler *handler.Handler) {
	router.Route("/user", func(r chi.Router) {
		r.Post("/register", handler.Register)
		r.Post("/login", handler.Login)
		r.
			With(middleware.AllowContentType("text/plain")).
			With(handler.SessionAuth).
			Post("/orders", handler.AddOrder)

		r.
			With(middleware.AllowContentType("application/json")).
			With(handler.SessionAuth).
			Get("/orders", handler.GetOrder)

		r.
			With(middleware.AllowContentType("application/json")).
			With(handler.SessionAuth).
			Get("/withdrawals", handler.GetOrder)

		r.Route("/balance", func(br chi.Router) {
			br.
				With(middleware.AllowContentType("application/json")).
				With(handler.SessionAuth).
				Get("/", handler.GetBalance)

			br.
				With(middleware.AllowContentType("application/json")).
				With(handler.SessionAuth).
				Post("/withdraw", handler.Withdraw)

		})
	})
}
