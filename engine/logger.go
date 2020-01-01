package engine

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

func log(ctx context.Context) *zap.Logger {
	lg := ctx.Value(loggerKey{}).(*zap.Logger)
	if v := ctx.Value(errLogInfo{}); v != nil {
		i := v.(*ErrorLog)
		return lg.With(
			zap.String("error_log_file", i.Name),
			zap.String("error_log_level", i.Level),
		)
	}
	return lg
}

func withLog(ctx context.Context, lg *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, lg)
}

func newLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}
