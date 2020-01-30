package main

import (
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

// Stmt defines a nginx configuration directive.
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

func (t *tokenIter) next() *token {
	if t.idx < len(t.tokens) {
		t.idx++
		return &t.tokens[t.idx-1]
	}
	return nil
}

type fileCtx struct {
	name string
	ctx  []string
}

type parsingContext struct {
	File   string
	Status string
	Errors []error
	Parsed []*Stmt
	opts   *parseOpts
}

func (p *parsingContext) handleErr(err error) {
	if p.opts.errHandler != nil {
		p.opts.errHandler(err)
	}
	p.Status = "failed"
	p.Errors = append(p.Errors, err)
}

type payload struct {
	Status string
	Errors []error
	Config []*parsingContext
}

func parseInternal(fs http.FileSystem, parsing *parsingContext, tokens *tokenIter, ctx []string, consume bool) []*Stmt {
	parsed := []*Stmt{}
	for token := tokens.next(); token != nil; token = tokens.next() {
		var commentsInArgs []string
		if token.text == "}" && !token.quote {
			break
		}
		if consume {
			if token.text == "{" && !token.quote {
				parseInternal(fs, parsing, tokens, nil, true)
				continue
			}
		}
		directive := token.text
		var stmt *Stmt
		if parsing.opts.combine {
			stmt = &Stmt{
				Filename:  parsing.File,
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
		token = tokens.next()
		for inDirArgs(token) {
			if len(token.text) > 0 && token.text[0] == '#' && !token.quote {
				commentsInArgs = append(commentsInArgs, token.text[1:])
			} else {
				stmt.Args = append(stmt.Args, token.text)
			}
			token = tokens.next()
		}
		if parsing.opts.ignore != nil && parsing.opts.ignore(stmt.Directive) {
			if token.text == "{" && !token.quote {
				parseInternal(fs, parsing, tokens, nil, true)
			}
			continue
		}
		if stmt.Directive == "if" {
			prepareIfArgs(stmt)
		}
		err := analyze(parsing.File, stmt, token.text, ctx, parsing.opts.strict, parsing.opts.checkCtx, parsing.opts.checkArgs)
		if err != nil {
			parsing.handleErr(err)
		}
		if !parsing.opts.single && stmt.Directive == "include" {
			pattern := stmt.Args[0]
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(parsing.opts.configDir, pattern)
			}
			var filenames []string
			if strings.Contains(pattern, "*") {
				n, err := filepath.Glob(pattern)
				if err != nil {
					parsing.handleErr(
						&NgxError{
							Reason:   err.Error(),
							Linenum:  stmt.Line,
							Filename: parsing.File,
						},
					)
				} else {
					filenames = n
					sort.Strings(filenames)
				}
			} else {
				f, err := fs.Open(pattern)
				if err != nil {
					parsing.handleErr(
						&NgxError{
							Reason:   err.Error(),
							Linenum:  stmt.Line,
							Filename: parsing.File,
						},
					)
				} else {
					n, err := filepath.Glob(pattern)
					if err != nil {
						parsing.handleErr(
							&NgxError{
								Reason:   err.Error(),
								Linenum:  stmt.Line,
								Filename: parsing.File,
							},
						)
					} else {
						filenames = n
					}
					f.Close()
				}

			}
			for _, name := range filenames {
				parsing.opts.included[name] = len(parsing.opts.includes)
				parsing.opts.includes = append(parsing.opts.includes, fileCtx{
					name: name,
					ctx:  ctx,
				})
				stmt.Includes = append(stmt.Includes, parsing.opts.included[name])
			}
		}
		if token.text == "{" && !token.quote {
			stmt.Blocks = parseInternal(
				fs,
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
	return parsed
}

func inDirArgs(tok *token) bool {
	return !checkTerminal(tok.text) || tok.quote
}

func checkTerminal(txt string) bool {
	switch txt {
	case "{", ";", "}":
		return true
	default:
		return false
	}
}

type parseOpts struct {
	catchErr   bool
	ignore     func(string) bool
	single     bool
	strict     bool
	comments   bool
	combine    bool
	checkCtx   bool
	checkArgs  bool
	configDir  string
	includes   []fileCtx
	included   map[string]int
	errHandler func(error)
}

func defaultParseOpts() *parseOpts {
	return &parseOpts{
		catchErr:  true,
		ignore:    func(_ string) bool { return false },
		checkCtx:  true,
		checkArgs: true,
	}
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

func parse(filename string, fs http.FileSystem, opts *parseOpts) *payload {
	opts.configDir = filepath.Dir(filename)
	opts.includes = append(opts.includes, fileCtx{name: filename})
	opts.included = map[string]int{
		"filename": 0,
	}
	pld := &payload{
		Status: "ok",
	}
	opts.errHandler = func(err error) {
		pld.Status = "failed"
		pld.Errors = append(pld.Errors, err)
	}
	it := &includeIter{opts: opts}
	for f := it.next(); f != nil; f = it.next() {
		parsing, err := parseInclude(fs, opts, f)
		if err != nil {
			pld.Status = "failed"
			pld.Errors = append(pld.Errors, err)
		}
		pld.Config = append(pld.Config, parsing)
	}
	if opts.combine {
		return combineParsedConfig(opts, pld)
	}
	return pld
}

func parseInclude(fs http.FileSystem, opts *parseOpts, f *fileCtx) (*parsingContext, error) {
	parsing := &parsingContext{
		File:   f.name,
		Status: "ok",
		opts:   opts,
	}
	file, err := fs.Open(f.name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	tokens, err := lex(file, f.name)
	if err != nil {
		return nil, err
	}
	it := &tokenIter{tokens: tokens}
	parsing.Parsed = parseInternal(fs, parsing, it, f.ctx, false)
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
		File:   p.Config[0].File,
		Status: "ok",
		opts:   opts,
	}
	for _, c := range p.Config {
		combine.Errors = append(combine.Errors, c.Errors...)
		if c.Status == "failed" {
			combine.Status = "failed"
		}
	}
	combine.Parsed = performInclude(p, p.Config[0].Parsed)
	return &payload{
		Status: p.Status,
		Errors: p.Errors,
		Config: []*parsingContext{combine},
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
				for _, v := range performInclude(p, p.Config[idx].Parsed) {
					o = append(o, v)
				}
			}
		} else {
			o = append(o, stmt)
		}
	}
	return o
}
