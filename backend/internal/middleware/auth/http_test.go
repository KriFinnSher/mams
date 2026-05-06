package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authcore "github.com/mams/backend/internal/auth"
	"github.com/mams/backend/internal/models"
	"github.com/google/uuid"
)

func TestRequireAuth(t *testing.T) {
	secret := "secret"
	issuer, err := authcore.NewJWTIssuer(secret, time.Hour)
	if err != nil {
		t.Fatalf("NewJWTIssuer() err = %v", err)
	}
	userID := uuid.New()
	orgID := uuid.New()
	token, err := issuer.IssueToken(models.User{
		ID:             userID,
		OrganizationID: orgID,
	})
	if err != nil {
		t.Fatalf("IssueToken() err = %v", err)
	}

	validator, err := NewJWTValidator(secret)
	if err != nil {
		t.Fatalf("NewJWTValidator() err = %v", err)
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantErr    string
		wantClaims bool
	}{
		{
			name:       "missing header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantErr:    "authorization header is required",
		},
		{
			name:       "invalid scheme",
			authHeader: "Token " + token,
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid authorization scheme",
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid token",
		},
		{
			name:       "valid token",
			authHeader: "Bearer " + token,
			wantStatus: http.StatusOK,
			wantClaims: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotClaims Claims
			var claimsOK bool

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotClaims, claimsOK = ClaimsFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})
			h := RequireAuth(validator, next)

			req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantErr != "" {
				var payload map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
					t.Fatalf("json.Unmarshal() err = %v", err)
				}
				if payload["error"] != tt.wantErr {
					t.Fatalf("error = %q, want %q", payload["error"], tt.wantErr)
				}
			}

			if tt.wantClaims {
				if !claimsOK {
					t.Fatalf("claims are missing in context")
				}
				if gotClaims.UserID != userID {
					t.Fatalf("claims user id = %v, want %v", gotClaims.UserID, userID)
				}
				if gotClaims.OrganizationID != orgID {
					t.Fatalf("claims org id = %v, want %v", gotClaims.OrganizationID, orgID)
				}
			}
		})
	}
}

func TestJWTValidatorValidate(t *testing.T) {
	validator, err := NewJWTValidator("secret")
	if err != nil {
		t.Fatalf("NewJWTValidator() err = %v", err)
	}

	issuer, err := authcore.NewJWTIssuer("secret", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTIssuer() err = %v", err)
	}
	validToken, err := issuer.IssueToken(models.User{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("IssueToken() err = %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{name: "valid token", token: validToken},
		{name: "invalid token", token: "bad-token", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.Validate(tt.token)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Validate() err = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() err = %v", err)
			}
		})
	}
}
