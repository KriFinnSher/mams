package services

import (
	"errors"
	"net/url"
	"strings"
)

var (
	errNameRequired         = errors.New("name is required")
	errInvalidType          = errors.New("type must be one of: business, composition")
	errInvalidCoverage      = errors.New("test_coverage must be between 0 and 100")
	errInvalidMinCoverage   = errors.New("minimum_test_coverage must be between 0 and 100")
	errInvalidImportance    = errors.New("importance must be one of: low, medium, high, critical")
	errRepositoryURLMissing = errors.New("repository_url is required")
	errRepositoryURLInvalid = errors.New("repository_url must be a valid url")
	errDefaultBranchMissing = errors.New("default_branch is required")
)

type createRequest struct {
	Name                       string `json:"name"`
	Description                string `json:"description"`
	Type                       string `json:"type"`
	TestCoverage               int    `json:"test_coverage"`
	MinimumTestCoverageEnabled bool   `json:"minimum_test_coverage_enabled"`
	MinimumTestCoverage        int    `json:"minimum_test_coverage"`
	PIISensitive               bool   `json:"pii_sensitive"`
	ResponsibleTeamRef         string `json:"responsible_team_ref"`
	Importance                 string `json:"importance"`
	RepositoryURL              string `json:"repository_url"`
	DefaultBranch              string `json:"default_branch"`
	GrafanaDashboardUID        string `json:"grafana_dashboard_uid"`
}

func (r createRequest) validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errNameRequired
	}
	if r.Type != "business" && r.Type != "composition" {
		return errInvalidType
	}
	if r.TestCoverage < 0 || r.TestCoverage > 100 {
		return errInvalidCoverage
	}
	if r.MinimumTestCoverage < 0 || r.MinimumTestCoverage > 100 {
		return errInvalidMinCoverage
	}
	if r.Importance != "low" && r.Importance != "medium" && r.Importance != "high" && r.Importance != "critical" {
		return errInvalidImportance
	}
	if strings.TrimSpace(r.RepositoryURL) == "" {
		return errRepositoryURLMissing
	}
	u, err := url.ParseRequestURI(r.RepositoryURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errRepositoryURLInvalid
	}
	if strings.TrimSpace(r.DefaultBranch) == "" {
		return errDefaultBranchMissing
	}

	return nil
}
