package badgerfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/assert"
	"gopkg.in/src-d/go-billy.v4"
)

func TestBasic(t *testing.T) {
	runAll(t,
		testCapabilities,
		testCreate,
		testCreateDepth,
		testCreateDepthAbsolute,
		testCreateOverides,
		testCreateAndClose,
		testOpen,
		testOpenNotExists,
		testOpenFiles,
		testOpenFilesNotTruncate,
		testOpenFileAppend,
	)
}

type noopLogger struct{}

func (noopLogger) Errorf(string, ...interface{})   {}
func (noopLogger) Warningf(string, ...interface{}) {}
func (noopLogger) Infof(string, ...interface{})    {}
func (noopLogger) Debugf(string, ...interface{})   {}

func runTest(t *testing.T, h FSTest) {
	dir, err := ioutil.TempDir("", "vince-fs-tes")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.RemoveAll(dir)
	}()
	opts := badger.DefaultOptions(dir).WithLogger(noopLogger{})
	db, err := badger.Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	fs := New(NewB(db))
	h(t, fs)
}

type FSTest func(*testing.T, billy.Filesystem)

func runAll(t *testing.T, hs ...FSTest) {
	for _, h := range hs {
		runTest(t, h)
	}
}

func testCapabilities(t *testing.T, fs billy.Filesystem) {
	t.Run("Capabilities", func(ts *testing.T) {
		_, ok := fs.(billy.Capable)
		assert.True(ts, ok)
		caps := billy.Capabilities(fs)
		assert.Equal(ts, billy.DefaultCapabilities&^billy.LockCapability, caps)
	})
}

func testCreate(ts *testing.T, fs billy.Filesystem) {
	ts.Run("Create", func(t *testing.T) {
		f, err := fs.Create("foo")
		assert.Nil(t, err)
		assert.Equal(t, f.Name(), "foo")
		assert.Nil(t, f.Close())
	})
}

func testCreateDepth(ts *testing.T, fs billy.Filesystem) {
	ts.Run("CreateDepth", func(t *testing.T) {
		f, err := fs.Create("bar/foo")
		assert.Nil(t, err)
		assert.Equal(t, f.Name(), fs.Join("bar", "foo"))
		assert.Nil(t, f.Close())
	})
}

func testCreateDepthAbsolute(ts *testing.T, fs billy.Filesystem) {
	ts.Run("CreateDepthAbsolute", func(t *testing.T) {
		f, err := fs.Create("/bar/foo")
		assert.Nil(t, err)
		assert.Equal(t, f.Name(), fs.Join("bar", "foo"))
		assert.Nil(t, f.Close())
	})
}

func testCreateOverides(ts *testing.T, fs billy.Filesystem) {
	ts.Run("CreateOverides", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			f, err := fs.Create("foo")
			assert.Nil(t, err)

			l, err := f.Write([]byte(fmt.Sprintf("foo%d", i)))
			assert.Nil(t, err)
			assert.Equal(t, 4, l)

			err = f.Close()
			assert.Nil(t, err)
		}

		f, err := fs.Open("foo")
		assert.Nil(t, err)

		wrote, err := ioutil.ReadAll(f)
		assert.Nil(t, err)
		assert.Equal(t, "foo2", string(wrote))
		assert.Nil(t, f.Close())
	})
}

func testCreateAndClose(ts *testing.T, fs billy.Filesystem) {
	ts.Run("CreateAndClose", func(t *testing.T) {
		f, err := fs.Create("foo")
		assert.Nil(t, err)

		_, err = f.Write([]byte("foo"))
		assert.Nil(t, err)
		assert.Nil(t, f.Close())

		f, err = fs.Open(f.Name())
		assert.Nil(t, err)

		wrote, err := ioutil.ReadAll(f)
		assert.Nil(t, err)
		assert.Equal(t, "foo", string(wrote))
		assert.Nil(t, f.Close())
	})
}

func testOpen(ts *testing.T, fs billy.Filesystem) {
	ts.Run("Open", func(t *testing.T) {
		f, err := fs.Create("foo")
		assert.Nil(t, err)
		assert.Equal(t, "foo", f.Name())
		assert.Nil(t, f.Close())

		f, err = fs.Open("foo")
		assert.Nil(t, err)
		assert.Equal(t, "foo", f.Name())
		assert.Nil(t, f.Close())
	})
}

func testOpenNotExists(ts *testing.T, fs billy.Filesystem) {
	ts.Run("OpenNotExists", func(t *testing.T) {
		f, err := fs.Open("not-exists")
		assert.NotNil(t, err)
		assert.Nil(t, f)
	})
}

func testOpenFiles(ts *testing.T, fs billy.Filesystem) {
	ts.Run("OpenFile", func(t *testing.T) {
		defaultMode := os.FileMode(0666)

		f, err := fs.OpenFile("foo1", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, defaultMode)
		assert.Nil(t, err)
		testWriteClose(t, f, "foo1")

		// Truncate if it exists
		f, err = fs.OpenFile("foo1", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, defaultMode)
		assert.Nil(t, err)
		assert.Equal(t, f.Name(), "foo1")
		testWriteClose(t, f, "foo1overwritten")

		// Read-only if it exists
		f, err = fs.OpenFile("foo1", os.O_RDONLY, defaultMode)
		assert.Nil(t, err)
		assert.Equal(t, f.Name(), "foo1")
		testReadClose(t, f, "foo1overwritten")

		// // Create when it does exist
		f, err = fs.OpenFile("foo1", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, defaultMode)
		assert.Nil(t, err)
		assert.Equal(t, f.Name(), "foo1")
		testWriteClose(t, f, "bar")

		f, err = fs.OpenFile("foo1", os.O_RDONLY, defaultMode)
		assert.Nil(t, err)
		testReadClose(t, f, "bar")
	})
}

func testWriteClose(t *testing.T, f billy.File, content string) {
	written, err := f.Write([]byte(content))
	assert.Nil(t, err)
	assert.Equal(t, len(content), written)
	assert.Nil(t, f.Close())
}

func testReadClose(t *testing.T, f billy.File, content string) {
	read, err := ioutil.ReadAll(f)
	assert.Nil(t, err)
	assert.Equal(t, content, string(read))
	assert.Nil(t, f.Close())
}

func testOpenFilesNotTruncate(ts *testing.T, fs billy.Filesystem) {
	ts.Run("OpenFileNotTruncate", func(t *testing.T) {
		defaultMode := os.FileMode(0666)

		// Create when it does not exist
		f, err := fs.OpenFile("foo1", os.O_CREATE|os.O_WRONLY, defaultMode)
		assert.Nil(t, err)
		assert.Equal(t, "foo1", f.Name())
		testWriteClose(t, f, "foo1")

		f, err = fs.OpenFile("foo1", os.O_RDONLY, defaultMode)
		assert.Nil(t, err)
		testReadClose(t, f, "foo1")

		// Create when it does exist
		f, err = fs.OpenFile("foo1", os.O_CREATE|os.O_WRONLY, defaultMode)
		assert.Nil(t, err)
		assert.Equal(t, "foo1", f.Name())
		testWriteClose(t, f, "bar")

		f, err = fs.OpenFile("foo1", os.O_RDONLY, defaultMode)
		assert.Nil(t, err)
		testReadClose(t, f, "bar1")
	})
}

func testOpenFileAppend(ts *testing.T, fs billy.Filesystem) {
	ts.Run("OpenFileAppend", func(t *testing.T) {
		defaultMode := os.FileMode(0666)

		f, err := fs.OpenFile("foo1", os.O_CREATE|os.O_WRONLY|os.O_APPEND, defaultMode)
		assert.Nil(t, err)
		assert.Equal(t, "foo1", f.Name())
		testWriteClose(t, f, "foo1")

		f, err = fs.OpenFile("foo1", os.O_WRONLY|os.O_APPEND, defaultMode)
		assert.Nil(t, err)
		assert.Equal(t, "foo1", f.Name())
		testWriteClose(t, f, "bar1")

		f, err = fs.OpenFile("foo1", os.O_RDONLY, defaultMode)
		assert.Nil(t, err)
		testReadClose(t, f, "foo1bar1")
	})
}
