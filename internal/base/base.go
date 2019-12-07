package base

import (
	"github.com/ergongate/nginxconfig/config"
	"github.com/ergongate/nginxconfig/internal/modules/core"
)

// Load traverse directive d and constucts a base configuration.
func Load(d *config.Directive) (*config.Base, error) {
	c := &config.Base{}
	var err error
	c.Core, err = core.Core(d)
	if err != nil {
		return nil, err
	}
	return c, nil
}
