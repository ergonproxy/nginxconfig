package core

import (
	"net"
	"time"

	"github.com/ergongate/nginxconfig/config"
	"github.com/ergongate/nginxconfig/internal/helpers"
)

// Core iterates on directives for core configurations.
func Core(directive *config.Directive) (config.Core, error) {
	e := config.NewError("core", directive.Name)
	if directive.Name != "main" {
		// we only parse core functionality from the main directive. This is
		// equivalent to nginx global context or main.
		e.Add(directive.Error("expect main directive"))
		return config.Core{}, e
	}
	var c config.Core
	if e.HasErrors() {
		return c, e
	}
	return c, nil
}

// AcceptMutex parses nginx accept_mutex directive.
//
// If accept_mutex is enabled, worker processes will accept new connections by
// turn. Otherwise, all worker processes will be notified about new connections,
// and if volume of new connections is low, some of the worker processes may
// just waste system resources
func AcceptMutex(d *config.Directive) (bool, error) {
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

// AcceptMutexDelay parses accept_mutex_delay directive to time.Duration
func AcceptMutexDelay(d *config.Directive) (time.Duration, error) {
	err := d.BasicCheck("accept_mutex_delay", 1, "events")
	if err != nil {
		return 0, err
	}
	v, err := helpers.ParseDuration(d.Params[0].Text)
	if err != nil {
		return 0, d.Params[0].Error(err.Error())
	}
	net.Dial()
	return v, nil
}
