package engine

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"
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

// resolveVariables replaces any variables with their values.
func resolveVariables(src []byte, ctx *sync.Map) []byte {
	return variableRegexp.ReplaceAllFunc(src, func(name []byte) []byte {
		n := string(name)
		if v, ok := ctx.Load(n); ok {
			return toByte(v)
		}
		return []byte{}
	})
}

func toByte(v interface{}) []byte {
	switch e := v.(type) {
	case []byte:
		return e
	case string:
		return []byte(e)
	default:
		return []byte(fmt.Sprint(v))
	}
}

const specialError = `
<html>
	<head>{{.code}} {{.text}}</head>
	<body>
		<center><h1>{{.code}} {{.text}}</h1></center>
		<hr><center>{{.server_version}}</center>
	</body>
</html`

var httpErrorTemplate = template.Must(template.New("error").Parse(specialError))

// sets variables $arg and $arg_name
func setArgs(r *http.Request, m *sync.Map) {
	m.Store("$arg", r.URL.RawQuery)
	m.Store("$query_string", r.URL.RawQuery)
	q := r.URL.Query()
	for k := range q {
		m.Store("$arg_"+k, q.Get(k))
	}
}

func setHeaders(r *http.Request, m *sync.Map) {
	src := []struct{ variable, header string }{
		{"$content_length", "Content-Length"},
		{"$content_type", "Content-Type"},
	}
	for _, v := range src {
		m.Store(v.variable, r.Header.Get(v.header))
	}
	for v := range r.Header {
		m.Store("$http_"+toname(v), r.Header.Get(v))
	}
}

func setCookies(r *http.Request, m *sync.Map) {
	for _, v := range r.Cookies() {
		m.Store("$cookie_"+toname(v.Name), v.Value)
	}
}

func toname(s string) string {
	return strings.Replace(strings.ToLower(s), "-", "_", -1)
}
