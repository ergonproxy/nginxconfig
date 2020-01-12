package main

import (
	"bufio"
	"fmt"
	"io"
	"unicode"
)

// NgxError this is error returned when parsing/lexing nginx configuration file.
type NgxError struct {
	Reason   string
	Linenum  int
	Filename string
}

func newError(reason string, line int, filename string) NgxError {
	return NgxError{Reason: reason, Linenum: line, Filename: filename}
}

func (n NgxError) Error() string {
	return fmt.Sprintf("%s:%d %s", n.Filename, n.Linenum, n.Reason)
}

// NgxParserDirectiveUnknownError is returned when parsing unknown directive
type NgxParserDirectiveUnknownError struct {
	NgxError
}

// NgxParserDirectiveContextError is returned when directive is in the wrong
// context.
type NgxParserDirectiveContextError struct {
	NgxError
}

// NgxParserDirectiveArgumentsError is returned for invalid arguments
type NgxParserDirectiveArgumentsError struct {
	NgxError
}

// Iter reads one rune at a time, returns io.EOF if it has reached the end on
// the stream.
type Iter interface {
	Next() (rune, error)
}

// IterLine returns a rune and line number on which the rune was read.
type IterLine interface {
	Next() (rune, int, error)
}

type lineCounter struct {
	Iter
	line int
}

func (ls *lineCounter) Next() (rune, int, error) {
	ch, err := ls.Iter.Next()
	if err != nil {
		return 0, 0, err
	}
	if ch == '\n' {
		ls.line++
	}
	return ch, ls.line, nil
}

type customLexer interface {
	lex(IterLine, string) ([]token, error)
}

type lexer struct {
	token                string
	tokens               []token
	line                 int
	nextTokenIsDirective bool
	iter                 IterLine
	externalLexer        map[string]customLexer
}

type token struct {
	text  string
	line  int
	quote bool
}

func defaultCustomLexers() map[string]customLexer {
	lua := luaLexer{}
	m := map[string]customLexer{}
	for _, directive := range lua.directives() {
		m[directive] = lua
	}
	return m
}

func newLexer(iter Iter, custom map[string]customLexer) *lexer {
	return &lexer{
		iter:          &lineCounter{Iter: iter, line: 1},
		externalLexer: custom,
	}
}

func (lx *lexer) lex() error {
	for {
	again:
		ch, line, err := lx.iter.Next()
		if err == io.EOF {
			return nil
		}
		if unicode.IsSpace(ch) {
			if lx.token != "" {
				lx.tokens = append(lx.tokens, token{
					text:  lx.token,
					line:  lx.line,
					quote: false,
				})
				if clx, ok := lx.custom(lx.token); ok {
					lexTokens, err := clx.lex(lx.iter, lx.token)
					if err != nil {
						return err
					}
					lx.tokens = append(lx.tokens, lexTokens...)
				} else {
					lx.nextTokenIsDirective = false
				}
				lx.token = ""
			}
			for unicode.IsSpace(ch) {
				ch, line, err = lx.iter.Next()
			}
		}
		if lx.token == "" && ch == '#' {
			for ch != '\n' {
				lx.token += string(ch)
				ch, _, err = lx.iter.Next()
			}
			lx.tokens = append(lx.tokens, token{
				text:  lx.token,
				line:  line,
				quote: false,
			})
			lx.token = ""
			goto again
		}
		if ch == '\\' {
			// escapes
			lx.token += string(ch)
			ch, line, err = lx.iter.Next()
			lx.token += string(ch)
			goto again
		}
		if lx.token == "" {
			lx.line = line
		}
		if lx.token != "" && lx.token[len(lx.token)-1] == '$' && ch == '{' {
			lx.nextTokenIsDirective = false
			for lx.token[len(lx.token)-1] != '}' && !unicode.IsSpace(ch) {
				lx.token += string(ch)
				ch, line, err = lx.iter.Next()
			}
		}
		if ch == '"' || ch == '\'' {
			if lx.token != "" {
				lx.token += string(ch)
				goto again
			}
			quote := ch
			ch, line, err = lx.iter.Next()
			for ch != quote {
				if ch == '\\' {
					ch, line, err = lx.iter.Next()
					if ch == quote {
						lx.token += string(quote)
					} else {
						lx.token += "\\" + string(ch)
					}
				} else {
					lx.token += string(ch)
				}
				ch, line, err = lx.iter.Next()
			}
			lx.tokens = append(lx.tokens, token{
				text:  lx.token,
				line:  lx.line,
				quote: true,
			})
			if clx, ok := lx.custom(lx.token); ok {
				lexTokens, err := clx.lex(lx.iter, lx.token)
				if err != nil {
					return err
				}
				lx.tokens = append(lx.tokens, lexTokens...)
				lx.nextTokenIsDirective = true
			} else {
				lx.nextTokenIsDirective = false
			}
			lx.token = ""
			goto again
		}
		if ch == '{' || ch == '}' || ch == ';' {
			if lx.token != "" {
				lx.tokens = append(lx.tokens, token{
					text: lx.token,
					line: lx.line,
				})
				lx.token = ""
			}
			lx.tokens = append(lx.tokens, token{
				text: string(ch),
				line: line,
			})
			lx.nextTokenIsDirective = true
			goto again
		}
		lx.token += string(ch)
	}
}

func (lx lexer) custom(token string) (c customLexer, ok bool) {
	if lx.nextTokenIsDirective && lx.externalLexer != nil {
		c, ok = lx.externalLexer[token]
	}
	return
}

func balanceBraces(tokens []token, filename string) error {
	var depth int
	var line int
	for i := 0; i < len(tokens); i++ {
		t := &tokens[i]
		line = t.line
		if t.text == "}" && !t.quote {
			depth--
		} else if t.text == "{" && !t.quote {
			depth++
		}
		if depth < 0 {
			return NgxError{
				Reason:   "unexpected }",
				Filename: filename,
				Linenum:  t.line,
			}
		}
	}
	if depth > 0 {
		return NgxError{
			Reason:   "unexpected end of file, expecting }",
			Filename: filename,
			Linenum:  line,
		}
	}
	return nil
}

type wrapBufio struct {
	*bufio.Reader
}

func newIter(rd io.Reader) Iter {
	return &wrapBufio{Reader: bufio.NewReader(rd)}
}

func (w *wrapBufio) Next() (r rune, err error) {
	r, _, err = w.ReadRune()
	return
}

func lex(rd io.Reader, filename string) ([]token, error) {
	lx := newLexer(newIter(rd), defaultCustomLexers())
	if err := lx.lex(); err != nil {
		return nil, err
	}
	if err := balanceBraces(lx.tokens, filename); err != nil {
		return nil, err
	}
	return lx.tokens, nil
}

var _ customLexer = luaLexer{}

type luaLexer struct{}

func (luaLexer) directives() []string {
	return []string{
		"access_by_lua_block",
		"balancer_by_lua_block",
		"body_filter_by_lua_block",
		"content_by_lua_block",
		"header_filter_by_lua_block",
		"init_by_lua_block",
		"init_worker_by_lua_block",
		"log_by_lua_block",
		"rewrite_by_lua_block",
		"set_by_lua_block",
		"ssl_certificate_by_lua_block",
		"ssl_session_fetch_by_lua_block",
		"ssl_session_store_by_lua_block",
	}
}

func (luaLexer) lex(it IterLine, directive string) (tok []token, err error) {
	if directive == "set_by_lua_block" {
		var arg string
		for ch, line, err := it.Next(); err == nil; ch, line, err = it.Next() {
			if unicode.IsSpace(ch) {
				if arg != "" {
					tok = append(tok, token{
						text: arg,
						line: line,
					})
				} else {
					for unicode.IsSpace(ch) {
						ch, line, err = it.Next()
					}
				}
			}
			arg += string(ch)
		}
	}

	ch, line, err := consumeSpace(it)
	if err != nil {
		return nil, err
	}
	if ch != '{' {
		return nil, &NgxError{
			Reason:  "expected { to start Lua block",
			Linenum: line,
		}
	}
	var depth int
	var tk string
	depth++
	for ch, line, err = it.Next(); err == nil; ch, line, err = it.Next() {
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
		case '"', '\'':
			quote := ch
			tk += string(ch)
			ch, line, err = it.Next()
			for ch != quote {
				if ch == quote {
					tk += string(quote)
				} else {
					tk += string(ch)
				}
				ch, line, err = it.Next()
			}
		}
		if depth < 0 {
			return nil, &NgxError{
				Reason:  "unexpected }",
				Linenum: line,
			}
		}
		if depth == 0 {
			tok = append(tok, token{
				text: tk,
				line: line,
			})
			tok = append(tok, token{
				text: ";",
				line: line,
			})
			break
		}
		tk += string(ch)
	}
	return nil, nil
}

func consumeSpace(it IterLine) (ch rune, line int, err error) {
	for ch, line, err = it.Next(); unicode.IsSpace(ch); ch, line, err = it.Next() {
	}
	return
}

func (luaLexer) build(buf string, stmt *Stmt, padding, depth int) string {
	buf += stmt.Directive
	if stmt.Directive == "set_by_lua_block" {
		if len(stmt.Args) > 0 {
			buf += " " + stmt.Args[0]
		}
		if len(stmt.Args) > 1 {
			buf += " {"
			buf += stmt.Args[1]
			buf += "}"
		}
	} else {
		if len(stmt.Args) > 0 {
			buf += " {"
			buf += stmt.Args[0]
			buf += "}"
		}
	}
	return buf
}
