package main

import (
	"io"
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
	return strings.Replace(arg, `\\`, `\`, -1)
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

func build(buf *strings.Builder, p []*Stmt, indent int, header bool) {
	if header {
		buf.WriteString(`# This config was built from JSON using NGINX crossplane.
		# If you encounter any bugs please report them here:
		# https://github.com/nginxinc/crossplane/issues
		`)
	}
	buildBlock(buf, p, indent, 0, 0)
}

func buildBlock(buf *strings.Builder, block []*Stmt, padding int, depth, lastLine int) {
	built := new(strings.Builder)
	padBuild := func() {
		margin(built, padding, depth)
	}
	padBuf := func() {
		margin(buf, padding, depth)
	}
	for _, stmt := range block {
		built.Reset()
		directive := enquote(stmt.Directive)
		if directive == "#" && stmt.Line == lastLine {
			buf.WriteByte('#')
			buf.WriteString(stmt.Comment)
			continue
		} else if directive == "#" {
			built.WriteRune('#')
			built.WriteString(stmt.Comment)
		} else {
			var args []string
			if len(stmt.Args) > 0 {
				args = make([]string, len(stmt.Args))
				for i := 0; i < len(stmt.Args); i++ {
					args[i] = enquote(stmt.Args[i])
				}
			}
			built.WriteString(directive)
			if directive == "if" {
				built.WriteString(" (")
				built.WriteString(strings.Join(args, " "))
				built.WriteRune(')')
			} else if args != nil {
				built.WriteString(strings.Join(args, " "))
			}
			if len(stmt.Blocks) > 0 {
				built.WriteString(" {")
				buildBlock(built, stmt.Blocks, padding, depth+1, stmt.Line)
				built.WriteRune('\n')
				padBuild()
				buf.WriteRune('}')
			} else {
				built.WriteRune(';')
			}
		}
		if buf.Len() > 0 {
			buf.WriteRune('\n')
		}
		padBuf()
		buf.WriteString(built.String())
		lastLine = stmt.Line
	}
}

func margin(buf *strings.Builder, padding int, depth int) {
	x := padding * depth
	for i := 0; i < x; i++ {
		buf.WriteRune(' ')
	}
}
