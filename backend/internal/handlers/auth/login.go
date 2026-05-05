package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mams/backend/internal/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
)

type UserReader interface {
	GetByLogin(ctx context.Context, login string) (models.User, error)
}

type TokenIssuer interface {
	IssueToken(user models.User) (string, error)
}

type LoginHandler struct {
	users  UserReader
	issuer TokenIssuer
}

func NewLoginHandler(users UserReader, issuer TokenIssuer) *LoginHandler {
	return &LoginHandler{users: users, issuer: issuer}
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *LoginHandler) Post(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Login == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	user, err := h.users.GetByLogin(r.Context(), req.Login)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		if errors.Is(err, auth.ErrInvalidPassword) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	token, err := h.issuer.IssueToken(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
