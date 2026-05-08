package services

import "github.com/mams/backend/internal/models"

type ServiceCardDTO struct {
	ID             string         `json:"id"`
	Overview       ServiceOverview `json:"overview"`
	Settings       ServiceSettings `json:"settings"`
	Modules        ServiceModules  `json:"modules"`
}

type ServiceOverview struct {
	OrganizationID      string `json:"organization_id"`
	CreatedByUserID     string `json:"created_by_user_id"`
	OwnerUserID         string `json:"owner_user_id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Type                string `json:"type"`
	Version             string `json:"version"`
	TestCoverage        int    `json:"test_coverage"`
	PIISensitive        bool   `json:"pii_sensitive"`
	ResponsibleTeamRef  string `json:"responsible_team_ref"`
	Importance          string `json:"importance"`
}

type ServiceSettings struct {
	MinimumTestCoverageEnabled bool           `json:"minimum_test_coverage_enabled"`
	MinimumTestCoverage        int            `json:"minimum_test_coverage"`
	Settings                   map[string]any `json:"settings"`
}

type ServiceModules struct {
	RepositoryURL      string `json:"repository_url"`
	DefaultBranch      string `json:"default_branch"`
	GrafanaDashboardUID string `json:"grafana_dashboard_uid"`
}

func toServiceCardDTO(svc models.Service) ServiceCardDTO {
	return ServiceCardDTO{
		ID: svc.ID.String(),
		Overview: ServiceOverview{
			OrganizationID:     svc.OrganizationID.String(),
			CreatedByUserID:    svc.CreatedByUserID.String(),
			OwnerUserID:        svc.OwnerUserID.String(),
			Name:               svc.Name,
			Description:        svc.Description,
			Type:               svc.Type,
			Version:            svc.Version,
			TestCoverage:       svc.TestCoverage,
			PIISensitive:       svc.PIISensitive,
			ResponsibleTeamRef: svc.ResponsibleTeamRef,
			Importance:         svc.Importance,
		},
		Settings: ServiceSettings{
			MinimumTestCoverageEnabled: svc.MinimumTestCoverageEnabled,
			MinimumTestCoverage:        svc.MinimumTestCoverage,
			Settings:                   svc.Settings,
		},
		Modules: ServiceModules{
			RepositoryURL:       svc.RepositoryURL,
			DefaultBranch:       svc.DefaultBranch,
			GrafanaDashboardUID: svc.GrafanaDashboardUID,
		},
	}
}
