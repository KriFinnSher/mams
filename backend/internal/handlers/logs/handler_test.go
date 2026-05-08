package logs

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/logx"
	"github.com/mams/backend/internal/handlers/logs/mocks"
	"github.com/mams/backend/internal/models"
	"github.com/mams/backend/internal/ws"
	"go.uber.org/mock/gomock"
)

func TestGet_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	lr := mocks.NewMockReader(ctrl)
	svc := mocks.NewMockServiceGetter(ctrl)
	orgs := mocks.NewMockOrgGetter(ctrl)
	sid := uuid.New()
	lr.EXPECT().ListByService(gomock.Any(), sid, gomock.Any()).Return([]models.LogEntry{{ServiceID: sid, Message: "ok"}}, nil)
	h := NewHandler(lr, nil, svc, orgs, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/logs", nil)
	req.SetPathValue("id", sid.String())
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestParseLogFilter_InvalidLimit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/services/a/logs?limit=0", nil)
	if _, err := parseLogFilter(req); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIngest_InvalidPayload(t *testing.T) {
	ctrl := gomock.NewController(t)
	lr := mocks.NewMockReader(ctrl)
	svc := mocks.NewMockServiceGetter(ctrl)
	orgs := mocks.NewMockOrgGetter(ctrl)
	h := NewHandler(lr, nil, svc, orgs, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	req := httptest.NewRequest(http.MethodPost, "/api/internal/services/"+uuid.New().String()+"/logs", nil)
	req.SetPathValue("id", uuid.New().String())
	rec := httptest.NewRecorder()
	h.Ingest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestIngest_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	lr := mocks.NewMockReader(ctrl)
	svc := mocks.NewMockServiceGetter(ctrl)
	orgs := mocks.NewMockOrgGetter(ctrl)
	sid := uuid.New()
	lr.EXPECT().Append(gomock.Any(), sid, "dev", "info", "hi").Return(&models.LogEntry{ServiceID: sid})
	h := NewHandler(lr, nil, svc, orgs, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	req := httptest.NewRequest(http.MethodPost, "/api/internal/services/"+sid.String()+"/logs", strings.NewReader(`[{"environment":"dev","level":"info","message":"hi"}]`))
	req.SetPathValue("id", sid.String())
	rec := httptest.NewRecorder()
	h.Ingest(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d", rec.Code)
	}
}
