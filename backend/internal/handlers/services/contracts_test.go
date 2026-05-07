package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/githubclient"
	"github.com/mams/backend/internal/handlers/services/mocks"
	"github.com/mams/backend/internal/logx"
	authmw "github.com/mams/backend/internal/middleware/auth"
	"github.com/mams/backend/internal/models"
	postgresrepo "github.com/mams/backend/internal/repository/postgres"
	"github.com/mams/backend/internal/ws"
	"go.uber.org/mock/gomock"
)

type testProtoReader struct {
	raw []byte
	err error
}

func (r testProtoReader) ReadProjectProto(_ context.Context, _, _ string) ([]byte, error) {
	return r.raw, r.err
}

func TestHandlerGetContracts(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	serviceID := uuid.New()

	tests := []struct {
		name       string
		setupRepo  func(m *mocks.MockServiceReader)
		proto      testProtoReader
		wantStatus int
		wantErr    string
	}{
		{
			name: "success",
			setupRepo: func(m *mocks.MockServiceReader) {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{
					ID:             serviceID,
					OrganizationID: orgID,
					RepositoryURL:  "https://github.com/acme/repo",
					DefaultBranch:  "main",
				}, nil)
			},
			proto: testProtoReader{raw: []byte(`service S { rpc Get(GetReq) returns (GetRes); } message GetReq { string id = 1; }`)},
			wantStatus: http.StatusOK,
		},
		{
			name: "proto missing",
			setupRepo: func(m *mocks.MockServiceReader) {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{
					ID:             serviceID,
					OrganizationID: orgID,
					RepositoryURL:  "https://github.com/acme/repo",
					DefaultBranch:  "main",
				}, nil)
			},
			proto: testProtoReader{err: githubclient.ErrProtoNotFound},
			wantStatus: http.StatusNotFound,
			wantErr: "project.proto is missing",
		},
		{
			name: "invalid proto",
			setupRepo: func(m *mocks.MockServiceReader) {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{
					ID:             serviceID,
					OrganizationID: orgID,
					RepositoryURL:  "https://github.com/acme/repo",
					DefaultBranch:  "main",
				}, nil)
			},
			proto: testProtoReader{raw: []byte(`syntax="proto3";`)},
			wantStatus: http.StatusBadRequest,
			wantErr: "project.proto is invalid",
		},
		{
			name: "service not found",
			setupRepo: func(m *mocks.MockServiceReader) {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{}, postgresrepo.ErrServiceNotFound)
			},
			proto: testProtoReader{},
			wantStatus: http.StatusNotFound,
			wantErr: "service not found",
		},
		{
			name: "repo error",
			setupRepo: func(m *mocks.MockServiceReader) {
				m.EXPECT().GetByID(gomock.Any(), serviceID).Return(models.Service{}, errors.New("db"))
			},
			proto: testProtoReader{},
			wantStatus: http.StatusInternalServerError,
			wantErr: "internal error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockServiceReader(ctrl)
			tt.setupRepo(repo)
			h := NewHandler(repo, nil, tt.proto, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
			req := httptest.NewRequest(http.MethodGet, "/api/services/"+serviceID.String()+"/contracts", nil)
			req.SetPathValue("id", serviceID.String())
			req = req.WithContext(authmw.WithClaims(req.Context(), authmw.Claims{OrganizationID: orgID}))
			rec := httptest.NewRecorder()

			h.GetContracts(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantErr != "" {
				var body map[string]string
				_ = json.Unmarshal(rec.Body.Bytes(), &body)
				if body["error"] != tt.wantErr {
					t.Fatalf("error = %q, want %q", body["error"], tt.wantErr)
				}
			}
		})
	}
}

