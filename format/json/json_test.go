package json_test

import (
	"context"
	"strings"
	"testing"

	"github.com/bounoable/dragoman/format/json"
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
			name: "more nested object",
			input: `{
				"nested": {
					"title": "This is a title.",
					"description": "This is a description.",
					"nested2": {
						"nested3": "Hello."
					}
				}
			}`,
			expected: []text.Range{
				{33, 49},
				{73, 95},
				{134, 140},
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
		{
			name: "object with umlauts",
			input: `{
				"nested": {
					"key1": "Hällo.",
					"key2": "Müst.",
					"key3": "Göödbye."
				}
			}`,
			expected: []text.Range{
				{32, 38},
				{55, 60},
				{77, 85},
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
