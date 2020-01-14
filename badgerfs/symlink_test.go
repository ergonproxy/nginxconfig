package badgerfs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/util"
)

var (
	customMode            os.FileMode = 0755
	expectedSymlinkTarget             = "/dir/file"
)

func TestSymlink(t *testing.T) {
	runAll(t,
		testSymlink,
		testSymlinkCrossDirs,
		testSymlinkNested,
		testSymlinkWithNonExistentdTarget,
		testSymlinkWithExistingLink,
		testOpenWithSymlinkToRelativePath,
		testOpenWithSymlinkToAbsolutePath,
		testReadlink,
		testReadlinkWithRelativePath,
		testReadlinkWithAbsolutePath,
		testReadlinkWithNonExistentTarget,
		testReadlinkWithNonExistentLink,
		testStatLink,
		testLstat,
		testLstatLink,
		testRenameWithSymlink,
		testRemoveWithSymlink,
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
func testSymlinkNested(ts *testing.T, fs billy.Filesystem) {
	ts.Run("SymlinkNested", func(t *testing.T) {
		err := util.WriteFile(fs, "file", []byte("hello world!"), 0644)
		assert.Nil(t, err)

		err = fs.Symlink("file", "linkA")
		assert.Nil(t, err)

		err = fs.Symlink("linkA", "linkB")
		assert.Nil(t, err)

		fi, err := fs.Stat("linkB")
		assert.Nil(t, err)
		assert.Equal(t, "linkB", fi.Name())
		assert.Equal(t, int64(12), fi.Size())
	})
}

func testSymlinkWithNonExistentdTarget(ts *testing.T, fs billy.Filesystem) {
	ts.Run("SymlinkWithNonExistentdTarget", func(t *testing.T) {
		err := fs.Symlink("file", "link")
		assert.Nil(t, err)

		_, err = fs.Stat("link")
		assert.True(t, os.IsNotExist(err))
	})
}
func testSymlinkWithExistingLink(ts *testing.T, fs billy.Filesystem) {
	ts.Run("SymlinkWithExistingLink", func(t *testing.T) {
		err := util.WriteFile(fs, "link", nil, 0644)
		assert.Nil(t, err)

		err = fs.Symlink("file", "link")
		assert.NotNil(t, err)
	})
}

func testOpenWithSymlinkToRelativePath(ts *testing.T, fs billy.Filesystem) {
	ts.Run("OpenWithSymlinkToRelativePath", func(t *testing.T) {
		err := util.WriteFile(fs, "dir/file", []byte("foo"), 0644)
		assert.Nil(t, err)

		err = fs.Symlink("file", "dir/link")
		assert.Nil(t, err)

		f, err := fs.Open("dir/link")
		assert.Nil(t, err)

		all, err := ioutil.ReadAll(f)
		assert.Nil(t, err)
		assert.Equal(t, "foo", string(all))
		assert.Nil(t, f.Close())
	})
}

func testOpenWithSymlinkToAbsolutePath(ts *testing.T, fs billy.Filesystem) {
	ts.Run("OpenWithSymlinkToAbsolutePath", func(t *testing.T) {
		err := util.WriteFile(fs, "dir/file", []byte("foo"), 0644)
		assert.Nil(t, err)

		err = fs.Symlink("/dir/file", "dir/link")
		assert.Nil(t, err)

		f, err := fs.Open("dir/link")
		assert.Nil(t, err)

		all, err := ioutil.ReadAll(f)
		assert.Nil(t, err)
		assert.Equal(t, "foo", string(all))
		assert.Nil(t, f.Close())
	})
}

func testReadlink(ts *testing.T, fs billy.Filesystem) {
	ts.Run("Readlink", func(t *testing.T) {
		err := util.WriteFile(fs, "file", nil, 0644)
		assert.Nil(t, err)

		_, err = fs.Readlink("file")
		assert.NotNil(t, err)
	})
}

func testReadlinkWithRelativePath(ts *testing.T, fs billy.Filesystem) {
	ts.Run("ReadlinkWithRelativePath", func(t *testing.T) {
		err := util.WriteFile(fs, "dir/file", nil, 0644)
		assert.Nil(t, err)

		err = fs.Symlink("file", "dir/link")
		assert.Nil(t, err)

		oldname, err := fs.Readlink("dir/link")
		assert.Nil(t, err)
		assert.Equal(t, "file", oldname)
	})
}

func testReadlinkWithAbsolutePath(ts *testing.T, fs billy.Filesystem) {
	ts.Run("ReadlinkWithAbsolutePath", func(t *testing.T) {
		err := util.WriteFile(fs, "dir/file", nil, 0644)
		assert.Nil(t, err)

		err = fs.Symlink("/dir/file", "dir/link")
		assert.Nil(t, err)

		oldname, err := fs.Readlink("dir/link")
		assert.Nil(t, err)
		assert.Equal(t, expectedSymlinkTarget, oldname)
	})
}

func testReadlinkWithNonExistentTarget(ts *testing.T, fs billy.Filesystem) {
	ts.Run("ReadlinkWithNonExistentTarget", func(t *testing.T) {
		err := fs.Symlink("file", "link")
		assert.Nil(t, err)

		oldname, err := fs.Readlink("link")
		assert.Nil(t, err)
		assert.Equal(t, "file", oldname)
	})
}
func testReadlinkWithNonExistentLink(ts *testing.T, fs billy.Filesystem) {
	ts.Run("ReadlinkWithNonExistentLink", func(t *testing.T) {
		_, err := fs.Readlink("link")
		assert.True(t, os.IsNotExist(err))
	})
}

func testStatLink(ts *testing.T, fs billy.Filesystem) {
	ts.Run("StatLink", func(t *testing.T) {
		util.WriteFile(fs, "foo/bar", []byte("foo"), customMode)
		fs.Symlink("bar", "foo/qux")

		fi, err := fs.Stat("foo/qux")
		assert.Nil(t, err)
		assert.Equal(t, "qux", fi.Name())
		assert.Equal(t, int64(3), fi.Size())
		assert.Equal(t, customMode, fi.Mode())
		assert.False(t, fi.ModTime().IsZero())
		assert.False(t, fi.IsDir())
	})
}

func testLstat(ts *testing.T, fs billy.Filesystem) {
	ts.Run("TestLstat", func(t *testing.T) {
		util.WriteFile(fs, "foo/bar", []byte("foo"), customMode)

		fi, err := fs.Lstat("foo/bar")
		assert.Nil(t, err)
		assert.Equal(t, "bar", fi.Name())
		assert.Equal(t, int64(3), fi.Size())
		assert.False(t, fi.Mode()&os.ModeSymlink != 0)
		assert.False(t, fi.ModTime().IsZero())
		assert.False(t, fi.IsDir())
	})
}

func testLstatLink(ts *testing.T, fs billy.Filesystem) {
	ts.Run("LstatLink", func(t *testing.T) {
		util.WriteFile(fs, "foo/bar", []byte("fosddddaaao"), customMode)
		fs.Symlink("bar", "foo/qux")

		fi, err := fs.Lstat("foo/qux")
		assert.Nil(t, err)
		assert.Equal(t, "qux", fi.Name())
		assert.True(t, fi.Mode()&os.ModeSymlink != 0)
		assert.False(t, fi.ModTime().IsZero())
		assert.False(t, fi.IsDir())
	})
}

func testRenameWithSymlink(ts *testing.T, fs billy.Filesystem) {
	ts.Run("RenameWithSymlink", func(t *testing.T) {
		err := fs.Symlink("file", "link")
		assert.Nil(t, err)

		err = fs.Rename("link", "newlink")
		assert.Nil(t, err)

		_, err = fs.Readlink("newlink")
		assert.Nil(t, err)
	})
}

func testRemoveWithSymlink(ts *testing.T, fs billy.Filesystem) {
	ts.Run("RemoveWithSymlink", func(t *testing.T) {
		err := util.WriteFile(fs, "file", []byte("foo"), 0644)
		assert.Nil(t, err)

		err = fs.Symlink("file", "link")
		assert.Nil(t, err)

		err = fs.Remove("link")
		assert.Nil(t, err)

		_, err = fs.Readlink("link")
		assert.True(t, os.IsNotExist(err))

		_, err = fs.Stat("link")
		assert.True(t, os.IsNotExist(err))

		_, err = fs.Stat("file")
		assert.Nil(t, err)
	})
}
