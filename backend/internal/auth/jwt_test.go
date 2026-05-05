package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/mams/backend/internal/models"
)

func TestNewJWTIssuer(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		ttl     time.Duration
		wantErr error
	}{
		{name: "ok", secret: "secret", ttl: time.Hour},
		{name: "empty secret", secret: "", ttl: time.Hour, wantErr: ErrEmptyJWTSecret},
		{name: "zero ttl uses default", secret: "secret", ttl: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer, err := NewJWTIssuer(tt.secret, tt.ttl)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewJWTIssuer() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewJWTIssuer() err = %v", err)
			}
			if issuer == nil {
				t.Fatalf("NewJWTIssuer() issuer is nil")
			}
		})
	}
}

func TestJWTIssuerIssueToken(t *testing.T) {
	issuer, err := NewJWTIssuer("secret", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTIssuer() err = %v", err)
	}
	issuer.now = func() time.Time {
		return time.Unix(1700000000, 0).UTC()
	}

	user := models.User{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
	}

	token, err := issuer.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken() err = %v", err)
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	parsed, err := parser.Parse(token, func(token *jwt.Token) (any, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse() err = %v", err)
	}
	if !parsed.Valid {
		t.Fatalf("token is not valid")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("claims type = %T, want jwt.MapClaims", parsed.Claims)
	}

	if claims["sub"] != user.ID.String() {
		t.Fatalf("sub = %v, want %v", claims["sub"], user.ID.String())
	}
	if claims["org"] != user.OrganizationID.String() {
		t.Fatalf("org = %v, want %v", claims["org"], user.OrganizationID.String())
	}
	if claims["iat"] != float64(1700000000) {
		t.Fatalf("iat = %v, want %v", claims["iat"], float64(1700000000))
	}
	if claims["exp"] != float64(1700003600) {
		t.Fatalf("exp = %v, want %v", claims["exp"], float64(1700003600))
	}
}
