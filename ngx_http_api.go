package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

const supportedAPIVersion = "6"

type httpContext interface {
	nginx() (*nginx, error)
	processes() (*processes, error)
	deleteProcesses() error
}

type filter interface {
	filter(...string) interface{}
}

type nginx struct {
	Version       string `json:"version"`
	Build         string `json:"build"`
	Address       string `json:"address"`
	Generation    int64  `json:"generation"`
	LoadTimeStamp string `json:"load_timestamp"`
	TimeStamp     string `json:"timestamp"`
	Pid           string `json:"pid"`
	PPid          string `json:"ppid"`
}

type processes struct {
	Respawned int `json:"respawned"`
}

func (n nginx) filter(keys ...string) interface{} {
	if len(keys) > 0 {
		m := map[string]interface{}{
			"version":        n.Version,
			"build":          n.Build,
			"address":        n.Address,
			"generation":     n.Generation,
			"load_timestamp": n.LoadTimeStamp,
			"timestamp":      n.TimeStamp,
			"pid":            n.Pid,
			"ppid":           n.PPid,
		}
		x := make(map[string]bool)
		for _, v := range keys {
			x[v] = true
		}
		for k := range m {
			if !x[k] {
				delete(m, k)
			}
		}
		return m
	}
	return n
}

func formatISO8601TimeStamp(t time.Time) string {
	return t.Format(iso8601Milli)
}

func newHTTPAPI(base string, httpCtx httpContext) http.Handler {
	e := echo.New()
	api := e.Group(withTrailSlash(base) + supportedAPIVersion)
	api.GET("/", func(ctx echo.Context) error {
		var routes []string
		for _, v := range ctx.Echo().Routes() {
			routes = append(routes, v.Path)
		}
		return ctx.JSON(http.StatusOK, routes)
	})
	api.GET("/nginx", func(ctx echo.Context) error {
		ngx, err := httpCtx.nginx()
		if err != nil {
			//TODO serve nginx api error
			return err
		}
		return ctx.JSON(http.StatusOK, apiObject(ngx, ctx.QueryParam("fields")))
	})
	api.GET("/processes", func(ctx echo.Context) error {
		p, err := httpCtx.processes()
		if err != nil {
			//TODO serve nginx api error
			return err
		}
		return ctx.JSON(http.StatusOK, p)
	})
	api.DELETE("/processes", func(ctx echo.Context) error {
		err := httpCtx.deleteProcesses()
		if err != nil {
			//TODO serve nginx api error
			return err
		}
		return ctx.JSON(http.StatusNoContent, nil)
	})
	return e
}

func withTrailSlash(s string) string {
	if s == "" {
		return "/"
	}
	if s[len(s)-1] == '/' {
		return s
	}
	return s + "/"
}

func apiObject(v interface{}, query string) interface{} {
	if query == "" {
		return v
	}
	if f, ok := v.(filter); ok {
		return f.filter(strings.Split(query, ",")...)
	}
	return v
}
