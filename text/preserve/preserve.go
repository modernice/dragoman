// Package preserve cuts substrings out of strings with the ability to reinsert them at a later time.
package preserve

import (
	"regexp"
	"strings"
)

// Regexp slices text into all substrings seperated by expr.
//
// The returned string slice `parts` contains the substrings without the seperators.
// Removed seperators are stored in items, together with the index at which
// they have to be reinserted to reconstruct the text.
//
// Example:
//   parts, preserved := preserve.Regexp(
//     regexp.MustCompile("{[a-zA-Z]+?}"),
//     "Hello {firstName}, this is a text with a {placeholder} variable.",
//   )
//   // parts: ["Hello ", "this is a text with a ", " variable."]
//   // preserved: [{{firstName} 1}, {{placeholder} 2}]
//
// You can then use preserve.Join() to reconstruct the orignal text:
//   original := preserve.Join(parts, preserved)
//   // original: "Hello {firstName}, this is a text with a {placeholder} variable."
func Regexp(expr *regexp.Regexp, text string) (parts []string, items []Item) {
	matches := expr.FindAllStringIndex(text, -1)
	var textStart int
	var partIndex int
	for _, match := range matches {
		partIndex++
		t := text[textStart:match[0]]
		textStart = match[1]
		if t != "" {
			parts = append(parts, t)
		} else {
			partIndex--
		}
		items = append(items, Item{Text: text[match[0]:match[1]], Index: partIndex})
	}

	if textStart < len(text) {
		parts = append(parts, text[textStart:])
	}

	return parts, items
}

// Join the substrings in parts and insert items at the given indices.
//
// Example:
//   result := preserve.Join(
//     []string{"Hello ", ", how are you ", "?"},
//     []preserve.Item{
//       {Text: "Bob", Index: 1},
//		 {Text: "today", Index: 2},
//     },
//   )
//   // result: "Hello Bob, how are you today?"
func Join(parts []string, items []Item) string {
	if len(items) == 0 {
		return strings.Join(parts, "")
	}

	var result strings.Builder
	for i, part := range parts {
		for j, item := range items {
			if item.Index != i {
				continue
			}
			result.WriteString(item.Text)
			items = items[j+1:]
			break
		}
		result.WriteString(part)
	}

	return result.String()
}

// Item is a `preserved` item from a text.
type Item struct {
	// The preserved text.
	Text string
	// Index at which the text has to be reinserted.
	Index int
}
