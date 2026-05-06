package mongo

import (
	"fmt"
	"time"
)

func toID(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func unixToTime(v int64) time.Time {
	if v <= 0 {
		return time.Unix(0, 0).UTC()
	}
	return time.Unix(v, 0).UTC()
}
