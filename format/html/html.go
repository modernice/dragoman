// Package html provides translation of HTML files.
package html

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/bounoable/dragoman/text"
	"golang.org/x/net/html"
)

// Ranger returns an HTML file ranger.
func Ranger(opts ...Option) text.Ranger {
	var r ranger
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

// Option is a ranger option.
type Option func(*ranger)

// WithAttributeFunc allows translations of HTML tag attributes.
//
// fns will be called for every html token that is a start tag or a
// self-closing start tag. The return values of fns will be merged and
// the resulting string slice determines the attributes that should be
// translated for the given HTML token.
func WithAttributeFunc(fns ...func(html.Token) []string) Option {
	return func(r *ranger) {
		r.attributeFuncs = append(r.attributeFuncs, fns...)
	}
}

// WithAttribute allows translations of HTML tag attributes with the specified name.
//
// If tags is not empty, this option is only applied to the specified tags.
// Otherwise this option applies to all tags.
func WithAttribute(name string, tags ...string) Option {
	return WithAttributeFunc(func(tok html.Token) []string {
		if len(tags) == 0 {
			return []string{name}
		}
		for _, tag := range tags {
			if tok.Data == tag {
				return []string{name}
			}
		}
		return nil
	})
}

type ranger struct {
	attributeFuncs []func(html.Token) []string
}

func (r ranger) Ranges(ctx context.Context, input io.Reader) (<-chan text.Range, <-chan error) {
	ranges := make(chan text.Range)
	errs := make(chan error)

	go func() {
		defer close(ranges)
		defer close(errs)

		tokenizer := html.NewTokenizer(input)
		var pos uint

		for {
			select {
			case <-ctx.Done():
				errs <- ctx.Err()
				return
			default:
			}

			tt := tokenizer.Next()
			tok := tokenizer.Token()
			err := tokenizer.Err()
			if errors.Is(err, io.EOF) {
				return
			}

			if err != nil {
				errs <- fmt.Errorf("tokenizer: %w", err)
				return
			}

			l := uint(len(tokenizer.Raw()))
			switch tt {
			case html.ErrorToken:
				errs <- fmt.Errorf("tokenizer: %w", err)
				return
			case html.TextToken:
				if strings.TrimSpace(tok.Data) != "" {
					ranges <- text.Range{pos, pos + l}
				}
			case html.StartTagToken, html.SelfClosingTagToken:
				attrs := r.filterAttributes(tok)
				if len(attrs) == 0 {
					break
				}

				attrRanges, err := r.rangeAttributes(ctx, tok, attrs)
				if err != nil {
					errs <- fmt.Errorf("range attributes: %w", err)
					return
				}

				for _, rang := range attrRanges {
					ranges <- text.Range{rang[0] + pos, rang[1] + pos}
				}
			}
			pos += l
		}
	}()

	return ranges, errs
}

func (r ranger) filterAttributes(t html.Token) (attrs []string) {
	if len(t.Attr) == 0 {
		return
	}
	if len(r.attributeFuncs) == 0 {
		return
	}
	for _, fn := range r.attributeFuncs {
		attrs = append(attrs, fn(t)...)
	}
	return
}

var (
	attrRE = regexp.MustCompile(`(?P<NAME>[[:word:]]+?)="(?P<VALUE>.+?)"`)
)

func (r ranger) rangeAttributes(ctx context.Context, tok html.Token, attrs []string) ([]text.Range, error) {
	var ranges []text.Range

	fullTag := tok.String()
	matches := attrRE.FindAllStringIndex(fullTag, -1)
	for _, match := range matches {
		attr := fullTag[match[0]:match[1]]
		name := attrRE.ReplaceAllString(attr, fmt.Sprintf("${%s}", attrRE.SubexpNames()[attrRE.SubexpIndex("NAME")]))

		var allowed bool
		for _, allowedName := range attrs {
			if name == allowedName {
				allowed = true
				break
			}
		}

		if !allowed {
			continue
		}

		val := attrRE.ReplaceAllString(attr, fmt.Sprintf("${%s}", attrRE.SubexpNames()[attrRE.SubexpIndex("VALUE")]))

		offset := strings.Index(attr, val)
		start := uint(match[0] + offset)
		end := start + uint(len(val))
		ranges = append(ranges, text.Range{start, end})
	}

	return ranges, nil
}
