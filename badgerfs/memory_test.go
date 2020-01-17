package badgerfs

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/util"
)

func TestMemory(t *testing.T) {
	runAll(t, testRemoveAllRelative)
}

func testRemoveAllRelative(ts *testing.T, fs billy.Filesystem) {
	ts.Run("RemoveAllRelative", func(t *testing.T) {
		fnames := []string{
			"foo/1",
			"foo/2",
			"foo/bar/1",
			"foo/bar/2",
			"foo/bar/baz/1",
			"foo/bar/baz/qux/1",
			"foo/bar/baz/qux/2",
			"foo/bar/baz/qux/3",
		}

		for _, fname := range fnames {
			err := util.WriteFile(fs, fname, nil, 0644)
			assert.Nil(t, err)
		}

		assert.Nil(t, util.RemoveAll(fs, "foo/bar/.."))

		for _, fname := range fnames {
			_, err := fs.Stat(fname)
			comment := fmt.Sprintf("not removed: %s %s", fname, err)
			assert.True(t, os.IsNotExist(err), comment)
		}
	})
}
