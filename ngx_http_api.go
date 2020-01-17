package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type httpContext interface {
	nginx() (*nginx, error)
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

func formatISO8601TimeStamp(t time.Time) string {
	return t.Format(iso8601Milli)
}

func newHttpAPI(base string, httpCtx httpContext) http.Handler {
	e := echo.New()
	api := e.Group(base)
	api.GET("/", func(ctx echo.Context) error {
		var routes []string
		for _, v := range ctx.Echo().Routes() {
			routes = append(routes, v.Path)
		}
		return ctx.JSON(http.StatusOK, routes)
	})
	api.GET("/nginx", func(ctx echo.Context) error {
		return nil
	})
	return e
}
