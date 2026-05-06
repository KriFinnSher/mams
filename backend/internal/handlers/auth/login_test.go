package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/mams/backend/internal/handlers/auth/mocks"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
)

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
		prepare    func(users *mocks.MockUserReader, issuer *mocks.MockTokenIssuer)
		wantStatus int
		wantToken  string
		wantErr    string
	}{
		{
			name:       "success",
			body:       map[string]string{"login": "vadim", "password": "secret"},
			prepare: func(users *mocks.MockUserReader, issuer *mocks.MockTokenIssuer) {
				users.EXPECT().GetByLogin(gomock.Any(), "vadim").Return(validUser, nil)
				issuer.EXPECT().IssueToken(validUser).Return("jwt-token", nil)
			},
			wantStatus: http.StatusOK,
			wantToken:  "jwt-token",
		},
		{
			name:       "invalid json",
			body:       "{bad-json",
			prepare:    func(_ *mocks.MockUserReader, _ *mocks.MockTokenIssuer) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    "invalid request body",
		},
		{
			name:       "missing login",
			body:       map[string]string{"password": "secret"},
			prepare:    func(_ *mocks.MockUserReader, _ *mocks.MockTokenIssuer) {},
			wantStatus: http.StatusBadRequest,
			wantErr:    "login and password are required",
		},
		{
			name:       "user not found",
			body:       map[string]string{"login": "vadim", "password": "secret"},
			prepare: func(users *mocks.MockUserReader, _ *mocks.MockTokenIssuer) {
				users.EXPECT().GetByLogin(gomock.Any(), "vadim").Return(models.User{}, postgresrepo.ErrUserNotFound)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid credentials",
		},
		{
			name:       "wrong password",
			body:       map[string]string{"login": "vadim", "password": "wrong"},
			prepare: func(users *mocks.MockUserReader, _ *mocks.MockTokenIssuer) {
				users.EXPECT().GetByLogin(gomock.Any(), "vadim").Return(validUser, nil)
			},
			wantStatus: http.StatusUnauthorized,
			wantErr:    "invalid credentials",
		},
		{
			name:       "token issue failure",
			body:       map[string]string{"login": "vadim", "password": "secret"},
			prepare: func(users *mocks.MockUserReader, issuer *mocks.MockTokenIssuer) {
				users.EXPECT().GetByLogin(gomock.Any(), "vadim").Return(validUser, nil)
				issuer.EXPECT().IssueToken(validUser).Return("", errors.New("issue error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			users := mocks.NewMockUserReader(ctrl)
			issuer := mocks.NewMockTokenIssuer(ctrl)
			tt.prepare(users, issuer)

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
			h := NewLoginHandler(users, issuer, slog.New(slog.NewTextHandler(io.Discard, nil)))

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
