package templates

import (
	"net/http"
	"os"
	"path"
	"sync"

	"github.com/rakyll/statik/fs"
)

// FS embedded filesystem
var cached http.FileSystem
var do sync.Once

func get() http.FileSystem {
	do.Do(func() {
		cached = must(fs.New())
	})
	return cached
}

// PrefixFS implements http.FileServer by joining prefix before calling Open on
// embedded fs
type PrefixFS string

// Open joins f with name before opening the file
func (f PrefixFS) Open(name string) (file http.File, err error) {
	return get().Open(path.Join(string(f), name))
}

// PrefixFallbackFS tries prefix if it fails falls back to os.Open
type PrefixFallbackFS string

// Open joins f with name before opening the file if the file is missing this
// will use os.Open(name) instead.
func (f PrefixFallbackFS) Open(name string) (file http.File, err error) {
	if path.IsAbs(name) {
		file, err = get().Open(path.Join(string(f), name))
		if os.IsNotExist(err) {
			return os.Open(name)
		}
		return
	}
	return os.Open(name)
}

// IncludeFS fs for embedded include configurations
const IncludeFS PrefixFallbackFS = "/confs/includes"

func must(v http.FileSystem, err error) http.FileSystem {
	if err != nil {
		panic(err)
	}
	return v
}
