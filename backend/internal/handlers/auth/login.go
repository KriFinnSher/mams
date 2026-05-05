package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mams/backend/internal/auth"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

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
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		if errors.Is(err, auth.ErrInvalidPassword) {
			utils.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	token, err := h.issuer.IssueToken(user)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, loginResponse{Token: token})
}
