package auth

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type Claims struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
}

type contextKey string

const claimsKey contextKey = "auth_claims"
const loggerKey contextKey = "auth_logger"

func WithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	v := ctx.Value(claimsKey)
	if v == nil {
		return Claims{}, false
	}
	claims, ok := v.(Claims)
	return claims, ok
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func loggerFromContext(ctx context.Context) (*slog.Logger, bool) {
	v := ctx.Value(loggerKey)
	if v == nil {
		return nil, false
	}
	l, ok := v.(*slog.Logger)
	return l, ok
}
