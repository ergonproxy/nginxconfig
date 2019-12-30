package engine

import (
	"encoding/json"
	"net/http"
	"sync"
)

const welcome = `<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>`

func defaultHand(w http.ResponseWriter, r *http.Request) {
	m := make(map[string]interface{})
	r.Context().Value(variables{}).(*sync.Map).Range(func(key, value interface{}) bool {
		m[key.(string)] = value
		return true
	})
	err := json.NewEncoder(w).Encode(m)
	if err != nil {
	}
}

func httpServer(h http.Handler) *http.Server {
	return &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			v := r.Context().Value(variables{}).(*sync.Map)
			setArgs(r, v)
			setHeaders(r, v)
			setCookies(r, v)
			h.ServeHTTP(w, r)
		}),
	}
}
