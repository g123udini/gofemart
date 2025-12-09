package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/g123udini/gofemart/internal/model"
	"github.com/g123udini/gofemart/internal/repository"
	"github.com/g123udini/gofemart/internal/service"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

type Handler struct {
	repo *repository.Repo
	ms   *service.MemStorage
}

func NewHandler(repository *repository.Repo, ms *service.MemStorage) *Handler {
	return &Handler{
		repo: repository,
		ms:   ms,
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

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
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

	u, err := h.repo.GetUserByLogin(input.Login)
	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	if err != nil {
		if errors.Is(err, repository.ErrNotFoundUser) || u.Password != string(hash) {
			http.Error(w, "Wrong pass or login", http.StatusUnauthorized)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionID, err := NewSessionID()
	h.ms.AddSession(sessionID, u.Login)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func NewSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

//func (handler *Handler) SessionAuth(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		c, err := r.Cookie("session_id")
//		if err != nil {
//			http.Error(w, "unauthorized", http.StatusUnauthorized)
//			return
//		}
//
//		// load user by session_id from storage
//		//userID, err := storage.GetUserID(c.Value)
//		if err != nil {
//			http.Error(w, "unauthorized", http.StatusUnauthorized)
//			return
//		}
//
//		next.ServeHTTP(w, r)
//	})
//}
