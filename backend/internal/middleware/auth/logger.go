package auth

import (
	"context"
	"log/slog"
)

func LoggerFromContext(ctx context.Context, fallback *slog.Logger) *slog.Logger {
	if l, ok := loggerFromContext(ctx); ok {
		return l
	}
	return fallback
}

