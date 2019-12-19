package badgerfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/util"
)

func TestSymlink(t *testing.T) {
	runAll(t,
		testSymlink,
		testSymlinkCrossDirs,
	)
}

func testSymlink(ts *testing.T, fs billy.Filesystem) {
	ts.Run("Symlink", func(t *testing.T) {
		err := util.WriteFile(fs, "file", nil, 0644)
		assert.Nil(t, err)

		err = fs.Symlink("file", "link")
		assert.Nil(t, err)

		fi, err := fs.Stat("link")
		assert.Nil(t, err)
		assert.Equal(t, "link", fi.Name())
	})
}

func testSymlinkCrossDirs(ts *testing.T, fs billy.Filesystem) {
	ts.Run("SymlinkCrossDirs", func(t *testing.T) {
		err := util.WriteFile(fs, "foo/file", nil, 0644)
		assert.Nil(t, err)

		err = fs.Symlink("../foo/file", "bar/link")
		assert.Nil(t, err)

		fi, err := fs.Stat("bar/link")
		assert.Nil(t, err)
		assert.Equal(t, "link", fi.Name())
	})
}
