package json_test

import (
	"context"
	"strings"
	"testing"

	"github.com/bounoable/dragoman/json"
	"github.com/bounoable/dragoman/text"
	"github.com/stretchr/testify/assert"
)

func TestRanger(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []text.Range
	}{
		{
			name:     "empty object",
			input:    `{}`,
			expected: nil,
		},
		{
			name: "flat object",
			input: `{
				"title": "This is a title.",
				"description": "This is a description."
			}`,
			expected: []text.Range{
				{16, 32},
				{55, 77},
			},
		},
		{
			name: "nested object",
			input: `{
				"nested": {
					"title": "This is a title.",
					"description": "This is a description."
				}
			}`,
			expected: []text.Range{
				{33, 49},
				{73, 95},
			},
		},
		{
			name:  "flat array",
			input: `["Hello", "Bob"]`,
			expected: []text.Range{
				{2, 7},
				{11, 14},
			},
		},
		{
			name: "nested object with array field",
			input: `{
				"nested": {
					"field": ["A", "BB", "CCC"]
				}
			}`,
			expected: []text.Range{
				{34, 35},
				{39, 41},
				{45, 48},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ranger := json.Ranger()
			rangeChan, errChan := ranger.Ranges(context.Background(), strings.NewReader(test.input))

			var ranges []text.Range
			for rang := range rangeChan {
				ranges = append(ranges, rang)
			}

			assert.Empty(t, errChan)
			assert.Equal(t, test.expected, ranges)
		})
	}
}
