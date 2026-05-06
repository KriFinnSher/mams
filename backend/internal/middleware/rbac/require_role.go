package rbac

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	authmw "github.com/mams/backend/internal/middleware/auth"
	rbaccore "github.com/mams/backend/internal/rbac"
	"github.com/mams/backend/internal/utils"
)

func RequireAtLeastRole(minRole string, services serviceReader, access accessReader, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := authmw.ClaimsFromContext(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		serviceID, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			utils.WriteError(w, http.StatusBadRequest, "invalid service id")
			return
		}

		svc, err := services.GetByID(r.Context(), serviceID)
		if err != nil || svc.OrganizationID != claims.OrganizationID {
			utils.WriteError(w, http.StatusNotFound, "service not found")
			return
		}

		role := rbaccore.RoleObserver
		a, err := access.GetByServiceAndUser(r.Context(), serviceID, claims.UserID)
		if err == nil {
			role = rbaccore.EffectiveRole(svc.OwnerUserID, claims.UserID, a.Role)
		} else if errors.Is(err, ErrAccessNotFound) {
			role = rbaccore.EffectiveRole(svc.OwnerUserID, claims.UserID, "")
		} else {
			utils.WriteError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if !hasAtLeastRole(role, minRole) {
			utils.WriteError(w, http.StatusForbidden, "forbidden")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func hasAtLeastRole(role, minRole string) bool {
	return roleWeight(role) >= roleWeight(minRole)
}

func roleWeight(role string) int {
	switch role {
	case rbaccore.RoleObserver:
		return 0
	case rbaccore.RoleDeveloper:
		return 1
	case rbaccore.RoleServiceOwner:
		return 2
	default:
		return -1
	}
}

type serviceReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (serviceView, error)
}

type accessReader interface {
	GetByServiceAndUser(ctx context.Context, serviceID, userID uuid.UUID) (accessView, error)
}

type serviceView struct {
	OrganizationID uuid.UUID
	OwnerUserID    uuid.UUID
}

type accessView struct {
	Role string
}
