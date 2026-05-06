package rbac

import "net/http"

func RequireMetricsAccess(services serviceReader, access accessReader, next http.Handler) http.Handler {
	return RequireLogsAccess(services, access, next)
}

