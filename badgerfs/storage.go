package badgerfs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	defaultDirectoryMode = 0755
	defaultCreateMode    = 0666
)

type storage struct {
	fs fsStore
}

func newStorage(kv KV) *storage {
	return &storage{
		fs: &KVFS{kv: kv},
	}
}

func (s *storage) Has(path string) bool {
	return s.fs.has(path)
}

func (s *storage) New(filename string, mode os.FileMode, flag int) (*file, error) {
	filename = clean(filename)
	if s.Has(filename) {
		if !s.MustGet(filename).FileMode.IsDir() {
			return nil, fmt.Errorf("file already exists %q", filename)
		}
		return nil, nil
	}

	dir, name := path.Split(filename)

	f := &file{
		FileName: name,
		Content:  &content{Name: name},
		FileMode: mode,
		flag:     flag,
		flush:    s.flushFile(filename),
	}
	if err := s.mkdirall(dir, mode); err != nil {
		return nil, err
	}
	if err := s.fs.set(filename, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (s *storage) flushFile(filename string) func(*file) error {
	return func(f *file) error {
		return s.fs.set(filename, f)
	}
}

func (s *storage) Children(dir string) []*file {
	dir = clean(dir)
	var l []*file
	ferr := s.fs.walk(dir, func(path string, err error, getFile func() (*file, error)) error {
		if err != nil {
			return err
		}
		if _, ok := child(dir, path); ok {
			f, err := getFile()
			if err != nil {
				return err
			}
			l = append(l, f)
		}
		return nil
	})
	if ferr != nil {
	}
	return l
}

func (s *storage) MustGet(path string) *file {
	f, ok := s.Get(path)
	if !ok {
		panic(fmt.Errorf("couldn't find %q", path))
	}
	return f
}

func (s *storage) Get(path string) (*file, bool) {
	path = clean(path)
	f, err := s.fs.get(path)
	if f != nil {
		f.flush = s.flushFile(path)
	}
	return f, err == nil
}

func (s *storage) Rename(from, to string) error {
	return s.move(from, to)
}

type pathFile struct {
	path string
	file *file
}

func collectFiles(fs fsStore, prefix string) ([]*pathFile, error) {
	var o []*pathFile
	ferr := fs.walk(prefix, func(path string, err error, getFile func() (*file, error)) error {
		if err != nil {
			return err
		}
		f, err := getFile()
		if err != nil {
			return err
		}
		o = append(o, &pathFile{path: path, file: f})
		return nil
	})
	if ferr != nil {
		return nil, ferr
	}
	return o, nil
}

func (s *storage) moveDir(from, to string, fromFile *file) error {
	if !s.fs.has(to) {
		err := s.mkdirall(to, fromFile.FileMode)
		if err != nil {
			return err
		}
	}
	files, err := collectFiles(s.fs, from)
	if err != nil {
		return err
	}
	for _, f := range files {
		newPath := path.Join(to, strings.TrimPrefix(from, f.path))
		if err := s.fs.set(newPath, f.file); err != nil {
			return err
		}
		if err := s.fs.remove(newPath); err != nil {
			return err
		}
	}
	return s.fs.remove(from)
}

func (s *storage) moveFile(from, to string, f *file) error {
	dir, name := path.Split(to)
	f.FileName = name
	if s.fs.has(to) {
		if err := s.fs.set(to, f); err != nil {
			return err
		}
		return s.fs.remove(from)
	}
	err := s.mkdirall(dir, defaultDirectoryMode)
	if err != nil {
		return err
	}
	if err := s.fs.set(to, f); err != nil {
		return err
	}
	return s.fs.remove(from)
}

func (s *storage) mkdirall(dirs string, mode os.FileMode) error {
	for _, dir := range mkdirAll(dirs, s.fs.has) {
		if err := s.mkdir(dir, mode); err != nil {
			return err
		}
	}
	return nil
}

func (s *storage) mkdir(dir string, mode os.FileMode) error {
	name := path.Base(dir)
	if !mode.IsDir() {
		// we make sure to mark this as directory
		mode |= os.ModeDir
	}
	f := &file{
		FileName: name,
		Content:  &content{Name: name},
		FileMode: mode | os.ModeDir,
	}
	return s.fs.set(dir, f)
}

func (s *storage) move(from, to string) error {
	f, err := s.fs.get(from)
	if err != nil {
		return err
	}
	if f.FileMode.IsDir() {
		return s.moveDir(from, to, f)
	}
	return s.moveFile(from, to, f)
}

// return true if dir contains other directories or files.
func (s *storage) dirHasChildren(dir string) bool {
	var ok bool
	s.fs.walk(dir, func(path string, err error, getFile func() (*file, error)) error {
		ok = true
		return io.EOF
	})
	return ok
}

func (s *storage) Remove(path string) error {
	path = clean(path)

	f, has := s.Get(path)
	if !has {
		return os.ErrNotExist
	}

	if f.FileMode.IsDir() && s.dirHasChildren(path) {
		return fmt.Errorf("dir: %s contains files", path)
	}
	return s.fs.remove(path)
}

func clean(f string) string {
	return path.Clean(filepath.ToSlash(f))
}

type content struct {
	Name  string
	Bytes []byte
}

func (c *content) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, &os.PathError{
			Op:   "writeat",
			Path: c.Name,
			Err:  errors.New("negative offset"),
		}
	}

	prev := len(c.Bytes)

	diff := int(off) - prev
	if diff > 0 {
		c.Bytes = append(c.Bytes, make([]byte, diff)...)
	}

	c.Bytes = append(c.Bytes[:off], p...)
	if len(c.Bytes) < prev {
		c.Bytes = c.Bytes[:prev]
	}

	return len(p), nil
}

func (c *content) ReadAt(b []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, &os.PathError{
			Op:   "readat",
			Path: c.Name,
			Err:  errors.New("negative offset"),
		}
	}

	size := int64(len(c.Bytes))
	if off >= size {
		return 0, io.EOF
	}

	l := int64(len(b))
	if off+l > size {
		l = size - off
	}

	btr := c.Bytes[off : off+l]
	if len(btr) < len(b) {
		err = io.EOF
	}
	n = copy(b, btr)

	return
}
