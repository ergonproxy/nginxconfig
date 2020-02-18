package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
)

type proxy struct {
	opts    proxyOption
	bad     bool
	rev     *httputil.ReverseProxy
	origURL *url.URL
}

type proxyOption struct {
	bind struct {
		address     stringValue
		transparent boolValue
		off         boolValue
	}
	buffer struct {
		size   intValue
		enable boolValue
	}
	cache struct {
		enabled boolValue
		bypass  stringSliceValue
		key     stringValue
	}
	pass struct {
		uri      stringTemplateValue
		header   stringSliceValue
		body     boolValue
		headers  boolValue
		method   stringValue
		redirect struct {
			isDefault boolValue
			off       boolValue
			replace   stringSliceValue
		}
	}
}

func (o *proxyOption) load(location *rule) {
	switch location.name {
	case "http", "server", "location": //pass
	default:
		return
	}
	if location.parent != nil {
		o.load(location.parent)
	}
	for _, v := range location.children {
		o.loadKey(v)
	}
}

func (o *proxyOption) loadKey(r *rule) {
	switch r.name {
	case "proxy_bind":
		o.bind.address.store(r.args[0])
		for i := 0; i < len(r.args); i++ {
			switch r.args[i] {
			case "transparent":
				o.bind.transparent.store(true)
			case "off":
				o.bind.off.store(true)
			}
		}
	case "proxy_pass":
		o.pass.uri.store(r.args[0])
	case "proxy_pass_header":
		o.pass.header.store(r.args[0])
	case "proxy_pass_request_body":
		switch r.args[0] {
		case "on":
			o.pass.body.store(true)
		case "off":
			o.pass.body.store(false)
		}
	case "proxy_method":
		o.pass.method.store(r.args[0])
	}
}

var baseTransport = &unixTransport{}

type unixTransport struct {
	transport http.Transport
	once      sync.Once
}

func (t *unixTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.HasPrefix(r.URL.Host, "unix:") {
		return t.getTransport().RoundTrip(r)
	}
	return http.DefaultTransport.RoundTrip(r)
}

func (t *unixTransport) getTransport() *http.Transport {
	t.once.Do(t.init)
	return &t.transport
}

func (t *unixTransport) init() {
	t.transport.DialContext = t.dialCtx
	t.transport.DialTLS = t.dialTLS
}

func (t *unixTransport) dialCtx(ctx context.Context, network, address string) (net.Conn, error) {
	var d net.Dialer
	h, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	return d.DialContext(ctx, "unix", h[5:])
}

func (t *unixTransport) dialTLS(network, address string) (net.Conn, error) {
	return nil, errors.New("vince: tls over unix socket is not supported")
}

func (p *proxy) init(location *rule, transport http.RoundTripper) {
	p.opts = proxyOption{}
	p.opts.load(location)
	p.rev = new(httputil.ReverseProxy)
	p.rev.Director = p.director
	p.rev.Transport = transport
	p.rev.ModifyResponse = p.modifyResponse
}

func (p *proxy) director(r *http.Request) {
	ctx := r.Context()
	v := ctx.Value(variables{}).(map[string]interface{})
	target := p.opts.pass.uri.Value(v)
	u, _ := parseProxyURL(target)
	p.origURL = r.URL
	if k, ok := v[vRequestMatchKind]; ok {
		m := k.(*match)
		switch m.kind {
		case matchPrefix:
			if u.Path == "/" && r.URL.Path != "/" {
				p := strings.TrimPrefix(r.URL.Path, m.rule.args[0])
				u.Path += strings.TrimPrefix(p, "/")
			}
		}
	}
	r.URL = u
	if p.opts.pass.body.set && !p.opts.pass.body.value {
		r.Body = nil
	}
	if p.opts.pass.method.set {
		r.Method = p.opts.pass.method.value
	}
}

func parseProxyURL(s string) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if u.Host != "unix:" {
		return u, nil
	}
	p := strings.Split(u.Path, ":")
	u.Host += p[0]
	u.Path = p[1]
	return u, nil
}
func (p *proxy) modifyResponse(w *http.Response) error {
	//proxy_redirect
	if p.opts.pass.redirect.isDefault.set {
		if l := w.Header.Get("Location"); l != "" {
			w.Header.Set("Location", p.origURL.RawPath)
		}
	}
	return nil
}

func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := p.valid(r); err != nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	p.rev.ServeHTTP(w, r)
}

func (p *proxy) valid(r *http.Request) error {
	if !p.opts.pass.uri.set {
		return errors.New("vince: proxy_pass url not set")
	}
	target := p.eval(p.opts.pass.uri.value, r)
	_, err := url.Parse(target)
	if err != nil {
		return err
	}
	return nil
}

func (p *proxy) eval(key string, r *http.Request) string {
	return key
}
