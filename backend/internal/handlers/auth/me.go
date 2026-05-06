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
	Services       []meServiceRole `json:"services"`
}

type meServiceRole struct {
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
	Role        string `json:"role"`
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
		h.log.ErrorCtx(r.Context(), "get user by id failed", "err", err)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	roles, err := h.users.ListUserNonObserverRoles(r.Context(), claims.UserID, claims.OrganizationID)
	if err != nil {
		h.log.ErrorCtx(r.Context(), "list profile service roles failed", "err", err)
		utils.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	items := make([]meServiceRole, 0, len(roles))
	for _, r := range roles {
		items = append(items, meServiceRole{
			ServiceID:   r.ServiceID.String(),
			ServiceName: r.ServiceName,
			Role:        r.Role,
		})
	}

	utils.WriteJSON(w, http.StatusOK, meResponse{
		ID:             user.ID.String(),
		Login:          user.Login,
		OrganizationID: user.OrganizationID.String(),
		Services:       items,
	})
}
