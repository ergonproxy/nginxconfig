package badgerfs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/util"
)

func TestDir(t *testing.T) {
	runAll(t,
		testMkdirAll,
		testMkdirAllNested,
		testMkdirAllIdempotent,
		testReadDir,
	)
}

func testMkdirAll(ts *testing.T, fs billy.Filesystem) {
	ts.Run("MKDirAll", func(t *testing.T) {
		err := fs.MkdirAll("empty", os.FileMode(0755))
		assert.Nil(t, err)

		fi, err := fs.Stat("empty")
		assert.Nil(t, err)
		assert.True(t, fi.IsDir())
	})
}

func testMkdirAllNested(ts *testing.T, fs billy.Filesystem) {
	ts.Run("MkdirAllNested", func(t *testing.T) {
		err := fs.MkdirAll("foo/bar/baz", os.FileMode(0755))
		assert.Nil(t, err)

		fi, err := fs.Stat("foo/bar/baz")
		assert.Nil(t, err)
		assert.True(t, fi.IsDir())

		fi, err = fs.Stat("foo/bar")
		assert.Nil(t, err)
		assert.True(t, fi.IsDir())

		fi, err = fs.Stat("foo")
		assert.Nil(t, err)
		assert.True(t, fi.IsDir())
	})
}

func testMkdirAllIdempotent(ts *testing.T, fs billy.Filesystem) {
	ts.Run("MkdirAllIdempotent", func(t *testing.T) {
		err := fs.MkdirAll("empty", 0755)
		assert.Nil(t, err)
		fi, err := fs.Stat("empty")
		assert.Nil(t, err)
		assert.True(t, fi.IsDir())

		// idempotent
		err = fs.MkdirAll("empty", 0755)
		assert.Nil(t, err)
		fi, err = fs.Stat("empty")
		assert.Nil(t, err)
		assert.True(t, fi.IsDir())
	})
}

func testReadDir(ts *testing.T, fs billy.Filesystem) {
	ts.Run("ReadDir", func(t *testing.T) {
		files := []string{"foo", "bar", "qux/baz", "qux/qux"}
		for _, name := range files {
			err := util.WriteFile(fs, name, nil, 0644)
			assert.Nil(t, err)
		}

		info, err := fs.ReadDir("/")
		assert.Nil(t, err)
		assert.Equal(t, 3, len(info))

		info, err = fs.ReadDir("/qux")
		assert.Nil(t, err)
		assert.Equal(t, 2, len(info))
	})
}
