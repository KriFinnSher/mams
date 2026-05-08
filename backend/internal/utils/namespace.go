package utils

import "strings"

func BuildNamespace(organizationSlug, environment string) string {
	org := strings.TrimSpace(organizationSlug)
	env := strings.TrimSpace(environment)
	if org == "" {
		return env
	}
	if env == "" {
		return org
	}

	return org + "-" + env
}
