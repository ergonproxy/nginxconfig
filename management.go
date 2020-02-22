package main

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/ergongate/vince/buffers"
	"github.com/ergongate/vince/templates"
	"github.com/labstack/echo/v4"
)

type management struct {
	ctx *serverCtx
	git gitOps
	h   http.Handler
}

func init() {
	echo.NotFoundHandler = func(c echo.Context) error {
		return e404(c.Response())
	}
}

func (m *management) init(ctx *serverCtx) {
	m.ctx = ctx
	h := echo.New()
	h.Use(instrumentEcho)
	h.Use(accessLog)
	h.GET("/", m.index)
	h.GET("/assets/*", m.static())
	h.GET("/metrics", echo.WrapHandler(http.HandlerFunc(metricsHandler)))
	var ops gitOpsOptions
	ops.dir = filepath.Join(ctx.config.dir, "configs")
	m.git.init(ops)
	m.git.handler(h)
	h.HTTPErrorHandler = echoErrorHandler
	m.h = h
}

func echoErrorHandler(err error, ctx echo.Context) {
	fmt.Println("not found  ", ctx.Path())
	if e, ok := err.(*echo.HTTPError); ok {
		eRender(ctx.Response(), e.Code)
		return
	}
	e500(ctx.Response())
}

func (m management) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.h.ServeHTTP(w, r)
}

func (m management) static() echo.HandlerFunc {
	static := http.FileServer(templates.WebFS)
	sh := http.StripPrefix("/assets", static)
	return func(ctx echo.Context) error {
		sh.ServeHTTP(ctx.Response(), ctx.Request())
		return nil
	}
}

func (m *management) index(ctx echo.Context) error {
	with := new(templates.Context)
	buf := buffers.GetBytes()
	defer buffers.PutBytes(buf)
	err := templates.ExecHTML(buf, "management/index.html", with)
	if err != nil {
		return err
	}
	return ctx.HTML(http.StatusOK, buf.String())
}
