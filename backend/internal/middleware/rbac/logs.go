package rbac

import (
	"errors"
	"net/http"
	rbaccore "github.com/mams/backend/internal/rbac"
)

var ErrAccessNotFound = errors.New("service access not found")

func RequireLogsAccess(services serviceReader, access accessReader, next http.Handler) http.Handler {
	return RequireAtLeastRole(rbaccore.RoleDeveloper, services, access, next)
}
