package engine

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

func log(ctx context.Context) *zap.Logger {
	return ctx.Value(loggerKey{}).(*zap.Logger)
}

func withLog(ctx context.Context, lg *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, lg)
}

func newLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}
