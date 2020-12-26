// Package directory can be used to translate whole directories.
package directory

//go:generate mockgen -source=directory.go -destination=./mocks/directory.go

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bounoable/dragoman"
	"github.com/bounoable/dragoman/text"
)

// Translator is an interface for dragoman.Translator.
type Translator interface {
	Translate(
		ctx context.Context,
		file io.Reader,
		sourceLang,
		targetLang string,
		r text.Ranger,
		opts ...dragoman.TranslateOption,
	) ([]byte, error)
}

// Dir is a translatable directory.
type Dir struct {
	path           string
	normalizedPath string
	rangers        map[string]text.Ranger
	ext            []string
}

// Option is a Directory option.
type Option func(*Dir)

// New initializes the specified dir to be translated. Use the Ranger() option
// to register text.Rangers for different file extensions. Only files with an
// extension for which a text.Ranger has been registered will be translated.
func New(dir string, opts ...Option) Dir {
	ps := string(os.PathSeparator)
	d := Dir{
		path:           dir,
		normalizedPath: strings.TrimSuffix(dir, ps) + ps,
		rangers:        make(map[string]text.Ranger),
	}

	for _, opt := range opts {
		opt(&d)
	}

	if len(d.rangers) > 0 {
		d.ext = make([]string, 0, len(d.rangers))
		for ext := range d.rangers {
			d.ext = append(d.ext, ext)
		}
	}

	return d
}

// Ranger returns an Option that specifies the text.Ranger r to be used for
// files with extension ext.
func Ranger(ext string, r text.Ranger) Option {
	return func(d *Dir) {
		d.rangers[ext] = r
	}
}

// Path returns the absolute path to the directory.
func (d Dir) Path() string {
	return d.path
}

// Files recursively walks the directory and returns a map of filepaths to
// strings.Readers. Filepaths are relative to d.Path() and have no leading path
// seperator. Only files with an extension that has a text.Ranger registered in
// d will be included in the map.
func (d Dir) Files(ctx context.Context) (map[string]*strings.Reader, error) {
	files := make(map[string]*strings.Reader)
	if err := filepath.Walk(d.path, func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		var found bool
		for _, ext := range d.ext {
			if strings.HasSuffix(path, ext) {
				found = true
				break
			}
		}
		if !found {
			return nil
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", path, err)
		}

		relPath := strings.TrimPrefix(path, d.normalizedPath)
		files[relPath] = strings.NewReader(string(b))

		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", d.path, err)
	}

	return files, nil
}

// Translate translates every file returned by d.Files(ctx) and returns a map
// of filepaths to the translation of those filepaths.
func (d Dir) Translate(
	ctx context.Context,
	t Translator,
	sourceLang,
	targetLang string,
	opts ...dragoman.TranslateOption,
) (map[string]string, error) {
	files, err := d.Files(ctx)
	if err != nil {
		return nil, err
	}

	res := make(map[string]string, len(files))
	for rp, f := range files {
		b, err := t.Translate(ctx, f, sourceLang, targetLang, d.ranger(filepath.Ext(rp)), opts...)
		if err != nil {
			return res, fmt.Errorf("translate file %s: %w", d.fullPath(rp), err)
		}
		res[rp] = string(b)
	}

	return res, nil
}

func (d Dir) ranger(ext string) text.Ranger {
	return d.rangers[ext] // guaranteed to exist
}

func (d Dir) fullPath(rp string) string {
	return d.normalizedPath + rp
}
