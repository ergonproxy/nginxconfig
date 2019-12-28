package core

import (
	"time"

	"github.com/ergongate/vince/config/internal/helpers"
	"github.com/ergongate/vince/config/nginx"
)

// Core iterates on directives for core configurations.
func Core(d *nginx.Directive) (nginx.Core, error) {
	err := d.BasicCheck("main", 0)
	if err != nil {
		return nginx.Core{}, err
	}
	e := nginx.NewError("core", d.Name)
	var c nginx.Core
	for _, child := range d.Body.Blocks {
		switch child.Name {
		case "events":
			c.Events, err = events(child)
			if err != nil {
				e.Add(err)
			}
		}
	}
	if e.HasErrors() {
		return c, e
	}
	return c, nil
}

func acceptMutex(d *nginx.Directive) (bool, error) {
	err := d.BasicCheck("accept_mutex", 1, "events")
	if err != nil {
		return false, err
	}
	v, err := helpers.ParseFlag(d.Params[0].Text)
	if err != nil {
		return false, nginx.ErrorAt(err.Error(), &d.Params[0].Start)
	}
	return v, nil
}

func acceptMutexDelay(d *nginx.Directive) (time.Duration, error) {
	err := d.BasicCheck("accept_mutex_delay", 1, "events")
	if err != nil {
		return 0, err
	}
	v, err := helpers.ParseDuration(d.Params[0].Text)
	if err != nil {
		return 0, d.Params[0].Error(err.Error())
	}
	return v, nil
}

func events(d *nginx.Directive) (*nginx.Events, error) {
	err := d.BasicCheck("events", 0, "main")
	if err != nil {
		return nil, err
	}
	e := nginx.NewError("core", d.Name)
	var ev nginx.Events
	for _, child := range d.Body.Blocks {
		switch child.Name {
		case "accept_mutex":
			ev.AcceptMutex, err = acceptMutex(child)
			if err != nil {
				e.Add(err)
			}
		case "accept_mutex_delay":
			ev.AcceptMutextDelay, err = acceptMutexDelay(child)
			if err != nil {
				e.Add(err)
			}
		case "debug_connection":
		case "master_process":
		case "use":
		case "worker_aio_requests":
		case "worker_connections":
		default:
			e.Add(child.Error("Unknown directive: " + child.Name))
		}
	}
	if e.HasErrors() {
		return nil, e
	}
	return &ev, nil
}
