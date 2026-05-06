package rbac

import (
	"net/http"

	rbaccore "github.com/mams/backend/internal/rbac"
)

func RequireMetricsAccess(services serviceReader, access accessReader, next http.Handler) http.Handler {
	return RequireAtLeastRole(rbaccore.RoleDeveloper, services, access, next)
}
