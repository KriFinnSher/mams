package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/mams/backend/internal/models"
)

var ErrEmptyJWTSecret = errors.New("jwt secret is empty")

type JWTIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewJWTIssuer(secret string, ttl time.Duration) (*JWTIssuer, error) {
	if secret == "" {
		return nil, ErrEmptyJWTSecret
	}
	if ttl <= 0 {
		ttl = time.Hour
	}

	return &JWTIssuer{
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}, nil
}

func (i *JWTIssuer) IssueToken(user models.User) (string, error) {
	now := i.now()
	claims := jwt.MapClaims{
		"sub": user.ID.String(),
		"org": user.OrganizationID.String(),
		"iat": now.Unix(),
		"exp": now.Add(i.ttl).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}
