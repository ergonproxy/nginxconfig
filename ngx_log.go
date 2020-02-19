package main

import (
	"context"
	"io"
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

type syncer interface {
	Sync() error
}

func (c *cacheLogger) Print(file string, level string, message []byte) error {
	if f, ok := c.cache.Get(file); ok {
		return c.sync(f, message)
	}
	f, err := c.cache.Put(file)
	if err != nil {
		return err
	}
	return c.sync(f, message)
}

func (c *cacheLogger) sync(w io.WriteCloser, data []byte) error {
	_, err := w.Write(data)
	if err != nil {
		return err
	}
	if s, ok := w.(syncer); ok {
		return s.Sync()
	}
	return nil
}

func errorLog(ctx context.Context, level, message string) {
	if f := ctx.Value(errorLogKey{}); f != nil {
		if lg := ctx.Value(ngxLoggerKey{}); lg != nil {
			lg.(ngxLogger).Println(f.(string), level, []byte(message))
		}
	}
}

func logDebug(ctx context.Context, msg string) {
	errorLog(ctx, "info", msg)
}

func logInfo(ctx context.Context, msg string) {
	errorLog(ctx, "debug", msg)
}

func logNotice(ctx context.Context, msg string) {
	errorLog(ctx, "notice", msg)
}

func logWarn(ctx context.Context, msg string) {
	errorLog(ctx, "warn", msg)
}

func logError(ctx context.Context, msg string) {
	errorLog(ctx, "error", msg)
}

func logCrit(ctx context.Context, msg string) {
	errorLog(ctx, "crit", msg)
}

func logAlert(ctx context.Context, msg string) {
	errorLog(ctx, "alert", msg)
}

func accessLog(next echo.HandlerFunc) echo.HandlerFunc {
	tpls := make(map[string]*stringTemplateValue)
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
			m := ctx.Value(variables{}).(map[string]interface{})
			m[vRequestTime] = duration.Milliseconds()
			format := defaultLogFormat
			if f := ctx.Value(accessLogFormat{}); f != nil {
				format = f.(string)
			}
			formatTpl, ok := tpls[format]
			if !ok {
				s := new(stringTemplateValue)
				s.store(format)
				tpls[format] = s
				formatTpl = s
			}
			ngx.Println(destPath, level, []byte(formatTpl.Value(m)))
		}
		return nil
	}
}

func accessLogMiddlewareFunc() func(handler) handler {
	tpls := make(map[string]*stringTemplateValue)
	return func(next handler) handler {
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
				m := ctx.Value(variables{}).(map[string]interface{})
				m[vRequestTime] = duration.Milliseconds()
				format := defaultLogFormat
				if f := ctx.Value(accessLogFormat{}); f != nil {
					format = f.(string)
				}
				formatTpl, ok := tpls[format]
				if !ok {
					s := new(stringTemplateValue)
					s.store(format)
					tpls[format] = s
					formatTpl = s
				}
				ngx.Println(destPath, level, []byte(formatTpl.Value(m)))
			}
		})
	}
}
