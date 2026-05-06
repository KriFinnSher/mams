package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mams/backend/internal/utils"
)

func RequireAuth(validator *JWTValidator, baseLogger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("Authorization")
		if raw == "" {
			utils.WriteError(w, http.StatusUnauthorized, "authorization header is required")
			return
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(raw, prefix) {
			utils.WriteError(w, http.StatusUnauthorized, "invalid authorization scheme")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
		if token == "" {
			utils.WriteError(w, http.StatusUnauthorized, "token is required")
			return
		}

		claims, err := validator.Validate(token)
		if err != nil {
			if errors.Is(err, ErrInvalidToken) {
				utils.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			utils.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		l := baseLogger.With(
			slog.String("user_id", claims.UserID.String()),
			slog.String("organization_id", claims.OrganizationID.String()),
		)
		ctx := WithClaims(r.Context(), claims)
		ctx = WithLogger(ctx, l)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
