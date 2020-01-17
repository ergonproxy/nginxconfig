// package badgerfs provides a billy filesystem base on memory.
package badgerfs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/util"
)

const separator = "/"

// Memory a very convenient filesystem based on memory files
type Memory struct {
	s         *storage
	tempCount int
	root      string
}

func (m *Memory) Root() string {
	return m.root
}
func (m *Memory) Chroot(root string) (billy.Filesystem, error) {
	return &Memory{root: root, s: m.s}, nil
}

//New returns a new Memory filesystem.
func New(kv KV) billy.Filesystem {
	return &Memory{s: newStorage(kv), root: separator}
}

func (fs *Memory) Create(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

func (fs *Memory) Open(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDONLY, 0)
}

func (fs *Memory) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	f, has := fs.s.Get(filename)
	if !has {
		if !isCreate(flag) {
			return nil, os.ErrNotExist
		}
		return fs.s.New(filename, perm, flag)
	} else {
		if target, isLink := fs.resolveLink(filename, f); isLink {
			return fs.OpenFile(target, flag, perm)
		}
	}

	if f.FileMode.IsDir() {
		return nil, fmt.Errorf("cannot open directory: %s", filename)
	}
	return f.Duplicate(filename, perm, flag), nil
}

var errNotLink = errors.New("not a link")

func (fs *Memory) resolveLink(fullpath string, f *file) (target string, isLink bool) {
	if !isSymlink(f.FileMode) {
		return fullpath, false
	}

	target = string(f.Content.Bytes)
	if !isAbs(target) {
		target = fs.Join(path.Dir(fullpath), target)
	}

	return target, true
}

// On Windows OS, IsAbs validates if a path is valid based on if stars with a
// unit (eg.: `C:\`)  to assert that is absolute, but in this mem implementation
// any path starting by `separator` is also considered absolute.
func isAbs(f string) bool {
	return path.IsAbs(f) || strings.HasPrefix(f, separator)
}

func (fs *Memory) Stat(filename string) (os.FileInfo, error) {
	f, has := fs.s.Get(filename)
	if !has {
		return nil, os.ErrNotExist
	}

	fi, _ := f.Stat()

	var err error
	if target, isLink := fs.resolveLink(filename, f); isLink {
		fi, err = fs.Stat(target)
		if err != nil {
			return nil, err
		}
	}

	// the name of the file should always the name of the stated file, so we
	// overwrite the Stat returned from the storage with it, since the
	// filename may belong to a link.
	fi.(*fileInfo).name = filepath.Base(filename)
	return fi, nil
}

func (fs *Memory) Lstat(filename string) (os.FileInfo, error) {
	f, has := fs.s.Get(filename)
	if !has {
		return nil, os.ErrNotExist
	}

	return f.Stat()
}

func (fs *Memory) ReadDir(path string) ([]os.FileInfo, error) {
	if f, has := fs.s.Get(path); has {
		if target, isLink := fs.resolveLink(path, f); isLink {
			return fs.ReadDir(target)
		}
	}

	var entries []os.FileInfo
	for _, f := range fs.s.Children(path) {
		fi, _ := f.Stat()
		entries = append(entries, fi)
	}

	return entries, nil
}

func (fs *Memory) MkdirAll(path string, perm os.FileMode) error {
	_, err := fs.s.New(path, perm|os.ModeDir, 0)
	return err
}

func (fs *Memory) TempFile(dir, prefix string) (billy.File, error) {
	return util.TempFile(fs, dir, prefix)
}

func (fs *Memory) getTempFilename(dir, prefix string) string {
	fs.tempCount++
	filename := fmt.Sprintf("%s_%d_%d", prefix, fs.tempCount, time.Now().UnixNano())
	return fs.Join(dir, filename)
}

func (fs *Memory) Rename(from, to string) error {
	return fs.s.Rename(from, to)
}

func (fs *Memory) Remove(filename string) error {
	filename = clean(filename)
	fmt.Println(filename)
	return fs.s.Remove(filename)
}

func (fs *Memory) RemoveAll(filename string) error {
	filename = clean(filename)
	stat, err := fs.Stat(filename)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fs.s.Remove(filename)
	}
	return fs.s.RemoveAll(filename)
}

func (fs *Memory) Join(elem ...string) string {
	return path.Join(elem...)
}

func (fs *Memory) Symlink(target, link string) error {
	_, err := fs.Stat(link)
	if err == nil {
		return os.ErrExist
	}

	if !os.IsNotExist(err) {
		return err
	}

	return util.WriteFile(fs, link, []byte(target), 0777|os.ModeSymlink)
}

func (fs *Memory) Readlink(link string) (string, error) {
	f, has := fs.s.Get(link)
	if !has {
		return "", os.ErrNotExist
	}

	if !isSymlink(f.FileMode) {
		return "", &os.PathError{
			Op:   "readlink",
			Path: link,
			Err:  fmt.Errorf("not a symlink"),
		}
	}

	return string(f.Content.Bytes), nil
}

// Capabilities implements the Capable interface.
func (fs *Memory) Capabilities() billy.Capability {
	return billy.WriteCapability |
		billy.ReadCapability |
		billy.ReadAndWriteCapability |
		billy.SeekCapability |
		billy.TruncateCapability
}

type file struct {
	FileName string
	Content  *content
	position int64
	flag     int
	FileMode os.FileMode

	isClosed bool
	flush    func(*file) error
}

func (f *file) Name() string {
	return f.FileName
}

func (f *file) Read(b []byte) (int, error) {
	n, err := f.ReadAt(b, f.position)
	f.position += int64(n)

	if err == io.EOF && n != 0 {
		err = nil
	}

	return n, err
}

func (f *file) ReadAt(b []byte, off int64) (int, error) {
	if f.isClosed {
		return 0, os.ErrClosed
	}

	if !isReadAndWrite(f.flag) && !isReadOnly(f.flag) {
		return 0, errors.New("read not supported")
	}

	n, err := f.Content.ReadAt(b, off)

	return n, err
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	if f.isClosed {
		return 0, os.ErrClosed
	}

	switch whence {
	case io.SeekCurrent:
		f.position += offset
	case io.SeekStart:
		f.position = offset
	case io.SeekEnd:
		f.position = int64(f.Content.Len()) + offset
	}

	return f.position, nil
}

func (f *file) Write(p []byte) (n int, err error) {
	defer func() {
		if err == nil && n > 0 {
			if f.flush != nil {
				err = f.flush(f)
			}
		}
	}()
	if f.isClosed {
		err = os.ErrClosed
		return
	}

	if !isReadAndWrite(f.flag) && !isWriteOnly(f.flag) {
		err = errors.New("write not supported")
		return
	}

	n, err = f.Content.WriteAt(p, f.position)
	f.position += int64(n)
	return
}

func (f *file) Close() error {
	if f.isClosed {
		return os.ErrClosed
	}

	f.isClosed = true
	return nil
}

func (f *file) Truncate(size int64) error {
	if size < int64(len(f.Content.Bytes)) {
		f.Content.Bytes = f.Content.Bytes[:size]
	} else if more := int(size) - len(f.Content.Bytes); more > 0 {
		f.Content.Bytes = append(f.Content.Bytes, make([]byte, more)...)
	}

	return nil
}

func (f *file) Duplicate(filename string, mode os.FileMode, flag int) billy.File {
	new := &file{
		FileName: filename,
		Content:  f.Content,
		FileMode: mode,
		flag:     flag,
		flush:    f.flush,
	}

	if isAppend(flag) {
		new.position = int64(new.Content.Len())
	}

	if isTruncate(flag) {
		new.Content.Truncate()
	}

	return new
}

func (f *file) Stat() (os.FileInfo, error) {
	return &fileInfo{
		name: f.Name(),
		mode: f.FileMode,
		size: f.Content.Len(),
	}, nil
}

// Lock is a no-op in memfs.
func (f *file) Lock() error {
	return nil
}

// Unlock is a no-op in memfs.
func (f *file) Unlock() error {
	return nil
}

type fileInfo struct {
	name string
	size int
	mode os.FileMode
}

func (fi *fileInfo) Name() string {
	return fi.name
}

func (fi *fileInfo) Size() int64 {
	return int64(fi.size)
}

func (fi *fileInfo) Mode() os.FileMode {
	return fi.mode
}

func (*fileInfo) ModTime() time.Time {
	return time.Now()
}

func (fi *fileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

func (*fileInfo) Sys() interface{} {
	return nil
}

func (c *content) Truncate() {
	c.Bytes = make([]byte, 0)
}

func (c *content) Len() int {
	return len(c.Bytes)
}

func isCreate(flag int) bool {
	return flag&os.O_CREATE != 0
}

func isAppend(flag int) bool {
	return flag&os.O_APPEND != 0
}

func isTruncate(flag int) bool {
	return flag&os.O_TRUNC != 0
}

func isReadAndWrite(flag int) bool {
	return flag&os.O_RDWR != 0
}

func isReadOnly(flag int) bool {
	return flag == os.O_RDONLY
}

func isWriteOnly(flag int) bool {
	return flag&os.O_WRONLY != 0
}

func isSymlink(m os.FileMode) bool {
	return m&os.ModeSymlink != 0
}
