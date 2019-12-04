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

type ErrorList struct {
	Section   string
	Directive string
	Errors    []error
}

func (e ErrorList) Error() string {
	return ""
}

func (e *ErrorList) Add(n error) {
	e.Errors = append(e.Errors, n)
}

func (e *ErrorList) HasErrors() bool {
	return len(e.Errors) > 0
}

func NewError(section, directive string) *ErrorList {
	return &ErrorList{
		Section:   section,
		Directive: directive,
	}
}

func ErrorAt(err string, pos *Position) error {
	return fmt.Errorf("%v: %s", pos, err)
}

func (d *Token) Error(err string) error {
	return ErrorAt(err, &d.Start)
}

func (d *Directive) Error(err string) error {
	return ErrorAt(err, &d.Start)
}

// ShouldHaveParams returns an error if the directive doesn't have exactly n
// params.
func (d *Directive) ShouldHaveParams(n int) error {
	if len(d.Params) != n {
		return d.Error(fmt.Sprintf("wrong number of params expected %d got %d", n, len(d.Params)))
	}
	return nil
}

func (d *Directive) ShouldHaveName(name string) error {
	if d.Name != name {
		return d.Error("expected " + name)
	}
	return nil
}

func (d *Directive) ShouldHaveParent(name ...string) error {
	if len(name) > 0 {
		var ok bool
		if d.Parent != nil {
			for _, v := range name {
				if v == d.Parent.Name {
					ok = true
					break
				}
			}
		}
		if !ok {
			return d.Error("directive is defined in a wrong context")
		}
	}
	return nil
}

func (d *Directive) BasicCheck(name string, params int, parents ...string) error {
	err := d.ShouldHaveName(name)
	if err != nil {
		return err
	}
	err = d.ShouldHaveParams(params)
	if err != nil {
		return err
	}
	return d.ShouldHaveParent(parents...)
}
