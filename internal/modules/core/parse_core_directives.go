package core

import (
	"time"

	"github.com/ergongate/nginxconfig/config"
	"github.com/ergongate/nginxconfig/internal/helpers"
)

// Core iterates on directives for core configurations.
func Core(d *config.Directive) (config.Core, error) {
	err := d.BasicCheck("main", 0)
	if err != nil {
		return config.Core{}, err
	}
	e := config.NewError("core", d.Name)
	var c config.Core
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

func acceptMutex(d *config.Directive) (bool, error) {
	err := d.BasicCheck("accept_mutex", 1, "events")
	if err != nil {
		return false, err
	}
	v, err := helpers.ParseFlag(d.Params[0].Text)
	if err != nil {
		return false, config.ErrorAt(err.Error(), &d.Params[0].Start)
	}
	return v, nil
}

func acceptMutexDelay(d *config.Directive) (time.Duration, error) {
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

func events(d *config.Directive) (*config.Events, error) {
	err := d.BasicCheck("events", 0, "main")
	if err != nil {
		return nil, err
	}
	e := config.NewError("core", d.Name)
	var ev config.Events
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
			e.Add(child.Error("Unknown directive"))
		}
	}
	if e.HasErrors() {
		return nil, e
	}
	return &ev, nil
}
