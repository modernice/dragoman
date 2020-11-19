package translator

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bounoable/translator/text"
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
) ([]byte, error) {
	inputBytes, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	inputText := string(inputBytes)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ranges, rangeErrs := ranger.Ranges(ctx, input)
	translatedRanges, translateRangeErrs := t.goTranslateRanges(ctx, ranges, inputText, sourceLang, targetLang)

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

func (t *Translator) goTranslateRanges(
	ctx context.Context,
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

				result, err := t.service.Translate(ctx, extracted, sourceLang, targetLang)
				if err != nil {
					errs <- &translateRangeError{r: r, err: err}
					break
				}

				translated <- translatedRange{
					r:    r,
					text: result,
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
