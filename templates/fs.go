package templates

import (
	"net/http"
	"os"
	"path"

	"github.com/rakyll/statik/fs"
)

type prefix struct {
	http.FileSystem
	prefix string
}

func (f *prefix) Open(name string) (file http.File, err error) {
	if path.IsAbs(name) {
		file, err = f.FileSystem.Open(path.Join(f.prefix, name))
		if os.IsNotExist(err) {
			return os.Open(name)
		}
		return
	}
	return os.Open(name)
}

func NewIncludeFS() (http.FileSystem, error) {
	return NewFS("/confs/includes")
}

func NewFS(pathPrefix string) (http.FileSystem, error) {
	f, err := fs.New()
	if err != nil {
		return nil, err
	}
	return &prefix{prefix: pathPrefix, FileSystem: f}, nil
}
