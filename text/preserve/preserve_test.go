package preserve_test

import (
	"regexp"
	"testing"

	"github.com/bounoable/dragoman/text/preserve"
	"github.com/stretchr/testify/assert"
)

func TestRegexp(t *testing.T) {
	tests := []struct {
		name              string
		text              string
		expr              string
		expectedParts     []string
		expectedPreserved []preserve.Item
	}{
		{
			name: "no placeholders",
			text: "Hello, how are you?",
			expr: "{[a-zA-Z0-9]+?}",
			expectedParts: []string{
				"Hello, how are you?",
			},
		},
		{
			name: "single placeholder",
			text: "Hello, {firstName}, how are you?",
			expr: "{[a-zA-Z0-9]+?}",
			expectedParts: []string{
				"Hello, ",
				", how are you?",
			},
			expectedPreserved: []preserve.Item{
				{Text: "{firstName}", Index: 1},
			},
		},
		{
			name: "multiple placeholders",
			text: "Hello, {firstName}, how are you {day}?",
			expr: "{[a-zA-Z0-9]+?}",
			expectedParts: []string{
				"Hello, ",
				", how are you ",
				"?",
			},
			expectedPreserved: []preserve.Item{
				{Text: "{firstName}", Index: 1},
				{Text: "{day}", Index: 2},
			},
		},
		{
			name: "many placeholders",
			text: "{greeting}, {firstName}, how are you {day}?",
			expr: "{[a-zA-Z0-9]+?}",
			expectedParts: []string{
				", ",
				", how are you ",
				"?",
			},
			expectedPreserved: []preserve.Item{
				{Text: "{greeting}", Index: 0},
				{Text: "{firstName}", Index: 1},
				{Text: "{day}", Index: 2},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expr := regexp.MustCompile(test.expr)
			parts, preserved := preserve.Regexp(expr, test.text)

			assert.Equal(t, test.expectedParts, parts)
			assert.Equal(t, test.expectedPreserved, preserved)
		})
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		items    []preserve.Item
		expected string
	}{
		{
			name: "no items",
			parts: []string{
				"Hello, ",
				"how are you?",
			},
			expected: "Hello, how are you?",
		},
		{
			name: "single item",
			parts: []string{
				"Hello, ",
				", how are you?",
			},
			items: []preserve.Item{
				{Text: "{firstName}", Index: 1},
			},
			expected: "Hello, {firstName}, how are you?",
		},
		{
			name: "multiple items",
			parts: []string{
				"Hello, ",
				", how are you",
				"?",
			},
			items: []preserve.Item{
				{Text: "{firstName}", Index: 1},
			},
			expected: "Hello, {firstName}, how are you?",
		},
		{
			name: "many items",
			parts: []string{
				", ",
				", how are you ",
				"?",
			},
			items: []preserve.Item{
				{Text: "{greeting}", Index: 0},
				{Text: "{firstName}", Index: 1},
				{Text: "{day}", Index: 2},
			},
			expected: "{greeting}, {firstName}, how are you {day}?",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, preserve.Join(test.parts, test.items))
		})
	}
}
