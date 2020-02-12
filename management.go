package main

import (
	"net/http"

	"github.com/ergongate/vince/buffers"
	"github.com/ergongate/vince/templates"
	"github.com/labstack/echo/v4"
)

type management struct {
	ctx *serverCtx
	h   http.Handler
}

func (m *management) init(ctx *serverCtx) {
	m.ctx = ctx
	h := echo.New()
	h.GET("/", m.index)
	h.GET("/assets/*", m.static())
	m.h = h
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
	err := m.ctx.http.tpl.ExecuteTemplate(buf, "management/index.html", with)
	if err != nil {
		return err
	}
	return ctx.HTML(http.StatusOK, buf.String())
}
