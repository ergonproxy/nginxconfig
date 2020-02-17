package octicons

import (
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"github.com/ergongate/vince/buffers"
	"github.com/shurcooL/octicon"
	"golang.org/x/net/html"
)

// Icon a template function for rendering github octicons
func Icon(name string, attrs ...string) (template.HTML, error) {
	h := octicon.Icon(name)
	if h != nil {
		if len(attrs) > 0 {
			for _, attr := range attrs {
				p := strings.Split(attr, "=")
				switch p[0] {
				case "size":
					i, err := strconv.Atoi(p[1])
					if err != nil {
						return "", err
					}
					octicon.SetSize(h, i)
				case "vertical-align":
					h.Attr[3].Val = strings.Replace(h.Attr[3].Val,
						"vertical-align: top", fmt.Sprintf("vertical-align: %s", p[1]))
				}
			}
		}
		return render(h)
	}
	return "", errors.New("octicons: " + name + "doesn't exist")
}

func render(h *html.Node) (template.HTML, error) {
	buf := buffers.GetBytes()
	defer buffers.PutBytes(buf)
	err := html.Render(buf, h)
	if err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
