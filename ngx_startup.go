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

func process(ctx context.Context, sctx *serverCtx, config *vinceConfiguration) error {
	var servers []*rule
	// main block
	for _, base := range sctx.core.children {
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
			if _, ok := sctx.address[ls.addrPort]; !ok {
				sctx.address[ls.addrPort] = &ls
			}
			if a, ok := sctx.ls1[ls.addrPort]; ok {
				sctx.ls1[ls.addrPort] = append(a, v)
			} else {
				sctx.ls1[ls.addrPort] = []*rule{v}
			}
		}
	}
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
		if err != nil {
			return err
		}
		sctx.ls2[opts.addrPort] = l
	}
	for k, rules := range sctx.ls1 {
		opts := sctx.address[k]
		srv, err := createHTTPServer(context.WithValue(ctx, serverCtxKey{}, sctx.with(opts)), rules, opts)
		if err != nil {
			return err
		}
		sctx.ls3[opts.addrPort] = srv
	}

	// we can start servers now
	for opts, srv := range sctx.ls3 {
		fmt.Printf("[vince] starting server on %q\n", sctx.ls2[opts].Addr().String())
		go srv.Serve(sctx.ls2[opts])
	}
	return nil
}

func startEverything(mainCtx context.Context, config *vinceConfiguration) error {
	ctx, cancel := context.WithCancel(mainCtx)
	defer cancel()
	fs, err := templates.NewIncludeFS()
	if err != nil {
		return err
	}
	p := parse(config.confFile, fs, defaultParseOpts())
	if p.Errors != nil {
		return fmt.Errorf("vince: parsing config %v", p.Errors)
	}
	d := &Stmt{Directive: "main"}
	d.Blocks = p.Config[0].Parsed
	core := ruleFromStmt(d, nil)
	sctx := newSrvCtx()
	sctx.core = core

	defer func() {
		// make sure all listeners are closed before exiting
		for _, l := range sctx.ls2 {
			l.Close() // TODO:(gernest) handle error
		}
	}()

	if err := process(ctx, sctx, config); err != nil {
		return err
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
		sig := <-ch
		fmt.Println("vince: received signal " + sig.String())
		switch sig {
		case syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT:
			return errors.New("exiting")
		case syscall.SIGQUIT:
			fmt.Println("Shutting down")
			return sctx.shutdown(ctx)
		case syscall.SIGHUP:
		case syscall.SIGUSR1:
		case syscall.SIGUSR2:
		case syscall.SIGWINCH:
		}
	}
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

func (s *serverCtx) handle(r *rule) func(http.Handler) http.Handler {
	switch r.name {
	default:
		return nextHandler
	}
}

func (s *serverCtx) shutdown(ctx context.Context) error {
	var errs []string
	for _, srv := range s.ls3 {
		if err := srv.Shutdown(ctx); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if errs != nil {
		return fmt.Errorf("vince: error trying to graceful shutdown %q", strings.Join(errs, ","))
	}
	return nil
}

func (s *serverCtx) chain(r ...*rule) alice {
	var a alice
	for _, v := range r {
		a = append(a, s.handle(v))
	}
	return a
}

func nextHandler(next http.Handler) http.Handler {
	return next
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
	if up != nil {
		for i := 0; i < len(up.rules); i++ {
			if ls.rules[i].kind == matchRegexp {
				if ls.rules[i].re.MatchString(path) {
					return ls.rules[i].rule
				}
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
		ctx := r.Context()
		variable := ctx.Value(variables{}).(*sync.Map)
		setRequestVariables(variable, r)
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
			c := l.collect(nil)
			// we start by executing child block
			block := sctx.chain(l.children...)
			// then we traverse the parents
			parent := sctx.chain(overide(c)...)
			block.then(parent.then(noopHandler{})).ServeHTTP(w, r)
		}
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	})
}

type alice []func(http.Handler) http.Handler

func (a alice) then(h http.Handler) http.Handler {
	if h == nil {
		h = noopHandler{}
	}
	for i := range a {
		h = a[len(a)-1-i](h)
	}
	return h
}

type noopHandler struct{}

func (noopHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

func findListener(r *rule, port int) []listenOpts {
	p := strconv.Itoa(port)
	var ls []listenOpts
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
