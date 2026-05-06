package logx

import (
	"context"
	"log/slog"

	authmw "github.com/mams/backend/internal/middleware/auth"
)

type Logger struct {
	base *slog.Logger
}

func New(base *slog.Logger) *Logger {
	return &Logger{base: base}
}

func (l *Logger) ErrorCtx(ctx context.Context, msg string, args ...any) {
	logger := l.base
	if claims, ok := authmw.ClaimsFromContext(ctx); ok {
		logger = logger.With(
			slog.String("user_id", claims.UserID.String()),
			slog.String("organization_id", claims.OrganizationID.String()),
		)
	}
	logger.Error(msg, args...)
}
