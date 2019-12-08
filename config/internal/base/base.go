package base

import (
	"github.com/ergongate/vince/config/internal/modules/core"
	"github.com/ergongate/vince/config/nginx"
)

// Load traverse directive d and constucts a base configuration.
func Load(d *nginx.Directive) (*nginx.Base, error) {
	c := &nginx.Base{}
	var err error
	c.Core, err = core.Core(d)
	if err != nil {
		return nil, err
	}
	return c, nil
}
