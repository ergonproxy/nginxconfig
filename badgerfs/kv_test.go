package badgerfs

import (
	"fmt"
	"path"
	"reflect"
	"testing"
)

func gen(src []string, n, depth int) []string {
	if depth <= 0 {
		return src
	}
	var o []string
	for _, base := range src {
		for i := 0; i < n; i++ {
			o = append(o, path.Join(base, fmt.Sprintf("c%d", i)))
		}
	}
	return gen(o, n, depth-1)
}

func TestChildren(t *testing.T) {
	sample := []struct {
		parent, target string
		expect         string
		ok             bool
	}{
		{"/a/b/c", "/a/b", "", false},
		{"/a/b/c", "/a/b/c", "", false},
		{"/a/b/c", "/e/f", "", false},
		{"/a/b/c", "/a/b/c/d", "/a/b/c/d", true},
		{"/a/b/c/", "/a/b/c/d/", "/a/b/c/d", true},
		{"/a/b/c", "/a/b/c/d/e/f", "/a/b/c/d", true},
	}
	for i, s := range sample {
		got, ok := child(s.parent, s.target)
		if ok != s.ok {
			t.Errorf("%d: expected %v got %v", i, s.ok, ok)
		}
		if got != s.expect {
			t.Errorf("%d: expected %v got %v", i, s.expect, got)
		}
	}
}

// func TestSplit(t *testing.T) {
// 	s := []string{
// 		"", "/a", "a/",
// 	}
// 	for _, v := range s {
// 		p := strings.Split(v, "/")
// 		t.Errorf("%#v", p)
// 	}
// }
func TestMKdirall(t *testing.T) {
	v := mkdirAll("/a/b/c/d", func(_ string) bool {
		return false
	})
	expect := []string{"/a/b/c/d", "/a/b/c", "/a/b", "/a"}
	if !reflect.DeepEqual(v, expect) {
		t.Errorf("expected %#v got %#v", expect, v)
	}

	v = mkdirAll("/a/b/c/d", func(s string) bool {
		return s == "/a/b/c"
	})
	expect = []string{"/a/b/c/d"}
	if !reflect.DeepEqual(v, expect) {
		t.Errorf("expected %#v got %#v", expect, v)
	}
}
