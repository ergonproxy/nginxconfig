package lex

import (
	"io"

	"github.com/ergongate/vince/config/nginx"
)

// File returns the top most directive.
func File(file string, src io.Reader) (*nginx.Directive, error) {
	s := new(Scanner).Init(src)
	s.Filename = file
	m := &nginx.Directive{
		Name: "main",
		Body: &nginx.List{
			Start: s.Pos(),
		},
	}
	d, err := lex(s, m, []string{"main"}, false, false)
	if err != nil {
		return nil, err
	}
	m.Body.Blocks = d
	m.Body.End = s.Pos()
	return m, nil
}

func lex(s *Scanner,
	parent *nginx.Directive,
	ctx []string,
	consume, checkCtx bool,
) ([]*nginx.Directive, error) {
	var parsed []*nginx.Directive
	for tok := s.Scan(); tok != EOF; tok = s.Scan() {
		if tok == RBrace {
			if parsed != nil {
				parsed[len(parsed)-1].End = s.Pos()
			}
			break
		}
		if consume {
			if tok == LBrace {
				lex(s, parent, nil, true, false)
			}
			continue
		}
		stmt := &nginx.Directive{
			Parent: parent,
			Name:   s.TokenText(),
			Start:  s.Position,
		}
		tok = s.Scan()
		for !isSpecialToken(tok) {
			stmt.Params = append(stmt.Params, nginx.Token{
				Text:  s.TokenText(),
				Start: s.Position,
				End:   s.Pos(),
			})
			tok = s.Scan()
		}
		switch tok {
		case LBrace:
			stmt.Body = &nginx.List{
				Start: s.Position,
			}
			tkn, err := lex(s, stmt, enterBlockContext(stmt, ctx), false, false)
			if err != nil {
				return nil, err
			}
			stmt.Body.Blocks = tkn
			stmt.Body.End = s.Pos()
		case SColon:
			// end of a simple block
			stmt.End = s.Pos()
		}
		parsed = append(parsed, stmt)
	}
	return parsed, nil
}

func isSpecialToken(tok rune) bool {
	switch tok {
	case LBrace, RBrace, SColon:
		return true
	default:
		return false
	}
}
