package engine

import (
	"net"
	"net/http"
	"regexp"
	"strings"
)

// Matcher selsects a handler to use from request.
type Matcher interface {
	Match(*http.Request) http.Handler
}

type ServerNameMatchKind uint

const (
	Empty ServerNameMatchKind = iota
	Exact
	Regexp
	WildCard
	All
	IP
)

type StringMatchFn func(string) bool

func isWildCardName(s string) bool {
	if s == "" {
		return false
	}
	if i := strings.IndexByte(s, '*'); i != -1 {
		if i != 0 || i != len(s)-1 {
			// * must be at the beginning or at the end
			return false
		}
		// * must be on the dot border
		if i == 0 && len(s) > 1 && s[1] == '.' {
			return true
		}
		if i == len(s)-1 && len(s) > 1 && s[len(s)-2] == '.' {
			return true
		}
	}
	return false
}

func GetServerNameMatchKind(name string) ServerNameMatchKind {
	if name == "" {
		return Empty
	}
	if name[0] == '~' {
		return Regexp
	}
	if isWildCardName(name) {
		return WildCard
	}
	return Exact
}

func matchExact(with string) StringMatchFn {
	return func(name string) bool {
		return name == with
	}
}

func matchRegexp(exp string) StringMatchFn {
	re := regexp.MustCompile(exp[1:])
	return func(name string) bool {
		return re.MatchString(name)
	}
}

func matchIP(with string) StringMatchFn {
	ip := net.ParseIP(with)
	return func(name string) bool {
		if ip == nil {
			return false
		}
		return ip.Equal(net.ParseIP(name))
	}
}

func matchWildCard(with string) StringMatchFn {
	x := strings.Split(with, ".")
	return func(name string) bool {
		n := strings.Split(name, ".")
		if len(n) != len(x) {
			return false
		}
		for k, v := range x {
			if v == "x" {
				continue
			}
			if v != n[k] {
				return false
			}
		}
		return true
	}
}

func matchAll(_ string) bool {
	return true
}

func ServerNameMacthes(names ...string) StringMatchFn {
	var ls []StringMatchFn
	for _, name := range names {
		switch GetServerNameMatchKind(name) {
		case Empty:
			ls = append(ls, matchAll)
		case Exact:
			ls = append(ls, matchExact(name))
		case WildCard:
			ls = append(ls, matchWildCard(name))
		case Regexp:
			ls = append(ls, matchRegexp(name))
		case All:
			ls = append(ls, matchAll)
		case IP:
			ls = append(ls, matchIP(name))
		}
	}
	return func(name string) bool {
		for _, fn := range ls {
			if fn(name) {
				return true
			}
		}
		return false
	}
}
