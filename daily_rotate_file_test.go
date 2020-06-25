package dailyrotate

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testWrite(t *testing.T, f *File, pathExp string) {
	var writtenAtPos int64

	n, err := io.WriteString(f, "hello\n")
	assert.NoError(t, err)
	assert.Equal(t, 0, int(f.lastWritePos))
	assert.Equal(t, n, 6)
	assert.Equal(t, pathExp, f.path)
	_, writtenAtPos, n, err = f.Write2([]byte("bar\n"), false)
	assert.NoError(t, err)
	assert.Equal(t, writtenAtPos, f.lastWritePos)
	assert.Equal(t, 6, int(writtenAtPos))
	assert.Equal(t, n, 4)
	err = f.Close()
	assert.NoError(t, err)

	d, err := ioutil.ReadFile(pathExp)
	assert.NoError(t, err)
	assert.Equal(t, string(d), "hello\nbar\n")

	path, off, n, err := f.Write2([]byte("and more\n"), true)
	assert.NoError(t, err)

	assert.Equal(t, len(d), int(off))

	assert.Equal(t, 9, n)
	assert.Equal(t, pathExp, path)
}

func TestBasic(t *testing.T) {
	os.RemoveAll("test_dir")
	defer os.RemoveAll("test_dir")

	pathFormat := filepath.Join("test_dir", "second", "2006-01-02.txt")
	pathExp := time.Now().UTC().Format(pathFormat)

	onOpenCalled := false
	onOpen := func(f *os.File, new bool) bool {
		onOpenCalled = true
		return true
	}
	onCloseCalled := false
	onClose := func(path string, didRotate bool) {
		onCloseCalled = true
	}
	f, err := NewFile(pathFormat, onOpen, onClose)
	assert.NoError(t, err)
	assert.NotNil(t, f)
	testWrite(t, f, pathExp)

	err = f.Close()
	assert.NoError(t, err)

	assert.True(t, onOpenCalled)
	assert.True(t, onCloseCalled)
	d, err := ioutil.ReadFile(pathExp)
	assert.NoError(t, err)
	assert.Equal(t, string(d), "hello\nbar\nand more\n")
}

func TestBasic_Location(t *testing.T) {
	os.RemoveAll("test_dir")
	defer os.RemoveAll("test_dir")

	loc := time.FixedZone("UTC-8", -8*60*60)
	pathFormat := filepath.Join("test_dir", "third", "2006-01-02.txt")
	pathExp := time.Now().In(loc).Format(pathFormat)
	f, err := NewFile(pathFormat, nil, nil)
	assert.NoError(t, err)
	f.Location = loc
	assert.Equal(t, loc, f.Location)

	n, err := io.WriteString(f, "hello\n")
	assert.NoError(t, err)
	assert.Equal(t, 0, int(f.lastWritePos))
	assert.Equal(t, n, 6)
	assert.Equal(t, pathExp, f.path)
}

func TestPathGenerator(t *testing.T) {
	os.RemoveAll("test_dir")
	defer os.RemoveAll("test_dir")

	pathFormat := filepath.Join("test_dir", "second", "2006-01-02.txt")
	pathExp := time.Now().UTC().Format(pathFormat)

	nCalled := 0
	pathGenerator := func(t time.Time, i uint) string {
		nCalled++
		return t.Format(pathFormat)
	}
	f, err := NewFileWithPathGenerator(pathGenerator, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, f)
	testWrite(t, f, pathExp)
	err = f.Close()
	assert.NoError(t, err)
	assert.True(t, nCalled > 0)
}

func TestPathGeneratorOnOpen(t *testing.T) {
	os.RemoveAll("test_dir")
	defer os.RemoveAll("test_dir")

	err := os.MkdirAll("test_dir", 0755)
	assert.NoError(t, err)
	testFile := filepath.Join("test_dir", "test.txt")
	tf, err := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE, 0644)
	assert.NoError(t, err)
	tf.Write([]byte("test string"))
	tf.Close()

	nOnOpenCalled := uint(0)
	onOpen := func(f *os.File, new bool) bool {
		if !new {
			b := make([]byte, 20)
			nf, err := os.Open(f.Name())
			assert.NoError(t, err)
			r, err := nf.Read(b)
			assert.NoError(t, err)
			assert.True(t, r == len("test string"))
			assert.True(t, string(b[:r]) == "test string")
			nf.Close()
		}
		return true
	}
	pathGeneratorIter := uint(0)
	pathGenerator := func(t time.Time, i uint) string {
		pathGeneratorIter = i
		return testFile
	}
	f, err := NewFileWithPathGenerator(pathGenerator, onOpen, nil)
	assert.NoError(t, err)
	assert.NotNil(t, f)
	err = f.Close()
	assert.NoError(t, err)
	assert.True(t, pathGeneratorIter == nOnOpenCalled)
}
