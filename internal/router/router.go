package router

import (
	"github.com/g123udini/gofemart/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(handler *handler.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.AllowContentType("application/json"))

	routeApi(r, handler)

	return r
}

func routeApi(router chi.Router, handler *handler.Handler) {
	router.Route("/api", func(r chi.Router) {
		routeUser(r, handler)
	})
}

func routeUser(router chi.Router, handler *handler.Handler) {
	router.Route("/user", func(r chi.Router) {
		r.Get("/register", handler.Register)
		r.Get("/login", handler.Login)
		r.With(handler.SessionAuth).Get("/test", handler.Test)
	})
}
