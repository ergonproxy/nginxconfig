package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type (
	accessLogPathKey  struct{}
	accessLogLevelKey struct{}
	accessLogFormat   struct{}
	ngxLoggerKey      struct{}
)

const defaultLogFormat = `$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"`

type logFormat struct {
	name     string
	template string
	escape   string
}

func (l *logFormat) defaults() {
	l.name = "combined"
	l.template = defaultLogFormat
	l.escape = "default"
}

func (l *logFormat) load(r *rule) {
	if len(r.args) > 0 {
		l.name = r.args[0]
		if len(r.args) > 1 {
			switch r.args[1] {
			case "default", "json", "none":
				l.escape = r.args[1]
				if len(r.args) > 2 {
					l.template = strings.Join(r.args[2:], "")
				}
			default:
				l.template = strings.Join(r.args[1:], "")
			}
		}
	}
}

type ngxLogger interface {
	Println(file string, level string, message []byte)
}

func errorLog(ctx context.Context, err error) {
	//TODO: implement
}

func accessLog(next echo.HandlerFunc) echo.HandlerFunc {
	return func(echoCtx echo.Context) error {
		ctx := echoCtx.Request().Context()
		dest := ctx.Value(accessLogPathKey{})
		if dest == nil {
			// access log wasn't set so it means it is disabled
			return next(echoCtx)
		}
		destPath := dest.(string)
		if destPath == "" || destPath == "/dev/null" {
			return next(echoCtx)
		}
		level := "info"
		if v := ctx.Value(accessLogLevelKey{}); v != nil {
			level = v.(string)
		}
		if v := ctx.Value(ngxLoggerKey{}); v != nil {
			ngx := v.(ngxLogger)
			start := time.Now()
			if err := next(echoCtx); err != nil {
				echoCtx.Error(err)
			}
			duration := time.Since(start)
			m := ctx.Value(variables{}).(*sync.Map)
			m.Store(vRequestTime, duration.Milliseconds())
			format := defaultLogFormat
			if f := ctx.Value(accessLogFormat{}); f != nil {
				format = f.(string)
			}
			ngx.Println(destPath, level, resolveVariables(m, []byte(format)))
		}
		return nil
	}
}
