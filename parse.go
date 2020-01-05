package main

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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

type fileCtx struct {
	name string
	ctx  []string
}

type parsingContext struct {
	file   string
	status string
	errors []error
	parsed []*Stmt
	opts   *parseOpts
}

type payload struct {
	status string
	errors []error
	config []*parsingContext
}

func parseInternal(parsing *parsingContext, tokens *tokenIter, ctx []string, consume bool) []*Stmt {
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
				parseInternal(parsing, tokens, nil, true)
				continue
			}
		}
		directive := token.text
		var stmt *Stmt
		if parsing.opts.combine {
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
			if parsing.opts.comments {
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
		if parsing.opts.ignore != nil && parsing.opts.ignore(stmt.Directive) {
			if token.text == "{" && !token.quote {
				parseInternal(parsing, tokens, nil, true)
			}
			continue
		}
		if stmt.Directive == "if" {
			prepareIfArgs(stmt)
		}
		// TODO call analyze
		if !parsing.opts.single && stmt.Directive == "include" {
			pattern := stmt.Args[0]
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(parsing.opts.configDir, pattern)
			}
			var fnames []string
			if strings.Contains(pattern, "*") {
				n, ferr := filepath.Glob(pattern)
				if err != nil {
					parsing.status = "failed"
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
					parsing.status = "failed"
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
				parsing.opts.included[name] = len(parsing.opts.includes)
				parsing.opts.includes = append(parsing.opts.includes, fileCtx{
					name: name,
					ctx:  ctx,
				})
				stmt.Includes = append(stmt.Includes, parsing.opts.included[name])
			}
			if token.text == "{" && !token.quote {
				stmt.Blocks = parseInternal(
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

type parseOpts struct {
	checkErr  bool
	ignore    func(string) bool
	single    bool
	comments  bool
	combine   bool
	checkCtx  bool
	configDir string
	includes  []fileCtx
	included  map[string]int
}

type includeIter struct {
	opts *parseOpts
	idx  int
}

func (i *includeIter) next() *fileCtx {
	if i.idx < len(i.opts.includes) {
		i.idx++
		return &i.opts.includes[i.idx-1]
	}
	return nil
}

func parse(filename string, opts *parseOpts) *payload {
	opts.configDir = filepath.Dir(filename)
	opts.includes = append(opts.includes, fileCtx{name: filename})
	opts.included = map[string]int{
		"filename": 0,
	}
	pld := &payload{
		status: "ok",
	}
	it := &includeIter{opts: opts}
	for f := it.next(); f != nil; f = it.next() {
		parsing, err := parseInclude(opts, f)
		if err != nil {
			pld.status = "failed"
			pld.errors = append(pld.errors, err)
		}
		pld.config = append(pld.config, parsing)
	}
	return pld
}

func parseInclude(opts *parseOpts, f *fileCtx) (*parsingContext, error) {
	parsing := &parsingContext{
		file:   f.name,
		status: "ok",
		opts:   opts,
	}
	fs, err := os.Open(f.name)
	if err != nil {
		return nil, err
	}
	defer fs.Close()
	tokens, err := lex(fs, f.name)
	if err != nil {
		return nil, err
	}
	it := &tokenIter{tokens: tokens}
	parsing.parsed = parseInternal(parsing, it, f.ctx, false)
	return parsing, nil
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

func combineParsedConfig(opts *parseOpts, p *payload) *payload {
	combine := &parsingContext{
		file:   p.config[0].file,
		status: "ok",
		opts:   opts,
	}
	for _, c := range p.config {
		combine.errors = append(combine.errors, c.errors...)
		if c.status == "failed" {
			combine.status = "failed"
		}
	}
	combine.parsed = performInclude(p, p.config[0].parsed)
	return &payload{
		status: p.status,
		errors: p.errors,
		config: []*parsingContext{combine},
	}
}

func performInclude(p *payload, block []*Stmt) []*Stmt {
	var o []*Stmt
	for _, stmt := range block {
		if stmt.Blocks != nil {
			stmt.Blocks = performInclude(p, stmt.Blocks)
		}
		if stmt.Includes != nil {
			for _, idx := range stmt.Includes {
				for _, v := range performInclude(p, p.config[idx].parsed) {
					o = append(o, v)
				}
			}
		} else {
			o = append(o, stmt)
		}
	}
	return o
}
