package utils

import "testing"

func TestBuildNamespace(t *testing.T) {
	tests := []struct {
		name string
		org  string
		env  string
		want string
	}{
		{name: "both values", org: "acme", env: "prod", want: "acme-prod"},
		{name: "trim spaces", org: " acme ", env: " prod ", want: "acme-prod"},
		{name: "no env", org: "acme", env: "", want: "acme"},
		{name: "no org", org: "", env: "prod", want: "prod"},
		{name: "both empty", org: "", env: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildNamespace(tt.org, tt.env)
			if got != tt.want {
				t.Fatalf("BuildNamespace() = %q, want %q", got, tt.want)
			}
		})
	}
}
