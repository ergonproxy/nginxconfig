package config

import (
	"io"

	"github.com/ergongate/vince/config/internal/base"
	"github.com/ergongate/vince/config/internal/lex"
	"github.com/ergongate/vince/config/nginx"
)

// Config is the configuration object for vince. This is a valid nginx
// configuration with additional vince specific modules.
type Config nginx.Base

// Load reads from src, then parses and returns a valid base nginx
// configuration. This is the main nginx configuration, linting and verification
// is done, meaning error will include all errors that wene encountered during
// linting and parsing of src.
//
// It should be known, the goal is correctness, if there is no error returned it
// means src is a correct nginx configuration.
func Load(filename string, src io.Reader) (*nginx.Base, error) {
	lx, err := lex.File(filename, src)
	if err != nil {
		return nil, err
	}
	return base.Load(lx)
}

func LoadDirective(filname string, src io.Reader) (*nginx.Directive, error) {
	return lex.File(filname, src)
}
