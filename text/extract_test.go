package text_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bounoable/dragoman/text"
	"github.com/stretchr/testify/assert"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		rang          text.Range
		expected      string
		expectedError error
	}{
		// single line
		{
			name:     "single line, start to end",
			input:    `This is a single line text.`,
			rang:     text.Range{0, 27},
			expected: `This is a single line text.`,
		},
		{
			name:     "single line, start to middle",
			input:    `This is a single line text.`,
			rang:     text.Range{0, 13},
			expected: `This is a sin`,
		},
		{
			name:     "single line, middle to end",
			input:    `This is a single line text.`,
			rang:     text.Range{13, 27},
			expected: `gle line text.`,
		},
		{
			name:     "single line, zero length",
			input:    `This is a single line text.`,
			rang:     text.Range{0, 0},
			expected: "",
		},
		{
			name:  "single line, negative length",
			input: `This is a single line text.`,
			rang:  text.Range{3, 0},
			expectedError: &text.RangeError{
				Range:   text.Range{3, 0},
				Message: "negative length range",
			},
		},
		{
			name:  "single line, start out of bounds",
			input: `This is a single line text.`,
			rang:  text.Range{27, 30},
			expectedError: &text.RangeError{
				Range:   text.Range{27, 30},
				Message: fmt.Sprintf("range start (pos %d) after input end", 27),
			},
		},
		{
			name:  "single line, start out of bounds 2",
			input: `This is a single line text.`,
			rang:  text.Range{30, 40},
			expectedError: &text.RangeError{
				Range:   text.Range{30, 40},
				Message: fmt.Sprintf("range start (pos %d) after input end", 30),
			},
		},
		{
			name:  "single line, end out of bounds",
			input: `This is a single line text.`,
			rang:  text.Range{0, 30},
			expectedError: &text.RangeError{
				Range:   text.Range{0, 30},
				Message: fmt.Sprintf("range end (pos %d) after input end (pos %d)", 30, 27),
			},
		},

		// multi line
		{
			name: "multi line, start to end",
			input: `This is a multi line text,
this is the second line.`,
			rang: text.Range{0, 51},
			expected: `This is a multi line text,
this is the second line.`,
		},
		{
			name: "multi line, start to middle",
			input: `This is a multi line text,
this is the second line.`,
			rang: text.Range{0, 31},
			expected: `This is a multi line text,
this`,
		},
		{
			name: "multi line, middle to end",
			input: `This is a multi line text,
this is the second line.`,
			rang: text.Range{15, 51},
			expected: ` line text,
this is the second line.`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Run("text.Extract()", func(t *testing.T) {
				extracted, err := text.Extract(strings.NewReader(test.input), test.rang)
				assert.Equal(t, test.expectedError, err)

				if test.expectedError == nil {
					assert.Equal(t, test.expected, extracted)
				}
			})

			t.Run("text.ExtractString()", func(t *testing.T) {
				extracted, err := text.ExtractString(test.input, test.rang)
				assert.Equal(t, test.expectedError, err)

				if test.expectedError == nil {
					assert.Equal(t, test.expected, extracted)
				}
			})
		})
	}
}
