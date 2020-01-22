package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
)

type serverCtxKey struct{}

type rule struct {
	parent   *rule
	name     string
	args     []string
	children []*rule
}

func (r rule) key() string {
	if r.parent != nil {
		return r.parent.key() + "." + r.name
	}
	return r.name
}

func (r rule) String() string {
	return r.key() + " [" + strings.Join(r.args, ",") + "]"
}

func (r *rule) collect(n *rule) []*rule {
	var v []*rule
	if n == nil {
		v = r.children
	} else {
		for _, a := range r.children {
			if a.name == n.name {
				break
			}
			v = append(v, a)
		}
	}
	if r.parent != nil {
		return append(r.parent.collect(r), v...)
	}
	return v
}

func ruleFromStmt(stmt *Stmt, parent *rule) *rule {
	r := &rule{name: stmt.Directive, parent: parent, args: stmt.Args}
	for _, b := range stmt.Blocks {
		r.children = append(r.children, ruleFromStmt(b, r))
	}
	return r
}

type startupOptions struct {
	defaultPort int
}

func startEverything(ctx context.Context, config *Stmt) error {
	core := ruleFromStmt(config, nil)
	sctx := newSrvCtx()
	var servers []*rule
	// main block
	for _, base := range core.children {
		if base.name == "http" {
			// http block
			for _, child := range base.children {
				if child.name == "server" {
					// server block
					servers = append(servers, child)
				}
			}
		}
	}
	defaultListener := listenOpts{
		net:      "tcp",
		addrPort: ":8000",
	}
	for _, v := range servers {
		ls := findListener(v, &defaultListener, "8000")
		if a, ok := sctx.ls1[ls]; ok {
			sctx.ls1[ls] = append(a, v)
		} else {
			sctx.ls1[ls] = []*rule{v}
		}
	}
	// start lisenters
	defer func() {
		// make sure all listeners are closed before exiting
		for _, l := range sctx.ls2 {
			l.Close() // TODO:(gernest) handle error
		}
	}()
	for opts := range sctx.ls1 {
		var l net.Listener
		var err error
		if opts.ssl {
			c, err := opts.sslOpts.config()
			if err != nil {
				return err
			}
			l, err = tls.Listen(opts.net, opts.addrPort, c)
		} else {
			l, err = net.Listen(opts.net, opts.addrPort)
		}
		l, err = net.Listen(opts.net, opts.addrPort)
		if err != nil {
			return err
		}
		sctx.ls2[opts] = l
	}
	for opts, rules := range sctx.ls1 {
		srv, err := createHTTPServer(context.WithValue(ctx, serverCtxKey{}, sctx), rules)
		if err != nil {
			return err
		}
		sctx.ls3[opts] = srv
	}

	// we can start servers now
	for opts, srv := range sctx.ls3 {
		go srv.Serve(sctx.ls2[opts])
	}
	return nil
}

type serverCtx struct {
	core   *rule
	ls1    map[*listenOpts][]*rule
	ls2    map[*listenOpts]net.Listener
	ls3    map[*listenOpts]*http.Server
	active *listenOpts
}

func (s *serverCtx) with(active *listenOpts) *serverCtx {
	return &serverCtx{core: s.core, ls1: s.ls1, ls2: s.ls2, ls3: s.ls3, active: active}
}

func newSrvCtx() *serverCtx {
	return &serverCtx{
		ls1: make(map[*listenOpts][]*rule),
		ls2: make(map[*listenOpts]net.Listener),
		ls3: make(map[*listenOpts]*http.Server),
	}
}

func createHTTPServer(ctx context.Context, rules []*rule) (*http.Server, error) {
	return nil, nil
}

func findListener(r *rule, def *listenOpts, port string) *listenOpts {
	o := *def
	for _, v := range r.children {
		if v.name == "listen" {
			o = parseListen(v, port)
			continue
		}
	}
	return &o
}
