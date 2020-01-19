package main

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type (
	defaultPortKey struct{}
)

type handlers []http.Handler

func httpCore(ctx context.Context, id func() int64, stmt *Stmt) *http.Server {
	return &http.Server{
		BaseContext: httpBaseCtx(ctx, id, stmt),
	}
}

func httpBaseCtx(ctx context.Context, id func() int64, stmt *Stmt) func(net.Listener) context.Context {
	servers := findServers(ctx, stmt)
	return func(ls net.Listener) context.Context {
		r := id() // assign a new id on any new connection.
		baseCtx := context.WithValue(ctx, requestID{}, r)
		setVariable(baseCtx, vRequestID, r)
		baseCtx = context.WithValue(baseCtx, serverKey{}, servers(ls))
		return baseCtx
	}
}

func findServers(ctx context.Context, stmt *Stmt) func(net.Listener) []*Stmt {
	var servers []*Stmt
	var ok bool
	return func(ls net.Listener) []*Stmt {
		if ok {
			return servers
		}
		for _, ch := range stmt.Blocks {
			if ch.Directive == "server" {
				if useServer(ctx, ch, ls) {
					servers = append(servers, ch)
				}
			}
		}
		ok = true
		return servers
	}
}

func useServer(ctx context.Context, stmt *Stmt, ls net.Listener) bool {
	for _, ch := range stmt.Blocks {
		if ch.Directive == "listen" {
			if matchListener(ch, ls.Addr()) {
				return true
			}
		}
	}
	return defaultAddress(ctx, ls.Addr())
}

func matchListener(stmt *Stmt, addr net.Addr) bool {
	return false
}

// returns true if add is bound  on the global default port. The
// default port acts as global virtual server, any server that doe not have a
// listen directive is listening on the default address.
//
// by default port 80 is used when running vince as root and 8000 is used when
// running as non root user.
func defaultAddress(ctx context.Context, addr net.Addr) bool {
	if p := ctx.Value(defaultPortKey{}); p != nil {
		return strings.HasSuffix(addr.String(), ":"+p.(string))
	}
	return false
}

type listenOpts struct {
	net           string
	addrPort      string
	defaultServer bool
	ssl           bool
	http2         bool
	spdy          bool
	proxyProtocol bool
}

func parseListen(stmt *Stmt, defaultPort string) listenOpts {
	var ls listenOpts
	if len(stmt.Args) > 0 {
		a := stmt.Args[0]
		if _, err := strconv.Atoi(a); err == nil {
			ls.net = "tcp"
			ls.addrPort = ":" + a
		} else if h, p, err := net.SplitHostPort(a); err == nil {
			if h == "unix" {
				ls.net = h
				ls.addrPort = p
			} else {
				ls.net = "tcp"
				ls.addrPort = a
			}
		} else {
			switch a {
			case "localhost", "127.0.0.1", "[::]", "[::1]":
				ls.net = "tcp"
				ls.addrPort = a + ":" + defaultPort
			default:
				u, err := url.Parse(a)
				if err == nil {
					ls.net = u.Scheme
					ls.addrPort = u.Host
					//TODO: ensure there is port set
				}
			}
		}
		if len(stmt.Args) > 1 {
			for _, a := range stmt.Args[1:] {
				switch a {
				case "default_server":
					ls.defaultServer = true
				case "ssl":
					ls.ssl = true
				case "http2":
					ls.http2 = true
				case "spdy":
					ls.spdy = true
				case "proxy_protocol":
					ls.proxyProtocol = true
				}
			}
		}
	}
	return ls
}
