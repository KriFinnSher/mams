package auth

import (
	"context"

	"github.com/google/uuid"
)

type Claims struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID
}

type contextKey string

const claimsKey contextKey = "auth_claims"

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
