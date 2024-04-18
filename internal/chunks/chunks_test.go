package chunks_test

import (
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/modernice/dragoman/internal/chunks"
)

func TestChunks(t *testing.T) {
	source := strings.TrimSpace(heredoc.Docf(`
		# Title

		Introduction.

		## Section 1

		Content.

		## Section 2

		More content.

		### Subsection

		Even more content.

		## Section 3

		Final content.

		## Conclusion

		Last words.
	`))

	tests := []struct {
		name          string
		splitPrefixes []string
		expected      []string
	}{
		{
			name:     "no levels",
			expected: []string{source},
		},
		{
			name:          "heading #1",
			splitPrefixes: []string{"# "},
			expected:      []string{source},
		},
		{
			name:          "heading #2",
			splitPrefixes: []string{"## "},
			expected: []string{
				takeLines(source, 3),
				skipAndTakeLines(source, 4, 3),
				skipAndTakeLines(source, 8, 7),
				skipAndTakeLines(source, 16, 3),
				skipAndTakeLines(source, 20, 3),
			},
		},
		{
			name:          "heading #3",
			splitPrefixes: []string{"### "},
			expected: []string{
				takeLines(source, 11),
				skipAndTakeLines(source, 12, 11),
			},
		},
		{
			name:          "heading #1 and #2",
			splitPrefixes: []string{"# ", "## "},
			expected: []string{
				takeLines(source, 3),
				skipAndTakeLines(source, 4, 3),
				skipAndTakeLines(source, 8, 7),
				skipAndTakeLines(source, 16, 3),
				skipAndTakeLines(source, 20, 3),
			},
		},
		{
			name:          "heading #2 and #3",
			splitPrefixes: []string{"## ", "### "},
			expected: []string{
				takeLines(source, 3),
				skipAndTakeLines(source, 4, 3),
				skipAndTakeLines(source, 8, 3),
				skipAndTakeLines(source, 12, 3),
				skipAndTakeLines(source, 16, 3),
				skipAndTakeLines(source, 20, 3),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunks.Chunks(source, tt.splitPrefixes)

			if len(tt.expected) != len(chunks) {
				t.Fatalf("unexpected number of chunks. want %d; got %d", len(tt.expected), len(chunks))
			}

			if !cmp.Equal(tt.expected, chunks) {
				t.Errorf("unexpected chunks (-want +got):\n%s", cmp.Diff(tt.expected, chunks))
			}
		})
	}
}

func takeLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if n >= len(lines) {
		return s
	}
	return strings.Join(lines[:n], "\n")
}

func skipLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if n >= len(lines) {
		return ""
	}
	return strings.Join(lines[n:], "\n")
}

func skipAndTakeLines(s string, skip, take int) string {
	return takeLines(skipLines(s, skip), take)
}
