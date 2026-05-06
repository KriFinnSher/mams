package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/mams/backend/internal/auth"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

type LoginHandler struct {
	users  UserReader
	issuer TokenIssuer
	log    *slog.Logger
}

func NewLoginHandler(users UserReader, issuer TokenIssuer, log *slog.Logger) *LoginHandler {
	return &LoginHandler{users: users, issuer: issuer, log: log}
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
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Login == "" || req.Password == "" {
		utils.WriteError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	user, err := h.users.GetByLogin(r.Context(), req.Login)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrUserNotFound) {
			utils.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		h.log.Error("get user by login failed", "err", err, "login", req.Login)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		if errors.Is(err, auth.ErrInvalidPassword) {
			utils.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		h.log.Error("verify password failed", "err", err, "login", req.Login)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	token, err := h.issuer.IssueToken(user)
	if err != nil {
		h.log.Error("issue token failed", "err", err, "user_id", user.ID)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, loginResponse{Token: token})
}
