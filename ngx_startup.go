package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/ergongate/vince/templates"
	"github.com/urfave/cli/v2"
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
			if a == n {
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
	return rules
}

func ruleFromStmt(stmt *Stmt, parent *rule) *rule {
	r := &rule{name: stmt.Directive, parent: parent, args: stmt.Args}
	for _, b := range stmt.Blocks {
		r.children = append(r.children, ruleFromStmt(b, r))
	}
	return r
}

func process(ctx context.Context, srvCtx *serverCtx, config *vinceConfiguration) error {
	var servers []*rule
	// main block
	for _, base := range srvCtx.core.children {
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

	for _, v := range servers {
		for _, ls := range findListener(v, config.defaultPort) {
			if _, ok := srvCtx.http.address[ls.addrPort]; !ok {
				srvCtx.http.address[ls.addrPort] = ls
			}
			if a, ok := srvCtx.http.serverRules[ls.addrPort]; ok {
				srvCtx.http.serverRules[ls.addrPort] = append(a, v)
			} else {
				srvCtx.http.serverRules[ls.addrPort] = []*rule{v}
			}
		}
	}
	for k := range srvCtx.http.serverRules {
		opts := srvCtx.http.address[k]
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
		if err != nil {
			return err
		}
		srvCtx.http.listeners[opts.addrPort] = l
	}
	for k, rules := range srvCtx.http.serverRules {
		opts := srvCtx.http.address[k]
		srv, err := createHTTPServer(ctx, srvCtx, rules, opts)
		if err != nil {
			return err
		}
		srvCtx.http.servers[opts.addrPort] = srv
	}

	// we can start servers now
	for opts, srv := range srvCtx.http.servers {
		fmt.Printf("[vince] starting server on %q\n", srvCtx.http.listeners[opts].Addr().String())
		go srv.Serve(srvCtx.http.listeners[opts])
	}
	return nil
}

func startEverything(mainCtx context.Context, config *vinceConfiguration, ready ...func()) error {
	ctx, cancel := context.WithCancel(mainCtx)
	defer cancel()
	p := parse(config.confFile, templates.IncludeFS, defaultParseOpts())
	if p.Errors != nil {
		return fmt.Errorf("vince: parsing config %v", p.Errors)
	}
	d := &Stmt{Directive: "main"}
	d.Blocks = p.Config[0].Parsed
	var srvCtx serverCtx
	srvCtx.init(ctx, d, config)
	ctx = context.WithValue(ctx, ngxLoggerKey{}, &cacheLogger{
		cache: srvCtx.fileCache,
	})
	defer func() {
		srvCtx.shutdown(context.Background())
	}()

	if err := process(ctx, &srvCtx, config); err != nil {
		return err
	}
	if config.management.enabled {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", config.management.port))
		if err != nil {
			// TODO log
			fmt.Println("vince: failed to start management server ", err)
		} else {
			srvCtx.http.address[l.Addr().String()] = httpListenOpts{
				net:      l.Addr().Network(),
				addrPort: l.Addr().String(),
			}
			srvCtx.http.listeners[l.Addr().String()] = l
			m := new(management)
			m.init(&srvCtx)
			srv := &http.Server{Handler: m}
			srvCtx.http.servers[l.Addr().String()] = srv
			fmt.Println("vince: staring management server at ", l.Addr().String())
			go srv.Serve(l)
		}
	}
	if len(ready) > 0 {
		ready[0]()
	}
	ch := make(chan os.Signal, 2)
	signal.Notify(
		ch,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGABRT,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGWINCH,
	)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-ch:
			fmt.Println("vince: received signal " + sig.String())
			switch sig {
			case syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT:
				return errors.New("exiting")
			case syscall.SIGQUIT:
				fmt.Println("Shutting down")
				return srvCtx.shutdown(ctx)
			case syscall.SIGHUP:
			case syscall.SIGUSR1:
			case syscall.SIGUSR2:
			case syscall.SIGWINCH:
			}
		}
	}
}

type serverCtx struct {
	core   *rule
	config *vinceConfiguration
	http   struct {
		defaultServer  map[string]*rule
		address        map[string]httpListenOpts
		serverRules    map[string][]*rule
		listeners      map[string]net.Listener
		servers        map[string]*http.Server
		connManager    *connManager
		activeListener *httpListenOpts
	}
	fileCache *readWriterCloserCache
}

func (s *serverCtx) with(active httpListenOpts) *serverCtx {
	n := new(serverCtx)
	n.core = s.core
	n.http.serverRules = s.http.serverRules
	n.http.listeners = s.http.listeners
	n.http.servers = s.http.servers
	n.http.activeListener = &active
	n.fileCache = s.fileCache
	n.http.connManager = s.http.connManager
	n.config = s.config
	return n
}

func (s *serverCtx) handle(r *rule) func(handler) handler {
	switch r.name {
	case "proxy_pass":
		p := new(proxy)
		p.init(r.parent, baseTransport)
		return wrap(p, true)
	case "allow":
		a := new(nginxAccess)
		a.init(r.args[0], true)
		return a.handle
	case "deny":
		a := new(nginxAccess)
		a.init(r.args[0], false)
		return a.handle
	default:
		return nextHandler
	}
}

func wrap(h handler, halt bool) func(handler) handler {
	return func(next handler) handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
			if halt {
				return // we are done evaluating the chain
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *serverCtx) shutdown(ctx context.Context) error {
	var errs []string
	// 1 shut down all http servers
	for _, srv := range s.http.servers {
		if err := srv.Shutdown(ctx); err != nil {
			errs = append(errs, err.Error())
		}
	}
	// 2 close all listeners
	for _, l := range s.http.listeners {
		if err := l.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	// 3 close all managed connections. This includes hijacked connections.
	if err := s.http.connManager.Close(); err != nil {
		errs = append(errs, err.Error())
	}
	if errs != nil {
		return fmt.Errorf("vince: error trying to graceful shutdown %q", strings.Join(errs, ","))
	}
	return nil
}

func (s *serverCtx) chain(r ...*rule) alice {
	a := alice{accessLogMiddlewareFunc()}
	for _, v := range r {
		a = append(a, s.handle(v))
	}
	return a
}

func nextHandler(next handler) handler {
	return next
}

func (s *serverCtx) init(ctx context.Context, stmt *Stmt, cfg *vinceConfiguration) {
	s.http.address = make(map[string]httpListenOpts)
	s.http.serverRules = make(map[string][]*rule)
	s.http.listeners = make(map[string]net.Listener)
	s.http.servers = make(map[string]*http.Server)

	core := ruleFromStmt(stmt, nil)
	s.core = core
	s.config = cfg
	var fo readWriterCloserCacheOption
	fo.defaults()
	s.fileCache = new(readWriterCloserCache)
	s.fileCache.initFile(ctx, fo)

	s.http.connManager = new(connManager)
	s.http.connManager.init()
}

func createHTTPServer(ctx context.Context, srv *serverCtx, servers []*rule, opts httpListenOpts) (*http.Server, error) {
	servCtx := srv.with(opts)
	ctx = context.WithValue(ctx, serverCtxKey{}, servCtx)
	s := &http.Server{}
	s.BaseContext = func(ls net.Listener) context.Context {
		return srv.http.connManager.baseCtx(ctx, ls)
	}
	s.ConnState = srv.http.connManager.manageConnState
	s.ConnContext = srv.http.connManager.connContext
	s.Handler = vinceHandler(ctx, servers)
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

func (ls *locationMatch) match(path string) *match {
	for i := 0; i < len(ls.rules); i++ {
		if ls.rules[i].kind == matchExact && ls.rules[i].rule.args[1] == path {
			return ls.rules[i]
		}
	}
	var m []*match
	for i := 0; i < len(ls.rules); i++ {
		switch ls.rules[i].kind {
		case matchPrefix:
			if strings.HasPrefix(path, ls.rules[i].rule.args[0]) {
				m = append(m, ls.rules[i])
			}
		case matchCaret:
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
			return selected
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
					return up.rules[i]
				}
			}
		}
	}
	if up != nil {
		for i := 0; i < len(up.rules); i++ {
			if ls.rules[i].kind == matchRegexp {
				if ls.rules[i].re.MatchString(path) {
					return ls.rules[i]
				}
			}
		}
	}

	if selected != nil {
		return selected
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

type handlerMatch struct {
	regExp        map[*regexp.Regexp]*rule
	exact         map[string][]*rule
	wild          map[string]*rule
	defaultServer *rule
}

func (h *handlerMatch) init(servers []*rule, defaultServer *rule) {
	h.defaultServer = defaultServer
	h.regExp = make(map[*regexp.Regexp]*rule)
	h.exact = make(map[string][]*rule)
	h.wild = make(map[string]*rule)
	for _, srv := range servers {
		for _, ch := range srv.children {
			if ch.name == "server_name" {
				for _, a := range ch.args {
					if a[0] == '~' {
						re := regexp.MustCompile(a[1:])
						h.regExp[re] = srv
						continue
					}
					if isWildCard(a) {
						h.wild[a] = srv
						continue
					}
					if v, ok := h.exact[a]; ok {
						h.exact[a] = append(v, srv)
					} else {
						h.exact[a] = []*rule{srv}
					}
				}
				break
			}
		}
	}
}

func (h *handlerMatch) find(name string) *rule {
	if r, ok := h.exact[name]; ok {
		return r[0]
	}
	if len(h.wild) > 0 {
		var match string
		for w := range h.wild {
			if matchWildCard(name, w) {
				if w > match {
					match = w
				}
			}
		}
		if match != "" {
			return h.wild[match]
		}
	}
	for re, r := range h.regExp {
		if re.MatchString(name) {
			return r
		}
	}
	return h.defaultServer
}

func vinceHandler(ctx context.Context, servers []*rule) http.Handler {
	srvCtx := ctx.Value(serverCtxKey{}).(*serverCtx)
	var hm handlerMatch
	hm.init(servers, srvCtx.http.defaultServer[srvCtx.http.activeListener.addrPort])

	location := new(sync.Map)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		variable := ctx.Value(variables{}).(map[string]interface{})
		setRequestVariables(variable, r)
		var srv *rule
		if len(servers) == 1 {
			srv = servers[0]
		} else {
			srv = hm.find(r.Host)
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
			c := l.rule.collect(nil)
			variable[vRequestMatchKind] = l
			srvCtx.chain(overide(c)...).then(nil).ServeHTTP(w, r)
			return
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})
}

type handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type alice []func(handler) handler

func (a alice) then(h handler) handler {
	if h == nil {
		h = notFoundHandler{}
	}
	for i := range a {
		h = a[len(a)-1-i](h)
	}
	return h
}

type notFoundHandler struct{}

func (notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func findListener(r *rule, port int) []httpListenOpts {
	p := strconv.Itoa(port)
	var ls []httpListenOpts
	for _, v := range r.children {
		if v.name == "listen" {
			ls = append(ls, parseListen(v, p))
			continue
		}
	}
	return ls
}

func start(ctx *cli.Context) error {
	c, err := getConfig(ctx)
	if err != nil {
		return err
	}
	return startEverything(context.Background(), c)
}
