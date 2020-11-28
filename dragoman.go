package dragoman

//go:generate mockgen -source=dragoman.go -destination=./mocks/dragoman.go

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/bounoable/dragoman/text"
	"github.com/bounoable/dragoman/text/preserve"
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
	inputTextReader := io.TeeReader(input, &rangerInput)

	b, err := ioutil.ReadAll(inputTextReader)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	inputText := string(b)
	translateInput := strings.NewReader(inputText)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ranges, rangeErrs := ranger.Ranges(ctx, &rangerInput)
	translatedRanges, translateRangeErrs := t.translateRanges(ctx, cfg, ranges, translateInput, sourceLang, targetLang)

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
	inputReader io.Reader,
	sourceLang, targetLang string,
) (<-chan translatedRange, <-chan error) {
	translated := make(chan translatedRange, len(ranges))
	errs := make(chan error, len(ranges))

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

	readers := map[int]*strings.Reader{}
	b, err := ioutil.ReadAll(inputReader)
	if err != nil {
		errs <- fmt.Errorf("read input: %w", err)
	}
	input := string(b)
	for i := 0; i < workers; i++ {
		readers[i] = strings.NewReader(input)
	}

	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			input := readers[i]

			for {
				select {
				case <-ctx.Done():
					return
				case r, ok := <-ranges:
					if !ok {
						return
					}

					if _, err := input.Seek(0, io.SeekStart); err != nil {
						errs <- fmt.Errorf("seek start of input: %w", err)
						break
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
		}(i)
	}

	return translated, errs
}

func (t *Translator) translateRange(
	ctx context.Context,
	cfg translateConfig,
	r text.Range,
	input io.Reader,
	sourceLang, targetLang string,
) (string, error) {
	extracted, err := text.Extract(input, r)
	if err != nil {
		return "", fmt.Errorf("extract range %v: %w", r, err)
	}

	parts := []string{extracted}
	var preserved []preserve.Item

	if cfg.preserve != nil {
		parts, preserved = preserve.Regexp(cfg.preserve, extracted)
	}

	for i, part := range parts {
		var leftSpace, rightSpace []rune
		part = strings.TrimLeftFunc(part, func(r rune) bool {
			if unicode.IsSpace(r) {
				leftSpace = append(leftSpace, r)
				return true
			}
			return false
		})
		part = strings.TrimRightFunc(part, func(r rune) bool {
			if unicode.IsSpace(r) {
				rightSpace = append(rightSpace, r)
				return true
			}
			return false
		})

		if isPunctuation(part) {
			continue
		}

		translated, err := t.service.Translate(ctx, part, sourceLang, targetLang)
		if err != nil {
			return "", fmt.Errorf("translate '%v': %w", part, err)
		}

		if len(leftSpace) == 0 && len(rightSpace) == 0 {
			parts[i] = translated
			continue
		}

		var builder strings.Builder
		builder.WriteString(string(leftSpace))
		builder.WriteString(translated)
		builder.WriteString(string(rightSpace))
		parts[i] = builder.String()
	}

	return preserve.Join(parts, preserved), nil
}

var (
	punctRE = regexp.MustCompile(`^[[:punct:]]+$`)
)

func isPunctuation(s string) bool {
	return punctRE.MatchString(s)
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
