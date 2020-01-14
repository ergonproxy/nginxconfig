package badgerfs

import (
	"bytes"
	"encoding/gob"
	"path"
	"strings"
)

type KV interface {
	Set(k string, v []byte) error
	Has(key string) bool
	Get(string) ([]byte, error)
	// Walk iterates over keys with prefix and calling fn on each key, if fn
	// returns false then this should exit(stop iterating further).
	Walk(prefix string, fn WalkFn) error
	Remove(string) error
}

type WalkFn func(key string, err error, value func() ([]byte, error)) error

type PrefixKV struct {
	kv     KV
	prefix string
}

func (p *PrefixKV) Set(key string, value []byte) error {
	return p.kv.Set(p.prefix+key, value)
}

func (p *PrefixKV) Get(key string) ([]byte, error) {
	return p.kv.Get(p.prefix + key)
}

func (p *PrefixKV) Has(key string) bool {
	return p.kv.Has(p.prefix + key)
}

type fsStore interface {
	get(string) (*file, error)
	walk(prefix string, h walkFileFn) error
	set(string, *file) error
	remove(string) error
	has(string) bool
}

var _ fsStore = (*KVFS)(nil)

type walkFileFn func(path string, err error, getFile func() (*file, error)) error

func marshalFile(f *file) ([]byte, error) {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(f)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func unMarshalFile(b []byte) (*file, error) {
	var f file
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&f)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

type KVFS struct {
	kv KV
}

func (fs *KVFS) get(path string) (*file, error) {
	b, err := fs.kv.Get(path)
	if err != nil {
		return nil, err
	}
	return unMarshalFile(b)
}

func (fs *KVFS) set(path string, f *file) error {
	b, err := marshalFile(f)
	if err != nil {
		return err
	}
	return fs.kv.Set(path, b)
}

func (fs *KVFS) remove(path string) error {
	return fs.kv.Remove(path)
}

func (fs *KVFS) has(path string) bool {
	return fs.kv.Has(path)
}

func (fs *KVFS) walk(prefix string, h walkFileFn) error {
	return fs.kv.Walk(prefix, func(key string, err error, value func() ([]byte, error)) error {
		return h(key, err, func() (*file, error) {
			b, err := value()
			if err != nil {
				return nil, err
			}
			return unMarshalFile(b)
		})
	})
}

// base /a/b/c
// target /a/b/c/d/e
// will return (/a/b/c/d, true)
func child(parent string, target string) (string, bool) {
	parent = clean(parent)
	target = clean(target)
	if len(parent) == len(target) || len(target) < len(parent) {
		return "", false
	}
	if !strings.HasPrefix(target, parent) {
		return "", false
	}
	target = strings.TrimPrefix(target, parent)
	if strings.HasPrefix(target, separator) {
		target = target[1:]
	}
	p := strings.Split(target, separator)
	if len(p) > 0 {
		target = p[0]
	}
	return path.Join(parent, target), true
}

func mkdirAll(dir string, has func(string) bool) []string {
	dir = clean(dir)
	p := strings.Split(dir, "/")
	var o []string
	for i := len(p); i > 0; i-- {
		n := strings.Join(p[0:i], "/")
		if n == "" {
			break
		}
		if !has(n) {
			o = append(o, n)
			continue
		}
		break
	}
	return o
}
