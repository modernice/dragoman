package directory_test

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bounoable/dragoman"
	"github.com/bounoable/dragoman/directory"
	mock_directory "github.com/bounoable/dragoman/directory/mocks"
	"github.com/bounoable/dragoman/format/html"
	"github.com/bounoable/dragoman/format/json"
	"github.com/bounoable/dragoman/text"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDirectory_Path(t *testing.T) {
	dir := directory.New("/foo/bar")
	assert.Equal(t, "/foo/bar", dir.Path())
}

func TestDirectory_Absolute(t *testing.T) {
	dir := directory.New("/foo/bar")
	assert.Equal(t, "/foo/bar/baz", dir.Absolute("baz"))
	assert.Equal(t, "/foo/bar/baz/foobar.json", dir.Absolute("baz/foobar.json"))
	assert.Equal(t, "/foo/bar/baz/foobar.json", dir.Absolute("/baz/foobar.json"))
	assert.Equal(t, "/foo/bar/baz/foobar.json", dir.Absolute("./baz/foobar.json"))
}

func TestDirectory_Files(t *testing.T) {
	wd, _ := os.Getwd()
	p := filepath.Join(wd, "testdata")

	dir := directory.New(p)

	files, err := dir.Files(context.Background())

	assert.Nil(t, err)
	assertFiles(t, files)

	dir = directory.New(p, directory.Ranger(".json", json.Ranger()), directory.Ranger(".html", html.Ranger()))

	files, err = dir.Files(context.Background())

	assert.Nil(t, err)
	assertFiles(t, files, "foo/bar.json", "bar/baz.json", "baz/bar.html")
}

func TestDirectory_Files_context(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := directory.New("/")
	files, err := dir.Files(ctx)

	assert.Nil(t, files)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestDirectory_Files_ranger(t *testing.T) {
	dir := exampleDir(directory.Ranger(".json", json.Ranger()))

	files, err := dir.Files(context.Background())

	assert.Nil(t, err)
	assertFiles(t, files, "foo/bar.json", "bar/baz.json")
}

func TestDirectory_Translate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dir := exampleDir(directory.Ranger(".json", json.Ranger()))
	sourceLang := "en"
	targetLang := "de"

	trans := mock_directory.NewMockTranslator(ctrl)
	trans.EXPECT().
		Translate(gomock.Any(), gomock.Any(), sourceLang, targetLang, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, r io.Reader, _, _ string, _ text.Ranger, opts ...dragoman.TranslateOption) ([]byte, error) {
			assert.Len(t, opts, 1)
			b, err := ioutil.ReadAll(r)
			assert.Nil(t, err)
			return []byte("TRANSLATED: " + string(b)), nil
		}).
		AnyTimes()

	res, err := dir.Translate(
		context.Background(),
		trans,
		sourceLang,
		targetLang,
		dragoman.Parallel(2),
	)

	assert.Nil(t, err)

	files, err := dir.Files(context.Background())
	assert.Nil(t, err)

	for rp, f := range files {
		b, err := ioutil.ReadAll(f)
		assert.Nil(t, err)
		s := string(b)
		assert.Equal(t, "TRANSLATED: "+s, res[rp])
	}
}

func exampleDir(opts ...directory.Option) directory.Dir {
	wd, _ := os.Getwd()
	p := filepath.Join(wd, "testdata")
	return directory.New(p, opts...)
}

func assertFiles(t *testing.T, files map[string]*strings.Reader, want ...string) {
	assert.Len(t, files, len(want))
	for _, p := range want {
		s := readExampleFile(p)
		b, err := ioutil.ReadAll(files[p])
		assert.Nil(t, err)
		assert.Equal(t, s, string(b))
	}
}

func readExampleFile(p string) string {
	wd, _ := os.Getwd()
	p = filepath.Join(wd, "testdata", p)
	b, err := ioutil.ReadFile(p)
	if err != nil {
		panic(err)
	}
	return string(b)
}
