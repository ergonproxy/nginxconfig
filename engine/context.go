package engine

import "github.com/ergongate/vince/config/nginx"

import "errors"

type errLogInfo struct{}

type mainContext struct {
	errorLog ErrorLog
	pid      string
}

func (m *mainContext) check() error {
	if m.pid == "" {
		return errors.New("missing pid directive")
	}
	return nil
}

func (ctx *mainContext) set(d *nginx.Directive) bool {
	switch d.Name {
	case "pid":
		ctx.pid = d.Params[0].Text
	case "error_log":
		ctx.errorLog.Name = d.Params[0].Text
	}
	return true
}

type ErrorLog struct {
	Name  string
	Level string
}

func loadContext(ctx *mainContext, d *nginx.Directive) {
	if d.Name == "main" && d.Body != nil {
		d.Body.Blocks.Iter(ctx.set)
	}
}
