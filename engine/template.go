package engine

import (
	"context"
	"regexp"
	"sync"
)

// we use this to hold a map of variable names in context this ia guaranteed to
// be present on request contextx.
type variables struct{}

// sets key/value into the context variables map. This is similar to set inside
// the nginx configuration.
func set(ctx context.Context, key, value interface{}) {
	if v := ctx.Value(variables{}); v != nil {
		m := v.(*sync.Map)
		m.Store(key, value)
	}
}

var variableRegexp = regexp.MustCompile(`\$([a-z_]\w*)`)

// VariablesToTemplates return src with all matching vnginx variable names
// repaced with go's template syntax
func VariablesToTemplates(src []byte) []byte {
	return variableRegexp.ReplaceAllFunc(src, func(name []byte) []byte {
		n := string(name)
		return []byte("{{." + n + "}}")
	})
}
