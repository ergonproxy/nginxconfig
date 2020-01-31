package main

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
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
		uri      stringValue
		header   stringSliceValue
		body     boolValue
		headers  boolValue
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
	}
}

func (p *proxy) init(location *rule, transport http.RoundTripper, eval func(string) string) {
	p.opts = proxyOption{}
	p.opts.load(location)
	p.rev = new(httputil.ReverseProxy)
	p.rev.Director = p.director
	p.rev.Transport = transport
	p.rev.ModifyResponse = p.modifyResponse
}

func (p *proxy) director(r *http.Request) {
	ctx := r.Context()
	v := ctx.Value(variables{}).(*sync.Map)
	target := eval(v, p.opts.pass.uri.value)
	u, _ := url.Parse(target)
	p.origURL = r.URL
	r.URL = u
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
