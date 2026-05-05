package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
)

type testUserReader struct {
	user models.User
	err  error
}

func (r testUserReader) GetByLogin(_ context.Context, _ string) (models.User, error) {
	if r.err != nil {
		return models.User{}, r.err
	}
	return r.user, nil
}

type testTokenIssuer struct {
	token string
	err   error
}

func (i testTokenIssuer) IssueToken(_ models.User) (string, error) {
	if i.err != nil {
		return "", i.err
	}
	return i.token, nil
}

func TestLoginHandlerPost(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() err = %v", err)
	}

	validUser := models.User{
		Login:        "vadim",
		PasswordHash: string(hash),
	}

	tests := []struct {
		name       string
		body       any
		users      UserReader
		issuer     TokenIssuer
		wantStatus int
		wantToken  string
		wantErr    string
	}{
		{
			name:       "success",
			body:       map[string]string{"login": "vadim", "password": "secret"},
			users:      testUserReader{user: validUser},
			issuer:     testTokenIssuer{token: "jwt-token"},
			wantStatus: http.StatusOK,
			wantToken:  "jwt-token",
		},
		{
			name:       "invalid json",
			body:       "{bad-json",
			users:      testUserReader{user: validUser},
			issuer:     testTokenIssuer{token: "jwt-token"},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:       "missing login",
			body:       map[string]string{"password": "secret"},
			users:      testUserReader{user: validUser},
			issuer:     testTokenIssuer{token: "jwt-token"},
			wantStatus: http.StatusBadRequest,
			wantErr:    "login and password are required",
		},
		{
			name:       "user not found",
			body:       map[string]string{"login": "vadim", "password": "secret"},
			users:      testUserReader{err: postgresrepo.ErrUserNotFound},
			issuer:     testTokenIssuer{token: "jwt-token"},
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid credentials",
		},
		{
			name:       "wrong password",
			body:       map[string]string{"login": "vadim", "password": "wrong"},
			users:      testUserReader{user: validUser},
			issuer:     testTokenIssuer{token: "jwt-token"},
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid credentials",
		},
		{
			name:       "token issue failure",
			body:       map[string]string{"login": "vadim", "password": "secret"},
			users:      testUserReader{user: validUser},
			issuer:     testTokenIssuer{err: errors.New("issue error")},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				var marshalErr error
				body, marshalErr = json.Marshal(v)
				if marshalErr != nil {
					t.Fatalf("json.Marshal() err = %v", marshalErr)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			h := NewLoginHandler(tt.users, tt.issuer)

			h.Post(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			var got map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("json.Unmarshal() err = %v", err)
			}

			if tt.wantToken != "" && got["token"] != tt.wantToken {
				t.Fatalf("token = %q, want %q", got["token"], tt.wantToken)
			}
			if tt.wantErr != "" && got["error"] != tt.wantErr {
				t.Fatalf("error = %q, want %q", got["error"], tt.wantErr)
			}
		})
	}
}
