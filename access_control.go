package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

var accessControlPrefix = []byte("/access/")
var accessControlModelPrefix = []byte("/access/model")

type accessAllowed struct{}

func passthrough(ctx context.Context) bool {
	if v := ctx.Value(accessAllowed{}); v != nil {
		return v.(bool)
	}
	return false
}

var _ persist.Adapter = (*accessControlAdapter)(nil)

type accessControlAdapter struct {
	store kvStore
	file  []byte
}

func (a *accessControlAdapter) LoadPolicy(m model.Model) error {
	b, err := a.store.get(joinSlice(accessControlModelPrefix, a.file))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &m)
}

func (a *accessControlAdapter) SavePolicy(m model.Model) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return a.store.set(joinSlice(accessControlModelPrefix, a.file), b)
}

func (a *accessControlAdapter) AddPolicy(sec string, ptype string, rule []string) error {
	m := make(model.Model)
	if err := a.LoadPolicy(m); err != nil {
		return err
	}
	m.AddPolicy(sec, ptype, rule)
	return a.SavePolicy(m)
}

func (a *accessControlAdapter) RemovePolicy(sec string, ptype string, rule []string) error {
	m := make(model.Model)
	if err := a.LoadPolicy(m); err != nil {
		return err
	}
	m.RemovePolicy(sec, ptype, rule)
	return a.SavePolicy(m)
}

func (a *accessControlAdapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	m := make(model.Model)
	if err := a.LoadPolicy(m); err != nil {
		return err
	}
	m.RemoveFilteredPolicy(sec, ptype, fieldIndex, fieldValues...)
	return a.SavePolicy(m)
}

type accessControl struct {
	adopt   *accessControlAdapter
	enforce *casbin.Enforcer
}

func (a *accessControl) init(adopt *accessControlAdapter) error {
	e := new(casbin.Enforcer)
	if err := e.InitWithModelAndAdapter(make(model.Model), adopt); err != nil {
		return err
	}
	a.adopt = adopt
	a.enforce = e
	adopt.store.onSet(a.reload)
	adopt.store.onRemove(a.reload)
	return nil
}

func (a *accessControl) with(file string) (*accessControl, error) {
	n := new(accessControl)
	err := n.init(&accessControlAdapter{file: []byte(file), store: a.adopt.store.clone()})
	return n, err
}

func (a *accessControl) Enforce(vals ...interface{}) (bool, error) {
	return a.enforce.Enforce(vals...)
}

func (a *accessControl) reload(key []byte) {
	if bytes.HasPrefix(key, accessControlModelPrefix) {
		err := a.enforce.LoadPolicy()
		if err != nil {
			// TODO:(gernest) log error
		}
	}
}

type nginxAccess struct {
	allow bool
	addr  string
	match func(*http.Request) bool
}

func (a *nginxAccess) init(addr string, allow bool) {
	a.allow = allow
	a.addr = addr
	a.match = func(_ *http.Request) bool { return false }
	if addr == "all" {
		a.match = func(_ *http.Request) bool { return true }
	} else if addr == "unix:" {
		a.match = func(r *http.Request) bool {
			return r.RemoteAddr == ""
		}
	} else if ip := net.ParseIP(addr); ip != nil {
		a.match = a.matchIP(ip)
	}
}

func (a *nginxAccess) matchIP(ip net.IP) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		if ap := realIP(r); ap != nil {
			return ip.Equal(ap)
		}
		return false
	}
}

func realIP(r *http.Request) net.IP {
	return net.ParseIP(realIPString(r))
}

func realIPString(r *http.Request) string {
	if ip := r.Header.Get(HeaderXForwardedFor); ip != "" {
		return strings.Split(ip, ", ")[0]
	}
	if ip := r.Header.Get(HeaderXRealIP); ip != "" {
		return ip
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

func (a *nginxAccess) handle(next handler) handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if !passthrough(ctx) {
			if a.allow {
				if !a.match(r) {
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}
			} else {
				if a.match(r) {
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, accessAllowed{}, true)))
			return
		}
		next.ServeHTTP(w, r)
	})
}
