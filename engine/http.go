package engine

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/ergongate/vince/config/nginx"
	"github.com/ergongate/vince/version"
	"go.uber.org/zap"
)

type Nginx struct {
	Handler http.Handler
	config  *nginx.Base
	server  *http.Server
}

type Listeners struct {
	port    map[string]nginx.DirectiveList
	unix    map[string]nginx.DirectiveList
	address map[string]nginx.DirectiveList
}

type ListenOptions struct {
	Protocol        string
	Address         string
	SSL             bool
	HTTP2           bool
	SPDY            bool
	ProxyProtocol   bool
	RCVBuf          int
	SendBuf         int
	AcceptFilter    string
	IPv6Only        bool
	ReusePort       bool
	Default         bool
	ServerDirective *nginx.Directive
}

type ListenersConfig struct {
	Opts map[string][]*ListenOptions
}

func SetupListeners(root nginx.DirectiveList, defaultPort string) (ls *ListenersConfig) {
	root.Iter(func(d *nginx.Directive) bool {
		if d.Name == "http" {
			if d.Body != nil {
				d.Body.Blocks.Iter(func(s *nginx.Directive) bool {
					if s.Name == "server" {
						if s.Body != nil {
							// we are only interested on the listen directive of the server.
							s.Body.Blocks.Iter(func(l *nginx.Directive) bool {
								if l.Name == "listen" {
									if ls == nil {
										ls = &ListenersConfig{
											Opts: make(map[string][]*ListenOptions),
										}
									}
									o := &ListenOptions{
										Protocol:        "tcp",
										ServerDirective: s,
									}

									if len(l.Params) > 0 {
										a := l.Params[0].Text
										if strings.HasPrefix(a, "unix://") {
											o.Protocol = "unix"
											o.Address = strings.TrimPrefix(a, "unix://")
										} else if ip := net.ParseIP(a); ip != nil {
											o.Address = a + ":" + defaultPort
										} else if _, err := strconv.Atoi(a); err == nil {
											o.Address = ":" + a
										} else {
											_, _, err := net.SplitHostPort(a)
											if err != nil {
												if strings.Contains(err.Error(), "missing port in address") {
													o.Address = a + ":" + defaultPort
												} else {
													//TODO log error
												}
											} else {
												o.Address = a
											}
										}
										l.Params.Iter(func(idx int, p *nginx.Token) bool {
											switch p.Text {
											case "default_server":
												o.Default = true
											case "udp":
												o.Protocol = "udp"
											case "SSL":
												o.SSL = true
											case "http2":
												o.HTTP2 = true
											case "spdy":
												o.SPDY = true
											case "proxy_protocol":
												o.ProxyProtocol = true
											}
											return true
										})
										if v, ok := ls.Opts[a]; ok {
											ls.Opts[a] = append(v, o)
										} else {
											ls.Opts[a] = []*ListenOptions{o}
										}
									}
									// a server directive can have multiple liste, this make it possible to
									// configure a server to handle both http and https traffick
								}
								return true
							})
						}
					}
					return true
				})
			}
			return false
		}
		return true
	})
	return
}

func (h *Nginx) Serve(ls net.Listener) error {
	return h.server.Serve(ls)
}

func (h *Nginx) ShutDown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *Nginx) Close() error {
	return h.server.Close()
}

func serverName() string {
	return version.Text + version.Version
}

func serveError(code int, w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(code)
	err := httpErrorTemplate.Execute(w, map[string]interface{}{
		"code":        code,
		"text":        http.StatusText(code),
		"server_name": serverName(),
	})
	if err != nil {
		log(r.Context()).Error("Rendering error template", zap.Error(err))
	}
}
