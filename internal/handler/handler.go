package handler

import (
	"github.com/g123udini/gofemart/internal/repository"
	"go.uber.org/zap"
	"net/http"
)

type Handler struct {
	logger     *zap.SugaredLogger
	repository *repository.Repo
}

func NewHandler(logger *zap.SugaredLogger, repository *repository.Repo) *Handler {
	return &Handler{
		logger:     logger,
		repository: repository,
	}
}

func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello World"))
}
