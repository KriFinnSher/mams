package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/mams/backend/internal/models"
)

func (h *Handler) createPendingRelease(
	ctx context.Context,
	serviceID, authorUserID uuid.UUID,
	gitTag, branch, environment, strategy, description string,
) (models.Release, error) {
	return h.releases.Create(ctx, models.Release{
		ServiceID:    serviceID,
		GitTag:       gitTag,
		Branch:       branch,
		Environment:  environment,
		Strategy:     strategy,
		Status:       "pending",
		Description:  description,
		AuthorUserID: authorUserID,
	})
}

