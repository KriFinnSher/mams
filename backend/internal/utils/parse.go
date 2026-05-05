package utils

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

func ParseTTLSeconds(raw string, fallback int64) time.Duration {
	if raw == "" {
		return time.Duration(fallback) * time.Second
	}
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || sec <= 0 {
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(sec) * time.Second
}

func MustUUID(v string) uuid.UUID {
	id, err := uuid.Parse(v)
	if err != nil {
		panic(err)
	}
	return id
}
