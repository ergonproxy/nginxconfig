package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type serverCtxKey struct{}

var serial int64

func nextID() int64 {
	return atomic.AddInt64(&serial, 1)
}

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

func overide(rules []*rule) []*rule {
	m := make(map[string]int)
	var remove []int
	for k, v := range rules {
		if i, ok := m[v.name]; ok {
			remove = append(remove, i)
		}
		m[v.name] = k
	}
	for _, i := range remove {
		rules = append(rules[:i], rules[i+1:]...)
	}
	return rules
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
		sctx.address[ls.addrPort] = ls
		if a, ok := sctx.ls1[ls.addrPort]; ok {
			sctx.ls1[ls.addrPort] = append(a, v)
		} else {
			sctx.ls1[ls.addrPort] = []*rule{v}
		}
	}
	// start lisenters
	defer func() {
		// make sure all listeners are closed before exiting
		for _, l := range sctx.ls2 {
			l.Close() // TODO:(gernest) handle error
		}
	}()
	for k := range sctx.ls1 {
		opts := sctx.address[k]
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
		sctx.ls2[opts.addrPort] = l
	}
	for k, rules := range sctx.ls1 {
		opts := sctx.address[k]
		srv, err := createHTTPServer(context.WithValue(ctx, serverCtxKey{}, sctx), rules, opts)
		if err != nil {
			return err
		}
		sctx.ls3[opts.addrPort] = srv
	}

	// we can start servers now
	for opts, srv := range sctx.ls3 {
		go srv.Serve(sctx.ls2[opts])
	}
	return nil
}

type serverCtx struct {
	core          *rule
	defaultServer map[string]*rule
	address       map[string]*listenOpts
	ls1           map[string][]*rule
	ls2           map[string]net.Listener
	ls3           map[string]*http.Server
	active        *listenOpts
}

func (s *serverCtx) with(active *listenOpts) *serverCtx {
	return &serverCtx{core: s.core, ls1: s.ls1, ls2: s.ls2, ls3: s.ls3, active: active}
}

func newSrvCtx() *serverCtx {
	return &serverCtx{
		address: make(map[string]*listenOpts),
		ls1:     make(map[string][]*rule),
		ls2:     make(map[string]net.Listener),
		ls3:     make(map[string]*http.Server),
	}
}

func createHTTPServer(ctx context.Context, servers []*rule, opts *listenOpts) (*http.Server, error) {
	s := &http.Server{}
	opts.manager = newHTTPConnManager(nextID)
	s.BaseContext = func(ls net.Listener) context.Context {
		return opts.manager.baseCtx(ctx, ls)
	}
	s.ConnState = opts.manager.manageConnState
	s.ConnContext = opts.manager.connContext
	s.Handler = ngnxHandler(ctx, servers)
	return s, nil
}

func matchWildCard(s string, wild string) bool {
	wp := strings.Split(wild, ".")
	sp := strings.Split(s, ".")
	if len(sp) < len(wp) {
		return false
	}
	sidx := 0
	widx := 0
	for ; sidx < len(sp); sidx++ {
		if wp[widx] == "*" {
			continue
		}
		if !(widx < len(wp) && sp[sidx] == wp[widx]) {
			return false
		}
		widx++
	}
	return true
}

func isWildCard(w string) bool {
	if strings.IndexByte(w, '*') == -1 {
		return false
	}
	start := strings.HasPrefix(w, "*.")
	end := strings.HasSuffix(w, ".*")
	return start || end
}

type locationMatch struct {
	rules []*match
}

type matchKind uint

const (
	matchExact matchKind = iota
	matchCaret
	matchPrefix
	matchRegexp
)

type match struct {
	kind matchKind
	rule *rule
	re   *regexp.Regexp
}

func (ls *locationMatch) match(path string) *rule {
	for i := 0; i < len(ls.rules); i++ {
		if ls.rules[i].kind == matchExact && ls.rules[i].rule.args[1] == path {
			return ls.rules[i].rule
		}
	}
	var m []*match
	for i := 0; i < len(ls.rules); i++ {
		switch ls.rules[i].kind {
		case matchPrefix, matchCaret:
			if strings.HasPrefix(path, ls.rules[i].rule.args[1]) {
				m = append(m, ls.rules[i])
			}
		}
	}
	var selected *match
	if m != nil {
		sort.Slice(m, func(i, j int) bool {
			return m[i].rule.args[i] > m[i].rule.args[j]
		})
		selected = m[0]
		if selected.kind == matchCaret {
			return selected.rule
		}
	}
	var up *locationMatch
	if selected != nil {
		up = new(locationMatch)
		up.load(selected.rule)
	}
	if up != nil {
		for i := 0; i < len(up.rules); i++ {
			if up.rules[i].kind == matchRegexp {
				if up.rules[i].re.MatchString(path) {
					return up.rules[i].rule
				}
			}
		}
	}
	for i := 0; i < len(up.rules); i++ {
		if ls.rules[i].kind == matchRegexp {
			if ls.rules[i].re.MatchString(path) {
				return ls.rules[i].rule
			}
		}
	}
	if selected != nil {
		return selected.rule
	}
	return nil
}

func (ls *locationMatch) load(srv *rule) {
	for _, ch := range srv.children {
		if ch.name == "location" {
			switch len(ch.args) {
			case 1:
				ls.rules = append(ls.rules, &match{
					kind: matchPrefix,
					rule: ch,
				})
			case 2:
				// with modifiers
				switch ch.args[0] {
				case "=":
					ls.rules = append(ls.rules, &match{
						kind: matchExact,
						rule: ch,
					})
				case "~":
					r := &match{rule: ch, kind: matchRegexp}
					r.re = regexp.MustCompile(ch.args[1])
					ls.rules = append(ls.rules, r)
				case "~*":
					r := &match{rule: ch, kind: matchRegexp}
					r.re = regexp.MustCompile("(?i)" + ch.args[1])
					ls.rules = append(ls.rules, r)
				case "^~":
					ls.rules = append(ls.rules, &match{
						kind: matchCaret,
						rule: ch,
					})
				}
			}
		}
	}
}

func ngnxHandler(ctx context.Context, servers []*rule) http.Handler {
	sctx := ctx.Value(serverCtxKey{}).(*serverCtx)
	reg := make(map[*regexp.Regexp]*rule)
	exact := make(map[string][]*rule)
	wild := make(map[string]*rule)
	for _, srv := range servers {
		for _, ch := range srv.children {
			if ch.name == "server_name" {
				for _, a := range ch.args {
					if a[0] == '~' {
						re := regexp.MustCompile(a[1:])
						reg[re] = srv
						continue
					}
					if isWildCard(a) {
						wild[a] = srv
						continue
					}
					if v, ok := exact[a]; ok {
						exact[a] = append(v, srv)
					} else {
						exact[a] = []*rule{srv}
					}
				}
				break
			}
		}
	}
	find := func(name string) *rule {
		if r, ok := exact[name]; ok {
			return r[0]
		}
		if len(wild) > 0 {
			var match string
			for w := range wild {
				if matchWildCard(name, w) {
					if w > match {
						match = w
					}
				}
			}
			if match != "" {
				return wild[match]
			}
		}
		for re, r := range reg {
			if re.MatchString(name) {
				return r
			}
		}
		return sctx.defaultServer[sctx.active.addrPort]
	}
	location := new(sync.Map)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var srv *rule
		if len(servers) == 1 {
			srv = servers[0]
		} else {
			srv = find(r.Host)
		}
		if srv == nil {
			srv = servers[0]
		}
		var loc *locationMatch
		if v, ok := location.Load(srv); ok {
			loc = v.(*locationMatch)
		} else {
			loc = new(locationMatch)
			loc.load(srv)
			location.Store(srv, loc)
		}
		if l := loc.match(r.URL.Path); l != nil {
			// TODO serve the returned location block
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})
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
