package text

//go:generate mockgen -source=text.go -destination=./mocks/text.go

import (
	"context"
	"fmt"
	"io"
)

// Ranger analyzes inputs and returns the ranges, that need to be translated.
type Ranger interface {
	// Ranges returns a channel of text ranges, that need to be translated.
	// Errors that occur during the scan of the input, should be reported
	// through the error channel.
	Ranges(context.Context, io.Reader) (<-chan Range, <-chan error)
}

// A Range consists of the start and end offsets [start, end) of a text.
type Range [2]uint

// Len returns the length of the range.
func (r Range) Len() int {
	return int(r[1] - r[0])
}

// Extract the text at range r from input.
func Extract(input string, r Range) (string, error) {
	l := int(r[1]) - int(r[0])
	if l == 0 {
		return "", nil
	} else if l < 0 {
		return "", &RangeError{Range: r, Message: "negative length range"}
	}

	inputLen := uint(len(input))

	if r[0] >= inputLen {
		return "", &RangeError{
			Range:   r,
			Message: fmt.Sprintf("range start (pos %d) after input end (pos %d)", r[0], inputLen),
		}
	}

	if r[1] > inputLen {
		return "", &RangeError{
			Range:   r,
			Message: fmt.Sprintf("range end (pos %d) after input end (pos %d)", r[1], inputLen),
		}
	}

	return input[r[0]:r[1]], nil
}

// RangeError is a range error.
type RangeError struct {
	Range   Range
	Message string
}

func (err RangeError) Error() string {
	return err.Message
}

// Replace the text at range [r[0], r[1]) with repl.
//
// Example:
//	Replace("This is a sentence.", "was", Range{5, 7}) = "This was a sentence."
func Replace(text, repl string, r Range) (string, error) {
	if tlen := len(text); r.Len() > len(text) {
		return text, &RangeError{
			Range:   r,
			Message: fmt.Sprintf("range [%d, %d) out of bounds [%d, %d)", r[0], r[1], 0, tlen),
		}
	}
	return text[:r[0]] + repl + text[r[1]:], nil
}

// ReplaceMany replaces the contents of input, according to replacements.
//
// Example:
//	ReplaceMany(
//		"This is a sentence.",
//		Replacement{Range: Range{0, 4}, Text: "Hi,"},
//		Replacement{Range: Range{5, 7}, Text: "I am"},
//	) = "Hi, I am a sentence."
func ReplaceMany(input string, replacements ...Replacement) (string, error) {
	type offset struct {
		start  uint
		offset int
	}

	output := input
	var offsets []offset

	for _, repl := range replacements {
		var off int
		for _, offset := range offsets {
			if offset.start <= repl.Range[0] {
				off += offset.offset
			}
		}

		output = output[:int(repl.Range[0])+off] + repl.Text + output[int(repl.Range[1])+off:]

		orgText, err := Extract(input, repl.Range)
		if err != nil {
			return "", fmt.Errorf("extract text: %w", err)
		}

		lenDiff := len(repl.Text) - len(orgText)
		if lenDiff != 0 {
			offsets = append(offsets, offset{start: repl.Range[1] + uint(off), offset: lenDiff})
		}
	}

	return output, nil
}

// Replacement is a ReplaceMany() replacement configuration.
type Replacement struct {
	// Range is the text range, that's being replaced.
	Range Range
	// Text is the replacement text.
	Text string
}
