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
	connections() (*connections, error)
	deleteConnections() error
	http() ([]string, error)
	httpRequests() (*httpRequests, error)
	deleteHttpRequests() error
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

type connections struct {
	Accepted int64 `json:"accepted"`
	Dropped  int64 `json:"dropped"`
	Active   int64 `json:"active"`
	Idle     int64 `json:"idle"`
}

func (c connections) filter(keys ...string) interface{} {
	if len(keys) > 0 {
		m := map[string]int64{
			"accepted": c.Accepted,
			"dropped":  c.Dropped,
			"active":   c.Active,
			"idle":     c.Idle,
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
	return c
}

type httpRequests struct {
	Total   int64 `json:"total"`
	Current int64 `json:"current"`
}

func (h httpRequests) filter(keys ...string) interface{} {
	if len(keys) > 0 {
		m := map[string]int64{
			"total":   h.Total,
			"current": h.Current,
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
	return h
}

type httpServerZones struct {
	Processing int64 `json:"processing"`
	Requests   int64 `json:"requests"`
	Responses  struct {
		R1xx  int64 `json:"1xx"`
		R2xx  int64 `json:"2xx"`
		R3xx  int64 `json:"3xx"`
		R4xx  int64 `json:"4xx"`
		R5xx  int64 `json:"5xx"`
		Total int64 `json:"total"`
	} `json:"responses"`
	Discarded int64 `json:"discarded"`
	Received  int64 `json:"received"`
	Sent      int64 `json:"sent"`
}

func formatISO8601TimeStamp(t time.Time) string {
	return t.Format(iso8601Milli)
}

func newHTTPAPI(base string, httpCtx httpContext) http.Handler {
	e := echo.New()
	e.Use(accessLog)
	api := e.Group(base)
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
			return api500Error(ctx, err)
		}
		return ctx.JSON(http.StatusOK, apiObject(ngx, ctx.QueryParam("fields")))
	})
	api.GET("/processes", func(ctx echo.Context) error {
		p, err := httpCtx.processes()
		if err != nil {
			return api500Error(ctx, err)
		}
		return ctx.JSON(http.StatusOK, p)
	})
	api.DELETE("/processes", func(ctx echo.Context) error {
		err := httpCtx.deleteProcesses()
		if err != nil {
			return api500Error(ctx, err)
		}
		return ctx.JSON(http.StatusNoContent, nil)
	})
	api.GET("/connections", func(ctx echo.Context) error {
		c, err := httpCtx.connections()
		if err != nil {
			return api500Error(ctx, err)
		}
		return ctx.JSON(http.StatusOK, apiObject(c, ctx.QueryParam("fields")))
	})
	api.DELETE("/connections", func(ctx echo.Context) error {
		err := httpCtx.deleteConnections()
		if err != nil {
			return api500Error(ctx, err)
		}
		return ctx.JSON(http.StatusNoContent, nil)
	})
	api.GET("/slabs", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusNotImplemented, nil)
	})
	api.GET("/slabs/:slabs_name", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusNotImplemented, nil)
	})
	api.DELETE("/slabs/:slabs_name", func(ctx echo.Context) error {
		return ctx.JSON(http.StatusNotImplemented, nil)
	})
	api.GET("/http/", func(ctx echo.Context) error {
		p, err := httpCtx.http()
		if err != nil {
			return api500Error(ctx, err)
		}
		return ctx.JSON(http.StatusOK, p)
	})
	api.GET("/http/requests", func(ctx echo.Context) error {
		c, err := httpCtx.connections()
		if err != nil {
			return api500Error(ctx, err)
		}
		return ctx.JSON(http.StatusOK, apiObject(c, ctx.QueryParam("fields")))
	})
	api.DELETE("/http/requests", func(ctx echo.Context) error {
		err := httpCtx.deleteHttpRequests()
		if err != nil {
			return api500Error(ctx, err)
		}
		return ctx.JSON(http.StatusNoContent, nil)
	})
	return e
}

func api500Error(echoCtx echo.Context, err error) error {
	ctx := echoCtx.Request().Context()
	errorLog(ctx, err)
	return echoCtx.JSON(
		http.StatusInternalServerError,
		newHTTPErrorResponse(http.StatusInternalServerError,
			ctx.Value(requestID{}).(string)),
	)
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
