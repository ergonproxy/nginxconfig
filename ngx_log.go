package main

import (
	"context"
	"net/http"
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

var levels = map[string]int{
	"debug":  6,
	"info":   5,
	"notice": 4,
	"warn":   3,
	"error":  2,
	"crit":   1,
	"alert":  0,
}

var levelMu sync.Mutex

// returns true if level a is within level b
func withinLevel(a, b string) bool {
	levelMu.Lock()
	defer levelMu.Unlock()
	return levels[a] < levels[b]
}

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

type cacheLogger struct {
	cache *readWriterCloserCache
}

func (c *cacheLogger) Print(file string, level string, message []byte) error {
	if f, ok := c.cache.Get(file); ok {
		f.Write(message)
	} else {
		f, err := c.cache.Put(file)
		if err != nil {
			return err
		}
		f.Write(message)
	}
	return nil
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

func logMiddleware(next handler) handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		dest := ctx.Value(accessLogPathKey{})
		if dest == nil {
			next.ServeHTTP(w, r)
			return
		}
		destPath := dest.(string)
		if destPath == "" || destPath == "/dev/null" {
			next.ServeHTTP(w, r)
			return
		}
		level := "info"
		if v := ctx.Value(accessLogLevelKey{}); v != nil {
			level = v.(string)
		}
		if v := ctx.Value(ngxLoggerKey{}); v != nil {
			ngx := v.(ngxLogger)
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start)
			m := ctx.Value(variables{}).(*sync.Map)
			m.Store(vRequestTime, duration.Milliseconds())
			format := defaultLogFormat
			if f := ctx.Value(accessLogFormat{}); f != nil {
				format = f.(string)
			}
			ngx.Println(destPath, level, resolveVariables(m, []byte(format)))
		}
	})
}
