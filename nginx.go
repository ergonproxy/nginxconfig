package nginx

import (
	"io"

	"github.com/ergongate/nginxconfig/config"
	"github.com/ergongate/nginxconfig/internal/base"
	"github.com/ergongate/nginxconfig/lex"
)

// Load reads from src, then parses and returns a valid base nginx
// configuration. This is the main nginx configuration, linting and verification
// is done, meaning error will include all errors that wene encountered during
// linting and parsing of src.
//
// It should be known, the goal is correctness, if there is no error returned it
// means src is a correct nginx configuration.
func Load(filename string, src io.Reader) (*config.Base, error) {
	lx, err := lex.File(filename, src)
	if err != nil {
		return nil, err
	}
	return base.Load(lx)
}
