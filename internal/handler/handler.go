package handler

import (
	"encoding/json"
	"errors"
	"github.com/g123udini/gofemart/internal/model"
	"github.com/g123udini/gofemart/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

type Handler struct {
	repo *repository.Repo
}

func NewHandler(repository *repository.Repo) *Handler {
	return &Handler{
		repo: repository,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&input); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	u := model.User{
		Login:    input.Login,
		Password: string(hash),
	}

	err := h.repo.SaveUser(u)

	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
