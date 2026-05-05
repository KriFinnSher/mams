package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type JWTValidator struct {
	secret []byte
}

func NewJWTValidator(secret string) (*JWTValidator, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	return &JWTValidator{secret: []byte(secret)}, nil
}

func (v *JWTValidator) Validate(raw string) (Claims, error) {
	token, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unexpected signing method", ErrInvalidToken)
		}
		return v.secret, nil
	})
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, ErrInvalidToken
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return Claims{}, ErrInvalidToken
	}
	org, ok := claims["org"].(string)
	if !ok || org == "" {
		return Claims{}, ErrInvalidToken
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	orgID, err := uuid.Parse(org)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	return Claims{
		UserID:         userID,
		OrganizationID: orgID,
	}, nil
}
