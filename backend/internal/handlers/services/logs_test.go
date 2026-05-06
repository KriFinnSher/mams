package services

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/handlers/services/mocks"
	"github.com/mams/backend/internal/logx"
	"github.com/mams/backend/internal/models"
	"github.com/mams/backend/internal/ws"
	"go.uber.org/mock/gomock"
)

func TestHandlerGetLogs(t *testing.T) {
	ctrl := gomock.NewController(t)
	sr := mocks.NewMockServiceReader(ctrl)
	lr := mocks.NewMockLogReader(ctrl)
	h := NewHandler(sr, lr, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	sid := uuid.New()
	lr.EXPECT().ListByService(gomock.Any(), sid, gomock.Any()).Return([]models.LogEntry{{ID: "1"}}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/services/"+sid.String()+"/logs?level=info&text=err&limit=10", nil)
	req.SetPathValue("id", sid.String())
	rec := httptest.NewRecorder()
	h.GetLogs(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

