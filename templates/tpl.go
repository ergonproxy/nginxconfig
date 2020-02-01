package templates

import (
	"html/template"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/ergongate/vince/buffers"
	"github.com/rakyll/statik/fs"
)

var htmlTpl *template.Template
var htmlOnce sync.Once

// HTML returns template with all embedded templates loaded
func HTML() *template.Template {
	htmlOnce.Do(func() {
		htmlTpl = template.Must(loadHTML())
	})
	return htmlTpl
}

func loadHTML() (*template.Template, error) {
	files, err := fs.New()
	if err != nil {
		return nil, err
	}
	tpl := template.New("vince")
	root := "/html"
	err = fs.Walk(files, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		name := strings.TrimPrefix(path, root+"/")
		t := tpl.New(name)
		f, err := files.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		_, err = t.Parse(string(b))
		return err
	})
	if err != nil {
		return nil, err
	}
	return tpl, nil
}

// Context this is passed to html template with values.
type Context struct {
	Title  string
	Meta   []Element
	Data   map[string]interface{}
	Footer []Element
}

// Element interface for a safe html element.
type Element interface {
	HTML() template.HTML
}

type attribute map[string]string

func (m attribute) html(s *strings.Builder) string {
	for k, v := range m {
		if s.Len() > 0 {
			s.WriteRune(' ')
		}
		s.WriteString(k)
		if v != "" {
			s.WriteString("=\"")
			s.WriteString(template.HTMLEscapeString(v))
			s.WriteString("\"")
		}
	}
	return s.String()
}

// Meta defines attributes for html meta tag
type Meta map[string]string

// HTML returns html meta tag
func (m Meta) HTML() template.HTML {
	s := buffers.GetString()
	defer func() {
		buffers.PutString(s)
	}()
	s.WriteString("<meta ")
	attribute(m).html(s)
	s.WriteRune('>')
	return template.HTML(s.String())
}
