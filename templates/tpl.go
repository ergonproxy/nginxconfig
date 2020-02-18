package templates

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ergongate/vince/buffers"
	"github.com/ergongate/vince/templates/octicons"
	"github.com/rakyll/statik/fs"
)

var htmlTpl *template.Template
var htmlOnce sync.Once

// time formats
const (
	iso8601Milli        = "2006-01-02T15:04:05.000Z"
	commonLogFormatTime = "02/Jan/2006:15:04:05 -0700"
)

// HTML returns template with all embedded templates loaded
func HTML() *template.Template {
	htmlOnce.Do(func() {
		htmlTpl = template.Must(loadHTML())
	})
	return htmlTpl
}

// IsVariableFunc returns true if variable is a template function
func IsVariableFunc(v string) bool {
	switch v {
	case "date_gmt", "date_local", "time_iso8601", "time_local":
		return true
	default:
		return false
	}
}

func loadHTML() (*template.Template, error) {
	files, err := fs.New()
	if err != nil {
		return nil, err
	}
	tpl := template.New("vince").Funcs(template.FuncMap{
		"octicon": octicons.Icon,
		"date_gmt": func() string {
			return time.Now().Format(http.TimeFormat)
		},
		"date_local": func() string {
			return time.Now().Format(time.RFC1123)
		},
		"time_iso8601": func() string {
			return time.Now().Format(iso8601Milli)
		},
		"time_local": func() string {
			return time.Now().Format(commonLogFormatTime)
		},
	})
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
