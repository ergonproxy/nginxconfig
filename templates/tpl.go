package templates

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"strings"

	"github.com/rakyll/statik/fs"
)

func HTML() (*template.Template, error) {
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
		fmt.Println(t.Name())
		_, err = t.Parse(string(b))
		return err
	})
	if err != nil {
		return nil, err
	}
	return tpl, nil
}
