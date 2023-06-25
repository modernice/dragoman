package text

//go:generate mockgen -source=text.go -destination=./mocks/text.go

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
)

// Ranger analyzes inputs and returns the ranges, that need to be translated.
type Ranger interface {
	// Ranges returns a channel of text ranges, that need to be translated.
	//
	// Errors that occur during the scan of the input, should be reported
	// through the error channel.
	//
	// The Ranger is responsible for closing the Range channel when it's done.
	Ranges(context.Context, io.Reader) (<-chan Range, <-chan error)
}

// A Range consists of a start and end rune-offset [start, end) of a text.
type Range [2]uint

// Len returns the length of the range.
func (r Range) Len() int {
	return int(r[1] - r[0])
}

// Extract extracts the text at range r from input.
// Use an io.ReadSeeker as input for lower memory consumption.
func Extract(input io.Reader, r Range) (string, error) {
	var rs io.ReadSeeker
	if irs, ok := input.(io.ReadSeeker); ok {
		rs = irs
	} else {
		b, err := ioutil.ReadAll(input)
		if err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		rs = bytes.NewReader(b)
	}

	rangeLen := r.Len()
	if rangeLen == 0 {
		return "", nil
	} else if rangeLen < 0 {
		return "", &RangeError{Range: r, Message: "negative length range"}
	}

	br := bufio.NewReader(rs)

	var run rune
	var err error
	for i := int(r[0]); i >= 0; i-- {
		run, _, err = br.ReadRune()
		if errors.Is(err, io.EOF) {
			return "", &RangeError{
				Range:   r,
				Message: fmt.Sprintf("range start (pos %d) after input end", r[0]),
			}
		}
	}

	runes := []rune{run}
	for l := rangeLen; l > 1; l-- {
		run, _, err := br.ReadRune()
		if errors.Is(err, io.EOF) {
			return "", &RangeError{
				Range:   r,
				Message: fmt.Sprintf("range end (pos %d) after input end (pos %d)", r[1], r[0]+uint(rangeLen-l+1)),
			}
		}
		if err != nil {
			return "", fmt.Errorf("read rune: %w", err)
		}
		runes = append(runes, run)
	}

	return string(runes), nil
}

// ExtractString extracts the text at range r from input.
func ExtractString(input string, r Range) (string, error) {
	return Extract(strings.NewReader(input), r)
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
//
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
//
//	ReplaceMany(
//		"This is a sentence.",
//		Replacement{Range: Range{0, 4}, Text: "Hi,"},
//		Replacement{Range: Range{5, 7}, Text: "I am"},
//	) = "Hi, I am a sentence."
func ReplaceMany(input string, replacements ...Replacement) (string, error) {
	sort.Slice(replacements, func(a, b int) bool {
		return replacements[a].Range[0] < replacements[b].Range[0]
	})

	output := []rune(input)

	var offset int
	for _, repl := range replacements {
		var builder strings.Builder
		builder.WriteString(string(output[:int(repl.Range[0])+offset]))
		builder.WriteString(repl.Text)
		builder.WriteString(string(output[int(repl.Range[1])+offset:]))
		output = []rune(builder.String())

		orgText, err := ExtractString(input, repl.Range)
		if err != nil {
			return "", fmt.Errorf("extract text: %w", err)
		}

		if lenDiff := len([]rune(repl.Text)) - len([]rune(orgText)); lenDiff != 0 {
			offset += lenDiff
		}
	}

	return string(output), nil
}

// Replacement is a ReplaceMany() replacement configuration.
type Replacement struct {
	// Range is the text range, that's being replaced.
	Range Range
	// Text is the replacement text.
	Text string
}
