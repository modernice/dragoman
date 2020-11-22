package translator

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"

	"github.com/bounoable/translator/text"
	"github.com/bounoable/translator/text/preserve"
)

//go:generate mockgen -source=translate.go -destination=./mocks/translate.go

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
	var cfg translateConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	inputBytes, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	inputText := string(inputBytes)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ranges, rangeErrs := ranger.Ranges(ctx, input)
	translatedRanges, translateRangeErrs := t.goTranslateRanges(ctx, cfg, ranges, inputText, sourceLang, targetLang)

	var translations []translatedRange

L:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-rangeErrs:
			return nil, fmt.Errorf("get ranges: %w", err)
		case err := <-translateRangeErrs:
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

type translateConfig struct {
	preserve *regexp.Regexp
}

func (t *Translator) goTranslateRanges(
	ctx context.Context,
	cfg translateConfig,
	ranges <-chan text.Range,
	input, sourceLang, targetLang string,
) (<-chan translatedRange, <-chan *translateRangeError) {
	translated := make(chan translatedRange, len(ranges))
	errs := make(chan *translateRangeError, len(ranges))

	go func() {
		defer close(translated)
		for {
			select {
			case <-ctx.Done():
				return
			case r, ok := <-ranges:
				if !ok {
					return
				}

				extracted, err := text.Extract(input, r)
				if err != nil {
					errs <- &translateRangeError{r: r, err: fmt.Errorf("extract range: %w", err)}
					break
				}

				parts := []string{extracted}
				var preserved []preserve.Item

				if cfg.preserve != nil {
					parts, preserved = preserve.Regexp(cfg.preserve, extracted)
				}

				for i, part := range parts {
					translated, err := t.service.Translate(ctx, part, sourceLang, targetLang)
					if err != nil {
						errs <- &translateRangeError{r: r, err: err}
						break
					}
					parts[i] = translated
				}

				translated <- translatedRange{
					r:    r,
					text: preserve.Join(parts, preserved),
				}
			}
		}
	}()

	return translated, errs
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
