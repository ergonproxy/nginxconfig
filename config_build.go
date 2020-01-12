package main

import (
	"io"
	"strconv"
	"strings"
	"unicode"
)

type runeIter struct {
	v   []rune
	idx int
}

func (r *runeIter) next() (rune, error) {
	if r.idx < len(r.v) {
		r.idx++
		return r.v[r.idx-1], nil
	}
	return 0, io.EOF
}

func (r *runeIter) peek() (rune, error) {
	if r.idx < len(r.v) {
		return r.v[r.idx], nil
	}
	return 0, io.EOF
}

func escapeToIter(s string) *runeIter {
	return &runeIter{
		v: escape(s),
	}
}

func escape(src string) []rune {
	var buf []rune
	var prev rune
	var ch rune
	for _, ch = range src {
		if prev == '\\' || (ch == '{' && prev == '$') {
			buf = append(buf, prev)
			buf = append(buf, ch)
			prev = ch
			continue
		}
		if prev == '$' {
			buf = append(buf, prev)
		}
		if ch != '\\' && ch != '$' {
			buf = append(buf, ch)
		}
		prev = ch
	}
	if ch == '\\' || ch == '$' {
		buf = append(buf, ch)
	}
	return buf
}

func enquote(arg string) string {
	if !needsQuote(arg) {
		return arg
	}
	arg = strings.Replace(arg, `\\\\`, `\\`, -1)
	return strconv.Quote(arg)
}

func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	it := escapeToIter(s)
	ch, err := it.next()
	if err != nil {
		return false
	}
	if unicode.IsSpace(ch) {
		return true
	}
	switch ch {
	case '{', '}', ';', '"', '\'':
		return true
	case '$':
		n, _ := it.peek()
		if n == '{' {
			return true
		}
	}
	ch, err = it.next()
	expanding := false
	for err == nil {
		if check01(ch) {
			return true
		} else if check002(it, ch, expanding) {
			return true
		} else if check003(it, ch, expanding) {
			expanding = !expanding
		}
		ch, err = it.next()
	}
	switch ch {
	case '\\', '$':
		return true
	default:
		return expanding
	}
}

func check01(ch rune) bool {
	if unicode.IsSpace(ch) {
		return true
	}
	switch ch {
	case '{', '}', ';', '"', '\'':
		return true
	}
	return false
}

func check002(it *runeIter, ch rune, expanding bool) bool {
	if expanding {
		if ch == '$' {
			p, _ := it.peek()
			if p == '$' {
				return true
			}
		}
	} else if ch == '}' {
		return true
	}
	return false
}

func check003(it *runeIter, ch rune, expanding bool) bool {
	if expanding {
		return ch == '}'
	}
	if ch == '$' {
		p, _ := it.peek()
		if p == '$' {
			return true
		}
	}
	return false
}

func build(p []*Stmt, indent int, header bool) string {
	o := ""
	if header {
		o = `# This config was built from JSON using NGINX crossplane.
		# If you encounter any bugs please report them here:
		# https://github.com/nginxinc/crossplane/issues
		`
	}
	return buildBlock(o, defaultCustomBuilder(), p, indent, 0, 0)
}

type customBuilder interface {
	build(buf string, stmt *Stmt, padding, depth int) string
}

func defaultCustomBuilder() map[string]customBuilder {
	lua := luaLexer{}
	m := make(map[string]customBuilder)
	for _, directive := range lua.directives() {
		m[directive] = lua
	}
	return m
}
func buildBlock(output string, custom map[string]customBuilder, block []*Stmt, padding int, depth, lastLine int) string {
	m := margin(padding, depth)
	for _, stmt := range block {
		built := ""
		directive := enquote(stmt.Directive)
		if directive == "#" && stmt.Line == lastLine {
			output += " #" + stmt.Comment
			continue
		} else if directive == "#" {
			built += " #" + stmt.Comment
		} else if c, ok := custom[directive]; ok {
			built = c.build(built, stmt, padding, depth)
		} else {
			var args []string
			if len(stmt.Args) > 0 {
				args = make([]string, len(stmt.Args))
				for i := 0; i < len(stmt.Args); i++ {
					args[i] = enquote(stmt.Args[i])
				}
			}
			if directive == "if" {
				built = " if" + strings.Join(args, " ") + ")"
			} else if args != nil {
				built = directive + " " + strings.Join(args, " ")
			} else {
				built = directive
			}
			if len(stmt.Blocks) > 0 {
				built += " {"
				built = buildBlock(built, custom, stmt.Blocks, padding, depth+1, stmt.Line)
				built += "\n" + m + "}"
			} else {
				built += ";"
			}
		}
		if len(output) > 0 {
			output += "\n"
		}
		output += m + built
		lastLine = stmt.Line
	}
	return output
}

func margin(padding int, depth int) string {
	x := padding * depth
	o := ""
	for i := 0; i < x; i++ {
		o += " "
	}
	return o
}
