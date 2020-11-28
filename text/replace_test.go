package text_test

import (
	"fmt"
	"testing"

	"github.com/bounoable/dragoman/text"
	"github.com/stretchr/testify/assert"
)

func TestReplace(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		rang          text.Range
		replacement   string
		expected      string
		expectedError error
	}{
		{
			name:        "single line",
			input:       "This is a sentence.",
			rang:        text.Range{5, 9},
			replacement: "could be a",
			expected:    `This could be a sentence.`,
		},
		{
			name: "multiline",
			input: `This is a
multiline sentence.`,
			rang:        text.Range{8, 19},
			replacement: "now a singleline",
			expected:    `This is now a singleline sentence.`,
		},
		{
			name:        "zero length",
			input:       "This is a sentence.",
			rang:        text.Range{8, 8},
			replacement: "still ",
			expected:    `This is still a sentence.`,
		},
		{
			name:        "out of bounds",
			input:       "This is a sentence.",
			rang:        text.Range{0, 20},
			replacement: "A fresh new start.",
			expected:    "This is a sentence.",
			expectedError: &text.RangeError{
				Range:   text.Range{0, 20},
				Message: fmt.Sprintf("range [%d, %d) out of bounds [%d, %d)", 0, 20, 0, 19),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := text.Replace(test.input, test.replacement, test.rang)
			assert.Equal(t, test.expectedError, err)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestReplaceMany(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		replacements []text.Replacement
		expected     string
	}{
		{
			name:  "single replacement",
			input: "This is a sentence.",
			replacements: []text.Replacement{
				{Range: text.Range{5, 7}, Text: "was"},
			},
			expected: "This was a sentence.",
		},
		{
			name: "multiple replacements",
			input: `This is a
multiline sentence, that
spans over 4
lines.`,
			replacements: []text.Replacement{
				{Range: text.Range{0, 4}, Text: "That"},
				{Range: text.Range{5, 7}, Text: "was"},
				{Range: text.Range{9, 10}, Text: " "},
				{Range: text.Range{15, 15}, Text: "-"},
				{Range: text.Range{28, 35}, Text: ". It "},
				{Range: text.Range{39, 40}, Text: "ned"},
				{Range: text.Range{47, 48}, Text: " "},
			},
			expected: "That was a multi-line sentence. It spanned over 4 lines.",
		},
		{
			name: "nested json",
			input: `{
				"nested1": {
					"nested2": {
						"key1": "Hello.",
						"key2": "Goodbye."
					}
					"nested3": {
						"key3": "Hi.",
						"key4": "Bye.",
						"key5": "No."
					}
				}
			}`,
			replacements: []text.Replacement{
				{Range: text.Range{52, 58}, Text: "Hi."},
				{Range: text.Range{76, 84}, Text: "Goodbye, my friend. It was a pleasure."},
				{Range: text.Range{126, 129}, Text: "Hello."},
				{Range: text.Range{147, 151}, Text: "Farewell."},
				{Range: text.Range{169, 172}, Text: "Sure thing."},
			},
			expected: `{
				"nested1": {
					"nested2": {
						"key1": "Hi.",
						"key2": "Goodbye, my friend. It was a pleasure."
					}
					"nested3": {
						"key3": "Hello.",
						"key4": "Farewell.",
						"key5": "Sure thing."
					}
				}
			}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := text.ReplaceMany(test.input, test.replacements...)
			assert.Nil(t, err)
			assert.Equal(t, test.expected, result)
		})
	}
}
