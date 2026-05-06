package services

import (
	"errors"
	"testing"
)

func TestCreateRequestValidate(t *testing.T) {
	t.Parallel()

	valid := createRequest{
		Name:                       "user-service",
		Description:                "desc",
		Type:                       "business",
		TestCoverage:               80,
		MinimumTestCoverageEnabled: true,
		MinimumTestCoverage:        70,
		PIISensitive:               true,
		ResponsibleTeamRef:         "@infra-team",
		Importance:                 "high",
		RepositoryURL:              "https://github.com/org/user-service",
		DefaultBranch:              "main",
		GrafanaDashboardUID:        "uid123",
	}

	tests := []struct {
		name    string
		req     createRequest
		wantErr error
	}{
		{
			name: "valid",
			req:  valid,
		},
		{
			name:    "missing name",
			req:     createRequest{Type: "business", TestCoverage: 1, MinimumTestCoverage: 1, Importance: "low", RepositoryURL: "https://github.com/org/repo", DefaultBranch: "main"},
			wantErr: errNameRequired,
		},
		{
			name:    "invalid type",
			req:     createRequest{Name: "svc", Type: "unknown", TestCoverage: 1, MinimumTestCoverage: 1, Importance: "low", RepositoryURL: "https://github.com/org/repo", DefaultBranch: "main"},
			wantErr: errInvalidType,
		},
		{
			name:    "invalid test coverage",
			req:     createRequest{Name: "svc", Type: "business", TestCoverage: 101, MinimumTestCoverage: 1, Importance: "low", RepositoryURL: "https://github.com/org/repo", DefaultBranch: "main"},
			wantErr: errInvalidCoverage,
		},
		{
			name:    "invalid minimum coverage",
			req:     createRequest{Name: "svc", Type: "business", TestCoverage: 80, MinimumTestCoverage: -1, Importance: "low", RepositoryURL: "https://github.com/org/repo", DefaultBranch: "main"},
			wantErr: errInvalidMinCoverage,
		},
		{
			name:    "invalid importance",
			req:     createRequest{Name: "svc", Type: "business", TestCoverage: 80, MinimumTestCoverage: 40, Importance: "urgent", RepositoryURL: "https://github.com/org/repo", DefaultBranch: "main"},
			wantErr: errInvalidImportance,
		},
		{
			name:    "missing repository url",
			req:     createRequest{Name: "svc", Type: "business", TestCoverage: 80, MinimumTestCoverage: 40, Importance: "low", DefaultBranch: "main"},
			wantErr: errRepositoryURLMissing,
		},
		{
			name:    "invalid repository url",
			req:     createRequest{Name: "svc", Type: "business", TestCoverage: 80, MinimumTestCoverage: 40, Importance: "low", RepositoryURL: "bad", DefaultBranch: "main"},
			wantErr: errRepositoryURLInvalid,
		},
		{
			name:    "missing default branch",
			req:     createRequest{Name: "svc", Type: "business", TestCoverage: 80, MinimumTestCoverage: 40, Importance: "low", RepositoryURL: "https://github.com/org/repo"},
			wantErr: errDefaultBranchMissing,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.req.validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("validate() err = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validate() err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
