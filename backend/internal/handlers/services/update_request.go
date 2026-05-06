package services

func validateUpdateInfo(r updateInfoRequest) error {
	req := createRequest{
		Name:               "stub",
		Type:               r.Type,
		TestCoverage:       r.TestCoverage,
		MinimumTestCoverage: 0,
		Importance:         r.Importance,
		RepositoryURL:      r.RepositoryURL,
		DefaultBranch:      r.DefaultBranch,
	}
	return req.validate()
}

