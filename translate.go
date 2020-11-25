package translator

//go:generate mockgen -source=translate.go -destination=./mocks/translate.go

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"sync"

	"github.com/bounoable/translator/text"
	"github.com/bounoable/translator/text/preserve"
)

var (
	defaultTranslateConfig = translateConfig{parallel: 1}
)

// New returns a structured-text translator.
func New(service Service) *Translator {
	return &Translator{
		service: service,
	}
}

// Translator is a structured-text translator.
type Translator struct {
	service Service
}

// Service is a translation service (e.g. Google Translate / DeepL).
type Service interface {
	Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error)
}

// Translate the contents of input from sourceLang to targetLang.
func (t *Translator) Translate(
	ctx context.Context,
	input io.Reader,
	sourceLang, targetLang string,
	ranger text.Ranger,
	opts ...TranslateOption,
) ([]byte, error) {
	cfg := defaultTranslateConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	var rangerInput bytes.Buffer
	tr := io.TeeReader(input, &rangerInput)
	inputBytes, err := ioutil.ReadAll(tr)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	inputText := string(inputBytes)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ranges, rangeErrs := ranger.Ranges(ctx, &rangerInput)
	translatedRanges, translateRangeErrs := t.translateRanges(ctx, cfg, ranges, inputText, sourceLang, targetLang)

	var translations []translatedRange

L:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err, ok := <-rangeErrs:
			if !ok {
				rangeErrs = nil
				break
			}
			return nil, fmt.Errorf("get ranges: %w", err)
		case err, ok := <-translateRangeErrs:
			if !ok {
				translateRangeErrs = nil
				break
			}
			return nil, err
		case tr, open := <-translatedRanges:
			if !open {
				break L
			}
			translations = append(translations, tr)
		}
	}

	replacements := make([]text.Replacement, len(translations))
	for i, trans := range translations {
		replacements[i] = text.Replacement{
			Range: trans.r,
			Text:  trans.text,
		}
	}

	result, err := text.ReplaceMany(inputText, replacements...)
	if err != nil {
		return []byte(result), fmt.Errorf("replace texts: %w", err)
	}

	return []byte(result), nil
}

// A TranslateOption configures the translation behavior.
type TranslateOption func(*translateConfig)

// Preserve (prevent translation of) strings that match the given expr.
//
// A typical use case are placeholder variables. Example:
//   r, err := t.Translate(
//     context.TODO(),
//	   "Hello, {firstName}!",
//	   "EN", "DE",
//     translator.Preserve(regexp.MustCompile(`{[a-zA-Z0-9]+?}`)),
//   )
//   // r: "Hallo, {firstName}!"
func Preserve(expr *regexp.Regexp) TranslateOption {
	return func(cfg *translateConfig) {
		cfg.preserve = expr
	}
}

// Parallel sets the maximum number of parallel translation requests.
func Parallel(n int) TranslateOption {
	return func(cfg *translateConfig) {
		cfg.parallel = n
	}
}

type translateConfig struct {
	preserve *regexp.Regexp
	parallel int
}

func (t *Translator) translateRanges(
	ctx context.Context,
	cfg translateConfig,
	ranges <-chan text.Range,
	input, sourceLang, targetLang string,
) (<-chan translatedRange, <-chan *translateRangeError) {
	translated := make(chan translatedRange, len(ranges))
	errs := make(chan *translateRangeError, len(ranges))

	workers := cfg.parallel
	if workers < 0 {
		workers = 0
	}

	var wg sync.WaitGroup
	wg.Add(workers)

	go func() {
		defer close(translated)
		defer close(errs)
		wg.Wait()
	}()

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case r, ok := <-ranges:
					if !ok {
						return
					}

					t, err := t.translateRange(ctx, cfg, r, input, sourceLang, targetLang)
					if err != nil {
						errs <- &translateRangeError{
							r:   r,
							err: err,
						}
						break
					}

					translated <- translatedRange{
						r:    r,
						text: t,
					}
				}
			}
		}()
	}

	return translated, errs
}

func (t *Translator) translateRange(
	ctx context.Context,
	cfg translateConfig,
	r text.Range,
	input, sourceLang, targetLang string,
) (string, error) {
	extracted, err := text.Extract(input, r)
	if err != nil {
		return "", fmt.Errorf("extract range: %w", err)
	}

	parts := []string{extracted}
	var preserved []preserve.Item

	if cfg.preserve != nil {
		parts, preserved = preserve.Regexp(cfg.preserve, extracted)
	}

	for i, part := range parts {
		translated, err := t.service.Translate(ctx, part, sourceLang, targetLang)
		if err != nil {
			return "", fmt.Errorf("translate: %w", err)
		}
		parts[i] = translated
	}

	return preserve.Join(parts, preserved), nil
}

type translatedRange struct {
	r    text.Range
	text string
}

type translateRangeError struct {
	r   text.Range
	err error
}

func (err translateRangeError) Error() string {
	return fmt.Sprintf("translate range [%d, %d): %s", err.r[0], err.r[1], err.err)
}
