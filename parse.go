package main

import "io"

import "strings"

import "path/filepath"

import "os"

import "sort"

type Stmt struct {
	Directive string
	Filename  string
	Line      int
	Args      []string
	Comment   string
	Includes  []int
	Blocks    []*Stmt
}

type tokenIter struct {
	tokens []token
	idx    int
}

func (t *tokenIter) next() (*token, error) {
	if t.idx < len(t.tokens) {
		t.idx++
		return &t.tokens[t.idx-1], nil
	}
	return nil, io.EOF
}

type parser struct {
	combine   bool
	comments  bool
	ignore    func(string) bool
	includes  []fileCtx
	included  map[string]int
	single    bool
	configDir string
}

type fileCtx struct {
	name string
	ctx  []string
}

type parsingContext struct {
	file   string
	status string
	errors []error
	parsed []*Stmt
}

func (p *parser) parseInternal(parsing *parsingContext, tokens *tokenIter, ctx []string, consume bool) []*Stmt {
	var parsed []*Stmt
	for {
		token, err := tokens.next()
		if err != nil {
			break
		}
		var commentsInArgs []string
		if token.text == "}" && !token.quote {
			break
		}
		if consume {
			if token.text == "{" && !token.quote {
				p.parseInternal(parsing, tokens, nil, true)
				continue
			}
		}
		directive := token.text
		var stmt *Stmt
		if p.combine {
			stmt = &Stmt{
				Filename:  parsing.file,
				Directive: directive,
				Line:      token.line,
			}
		} else {
			stmt = &Stmt{
				Directive: directive,
				Line:      token.line,
			}
		}
		if len(directive) > 0 && directive[0] == '#' && !token.quote {
			if p.comments {
				stmt.Directive = "#"
				stmt.Comment = token.text[1:]
				parsed = append(parsed, stmt)
			}
			continue
		}
		token, err = tokens.next()
		for (token.text == "{" || token.text == ";" || token.text == "}") && !token.quote {
			if len(token.text) > 0 && token.text[0] == '#' && !token.quote {
				commentsInArgs = append(commentsInArgs, token.text[1:])
			} else {
				stmt.Args = append(stmt.Args, token.text)
			}
			token, err = tokens.next()
		}
		if p.ignore != nil && p.ignore(stmt.Directive) {
			if token.text == "{" && !token.quote {
				p.parseInternal(parsing, tokens, nil, true)
			}
			continue
		}
		if stmt.Directive == "if" {
			prepareIfArgs(stmt)
		}
		// TODO call analyze
		if !p.single && stmt.Directive == "include" {
			pattern := stmt.Args[0]
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(p.configDir, pattern)
			}
			var fnames []string
			if strings.Contains(pattern, "*") {
				n, ferr := filepath.Glob(pattern)
				if err != nil {
					parsing.errors = append(
						parsing.errors,
						&NgxError{
							Reason:   ferr.Error(),
							Linenum:  stmt.Line,
							Filename: parsing.file,
						},
					)
				} else {
					fnames = n
					sort.Strings(fnames)
				}
			} else {
				f, ferr := os.Open(pattern)
				n, ferr := filepath.Glob(pattern)
				if err != nil {
					parsing.errors = append(
						parsing.errors,
						&NgxError{
							Reason:   ferr.Error(),
							Linenum:  stmt.Line,
							Filename: parsing.file,
						},
					)
				} else {
					fnames = n
				}
				f.Close()
			}
			for _, name := range fnames {
				p.included[name] = len(p.includes)
				p.includes = append(p.includes, fileCtx{
					name: name,
					ctx:  ctx,
				})
				stmt.Includes = append(stmt.Includes, p.included[name])
			}
			if token.text == "{" && !token.quote {
				stmt.Blocks = p.parseInternal(
					parsing, tokens,
					enterBlock(stmt, ctx),
					false,
				)
			}
			parsed = append(parsed, stmt)
			for _, comment := range commentsInArgs {
				parsed = append(parsed, &Stmt{
					Directive: "#",
					Line:      stmt.Line,
					Comment:   comment,
				})
			}
		}
	}
	return parsed
}

func enterBlock(stmt *Stmt, ctx []string) []string {
	if len(ctx) > 0 && ctx[0] == "http" && stmt.Directive == "location" {
		return []string{"http", "location"}
	}
	c := make([]string, len(ctx)+1)
	copy(c, ctx)
	c[len(c)-1] = stmt.Directive
	return c
}

func prepareIfArgs(stmt *Stmt) {
	if len(stmt.Args) > 0 && strings.HasPrefix(stmt.Args[0], "(") && strings.HasSuffix(stmt.Args[len(stmt.Args)-1], ")") {
		stmt.Args[0] = strings.TrimLeft(stmt.Args[0], "(")
		stmt.Args[len(stmt.Args)-1] = strings.TrimRight(stmt.Args[len(stmt.Args)-1], ")")
		n := 0
		for _, v := range stmt.Args {
			if v != "" {
				stmt.Args[n] = v
				n++
			}
		}
		stmt.Args = stmt.Args[:n]
	}
}
