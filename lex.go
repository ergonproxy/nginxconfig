package main

import (
	"bufio"
	"fmt"
	"io"
	"unicode"
)

type NgxError struct {
	Reason   string
	Linenum  int
	Filename string
}

func newError(reason string, line int, fname string) NgxError {
	return NgxError{Reason: reason, Linenum: line, Filename: fname}
}

func (n NgxError) Error() string {
	return fmt.Sprintf("%s:%d %s", n.Filename, n.Linenum, n.Reason)
}

type NgxParserDirectiveUnknownError struct {
	NgxError
}

type NgxParserDirectiveContextError struct {
	NgxError
}

type NgxParserDirectiveArgumentsError struct {
	NgxError
}

type Iter interface {
	Next() (rune, error)
}

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
	return map[string]customLexer{}
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
					tkns, err := clx.lex(lx.iter, lx.token)
					if err != nil {
						return err
					}
					lx.tokens = append(lx.tokens, tkns...)
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
				tkns, err := clx.lex(lx.iter, lx.token)
				if err != nil {
					return err
				}
				lx.tokens = append(lx.tokens, tkns...)
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
