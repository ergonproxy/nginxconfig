package config

import "fmt"

type Position struct {
	Filename string // filename, if any
	Offset   int    // byte offset, starting at 0
	Line     int    // line number, starting at 1
	Column   int    // column number, starting at 1 (character count per line)
}

// IsValid reports whether the position is valid.
func (pos *Position) IsValid() bool { return pos.Line > 0 }

func (pos Position) String() string {
	s := pos.Filename
	if s == "" {
		s = "<input>"
	}
	if pos.IsValid() {
		s += fmt.Sprintf(":%d:%d", pos.Line, pos.Column)
	}
	return s
}

type Token struct {
	Text  string
	Start Position
	End   Position
}

type Directive struct {
	Parent     *Directive `json:"-"`
	Name       string
	Start, End Position
	Params     []Token
	Body       []*Directive
	Comments   []Token
}
