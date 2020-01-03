package engine

import (
	"errors"

	"github.com/ergongate/vince/config/nginx"
)

type errLogInfo struct{}

type configContext struct {
	ErrorLog ErrorLog
	PID      string
}

func (m *configContext) check() error {
	if m.PID == "" {
		return errors.New("missing pid directive")
	}
	return nil
}

func (ctx *configContext) set(d *nginx.Directive) bool {
	switch d.Name {
	case "pid":
		ctx.PID = d.Params[0].Text
	case "error_log":
		ctx.ErrorLog.Name = d.Params[0].Text
	}
	return true
}

type ErrorLog struct {
	Name  string
	Level string
}

func loadContext(ctx *configContext, d *nginx.Directive) {
	if d.Name == "main" && d.Body != nil {
		d.Body.Blocks.Iter(ctx.set)
	}
}
