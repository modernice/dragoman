// Package json implements translation for JSON files.
package json

import (
	"context"
	"fmt"
	"io"

	"github.com/bounoable/translator/json/internal/lex"
	"github.com/bounoable/translator/text"
)

// Ranger returns a JSON file ranger.
func Ranger() text.Ranger {
	return ranger{}
}

type ranger struct{}

func (r ranger) Ranges(ctx context.Context, input io.Reader) (<-chan text.Range, <-chan error) {
	ranges := make(chan text.Range)
	errs := make(chan error)

	tokens := lex.Lex(input)
	go func() {
		defer close(ranges)
		for tok := range tokens {
			switch tok.Type {
			case lex.Error:
				errs <- fmt.Errorf("lex: %s", tok.Value)
				return
			case lex.EOF:
				return
			case lex.String:
				start := uint(tok.Pos + 1)
				end := uint(tok.Pos + len(tok.Value) - 1)
				ranges <- text.Range{start, end}
			}
		}
	}()

	return ranges, errs
}
