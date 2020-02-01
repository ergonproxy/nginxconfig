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
	m.h = h
}

func (m management) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.h.ServeHTTP(w, r)
}

func (m *management) index(ctx echo.Context) error {
	with := new(templates.Context)
	buf := buffers.GetBytes()
	defer buffers.PutBytes(buf)
	err := m.ctx.tpl.ExecuteTemplate(buf, "management/index.html", with)
	if err != nil {
		return err
	}
	return ctx.HTML(http.StatusOK, buf.String())
}
