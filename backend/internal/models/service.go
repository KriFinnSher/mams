package models

import (
	"time"

	"github.com/google/uuid"
)

type Service struct {
	ID                         uuid.UUID
	OrganizationID             uuid.UUID
	CreatedByUserID            uuid.UUID
	OwnerUserID                uuid.UUID
	Name                       string
	Description                string
	Type                       string
	Version                    string
	TestCoverage               int
	MinimumTestCoverageEnabled bool
	MinimumTestCoverage        int
	PIISensitive               bool
	ResponsibleTeamRef         string
	Importance                 string
	RepositoryURL              string
	DefaultBranch              string
	GrafanaDashboardUID        string
	Settings                   map[string]any
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}
