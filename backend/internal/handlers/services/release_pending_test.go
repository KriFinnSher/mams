package services

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/handlers/services/mocks"
	"github.com/mams/backend/internal/logx"
	"github.com/mams/backend/internal/models"
	"github.com/mams/backend/internal/ws"
	"go.uber.org/mock/gomock"
)

func TestCreatePendingRelease(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	serviceID := uuid.New()
	authorID := uuid.New()
	releases := mocks.NewMockReleaseReader(ctrl)
	releases.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, rel models.Release) (models.Release, error) {
			if rel.Status != "pending" {
				t.Fatalf("status = %q, want %q", rel.Status, "pending")
			}
			if rel.ServiceID != serviceID {
				t.Fatalf("service_id = %v, want %v", rel.ServiceID, serviceID)
			}
			if rel.AuthorUserID != authorID {
				t.Fatalf("author_user_id = %v, want %v", rel.AuthorUserID, authorID)
			}
			return rel, nil
		},
	)

	h := NewHandler(nil, nil, nil, releases, ws.NewHub(), logx.New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	_, err := h.createPendingRelease(context.Background(), serviceID, authorID, "v1.0.0", "", "prod", "rolling", "demo")
	if err != nil {
		t.Fatalf("createPendingRelease error = %v", err)
	}
}

