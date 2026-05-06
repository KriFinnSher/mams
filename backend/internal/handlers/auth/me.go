package auth

import (
	"errors"
	"net/http"

	authmw "github.com/mams/backend/internal/middleware/auth"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/utils"
)

type meResponse struct {
	ID             string `json:"id"`
	Login          string `json:"login"`
	OrganizationID string `json:"organization_id"`
}

func (h *LoginHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := authmw.ClaimsFromContext(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, postgresrepo.ErrUserNotFound) {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, meResponse{
		ID:             user.ID.String(),
		Login:          user.Login,
		OrganizationID: user.OrganizationID.String(),
	})
}
