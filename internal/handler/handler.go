package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/g123udini/gofemart/internal/model"
	"github.com/g123udini/gofemart/internal/repository"
	"github.com/g123udini/gofemart/internal/service"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	repo *repository.Repo
	ms   *service.MemSessionStorage
}

func NewHandler(repository *repository.Repo, ms *service.MemSessionStorage) *Handler {
	return &Handler{
		repo: repository,
		ms:   ms,
	}
}

func (handler *Handler) Test(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (handler *Handler) Register(w http.ResponseWriter, r *http.Request) {
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

	err := handler.repo.SaveUser(u)

	if err != nil {
		if errors.Is(err, repository.UniqConstraitErr) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (handler *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	login, _ := handler.ms.GetSession(cookie.Value)
	user, err := handler.repo.GetUserByLogin(login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "user not found", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orders, err := handler.repo.GetOrdersByUser(user)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "orders not found", http.StatusNoContent)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(orders); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (handler *Handler) AddOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	orderNumber := strings.TrimSpace(string(body))

	if !service.ValidLun(orderNumber) {
		http.Error(w, "order number not valid", http.StatusUnprocessableEntity)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	login, _ := handler.ms.GetSession(cookie.Value)
	user, err := handler.repo.GetUserByLogin(login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "user not found", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	existing, err := handler.repo.GetOrderByNumberUser(orderNumber, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("exists"))
		return
	}

	order := &model.Order{
		Number:     orderNumber,
		Status:     "NEW",
		Accrual:    0,
		UploadedAt: time.Now(),
		UserId:     user.Id,
	}

	if err = handler.repo.SaveOrder(order); err != nil {
		if errors.Is(err, repository.UniqConstraitErr) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("ok"))
}

func (handler *Handler) Login(w http.ResponseWriter, r *http.Request) {
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

	u, err := handler.repo.GetUserByLogin(input.Login)
	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)

	if err != nil {
		if errors.Is(err, repository.ErrNotFound) || u.Password != string(hash) {
			http.Error(w, "Wrong pass or login", http.StatusUnauthorized)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionID, err := NewSessionID()
	handler.ms.AddSession(sessionID, u.Login)

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

func (handler *Handler) SessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		_, ok := handler.ms.GetSession(c.Value)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
