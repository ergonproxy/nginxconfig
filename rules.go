package main

//go:generate go run scripts/nginx_configuration.go
import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type (
	pathKey struct{}
)

type rule struct {
	name string
	args []interface{}
}

type ruleApply interface {
	apply(context.Context)
}

type ruleApplyChain func(ruleApply) ruleApply

type ruleApplyFn func(context.Context)

func (f ruleApplyFn) apply(ctx context.Context) { f(ctx) }

type ruleList []ruleApplyChain

func (r ruleList) apply(ctx context.Context) {
	var f ruleApply
	for _, v := range r {
		f = v(f)
	}
	if f != nil {
		f.apply(ctx)
	}
}

func set(key, value interface{}) ruleApplyChain {
	return func(next ruleApply) ruleApply {
		return ruleApplyFn(func(ctx context.Context) {
			next.apply(context.WithValue(ctx, key, value))
		})
	}
}

func interpret(ctx context.Context, value interface{}) interface{} {
	s, ok := value.(string)
	if !ok {
		// we only interpret string values
		return value
	}
	if !strings.Contains(s, "$") {
		return value
	}
	// if we can retrive full value from context then its good
	if v := ctx.Value(s); v != nil {
		return v
	}
	return resolveVariables(ctx, []byte(s))
}

func ngxAlias(path string) ruleApplyChain {
	return ruleKVChain(pathKey{}, path)
}

func ngxAbsoluteRedirect(ok string) ruleApplyChain {
	value := boolValue(ok)
	return ruleKVChain(absoluteRedirectKey{}, value)
}

func boolValue(value string) interface{} {
	if value == "" {
		return false
	}
	switch value {
	case "on", "true":
		return true
	case "off", "false":
		return false
	default:
		return value
	}
}

func ruleKVChain(key, value interface{}) ruleApplyChain {
	return func(next ruleApply) ruleApply {
		return ruleApplyFn(func(ctx context.Context) {
			next.apply(context.WithValue(ctx, key, value))
		})
	}
}

var variableRegexp = regexp.MustCompile(`\$([a-z_]\w*)`)

func resolveVariables(ctx context.Context, src []byte) []byte {
	return variableRegexp.ReplaceAllFunc(src, func(name []byte) []byte {
		n := string(name)
		if v := ctx.Value(n); v != nil {
			return toByte(v)
		}
		return []byte{}
	})
}

func toByte(v interface{}) []byte {
	switch e := v.(type) {
	case []byte:
		return e
	case string:
		return []byte(e)
	default:
		return []byte(fmt.Sprint(v))
	}
}
