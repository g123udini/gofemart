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
	"math"
	"net/http"
	"strings"
	"time"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrUserNotFound = errors.New("user not found")
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
		Balance: model.Balance{
			Current:   0,
			Withdrawn: 0,
		},
	}

	err := handler.repo.SaveUser(&u)

	if err != nil {
		if errors.Is(err, repository.ErrUniqConstrait) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	handler.startSession(&u, w)

	w.WriteHeader(http.StatusOK)
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

	handler.startSession(u, w)

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

func (handler *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
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

	orders, err := handler.repo.GetWithdrawalsByUser(user)

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

func (handler *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	user, err := handler.getUser(r)
	if errors.Is(err, ErrUnauthorized) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if errors.Is(err, ErrUserNotFound) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(user.Balance); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (handler *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"` // рубли с копейками
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&input); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if !service.ValidLun(input.Order) {
		http.Error(w, "order number not valid", http.StatusUnprocessableEntity)
		return
	}

	if input.Sum <= 0 {
		http.Error(w, "sum must be positive", http.StatusBadRequest)
		return
	}

	user, err := handler.getUser(r)
	if errors.Is(err, ErrUnauthorized) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if errors.Is(err, ErrUserNotFound) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sumCents := int(math.Round(input.Sum * 100))

	newBalance := user.Balance.Current - sumCents
	if newBalance < 0 {
		http.Error(w, "Insufficient balance", http.StatusPaymentRequired)
		return
	}

	user.Balance.Current = newBalance
	user.Balance.Withdrawn += sumCents

	withdrawal := model.Withdrawal{
		Number: input.Order,
		Sum:    sumCents, // копейки
		UserID: user.ID,
	}

	if err := handler.repo.SaveWithdrawal(&withdrawal); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := handler.repo.UpdateUser(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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

	user, err := handler.getUser(r)
	if errors.Is(err, ErrUnauthorized) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if errors.Is(err, ErrUserNotFound) {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		UserID:     user.ID,
	}

	if err = handler.repo.SaveOrder(order); err != nil {
		if errors.Is(err, repository.ErrUniqConstrait) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
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

func (handler *Handler) startSession(u *model.User, w http.ResponseWriter) {
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
}

func (handler *Handler) getUser(r *http.Request) (*model.User, error) {
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		return nil, ErrUnauthorized
	}

	login, ok := handler.ms.GetSession(cookie.Value)
	if !ok {
		return nil, ErrUnauthorized
	}

	user, err := handler.repo.GetUserByLogin(login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}
