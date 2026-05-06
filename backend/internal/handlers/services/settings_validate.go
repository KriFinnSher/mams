package services

import "errors"

var errInvalidSettings = errors.New("invalid settings")

func touchesOwnerOnlySettings(settings map[string]any) bool {
	_, a := settings["minimum_test_coverage_enabled"]
	_, b := settings["minimum_test_coverage"]
	return a || b
}

func validateSettingsPatch(settings map[string]any) error {
	if v, ok := settings["minimum_test_coverage_enabled"]; ok {
		if _, ok := v.(bool); !ok {
			return errInvalidSettings
		}
	}
	if v, ok := settings["minimum_test_coverage"]; ok {
		n, ok := asInt(v)
		if !ok || n < 0 || n > 100 {
			return errInvalidSettings
		}
		settings["minimum_test_coverage"] = n
	}
	return nil
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), float64(int(n)) == n
	case int:
		return n, true
	default:
		return 0, false
	}
}

