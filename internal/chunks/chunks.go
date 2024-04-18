package chunks

import (
	"strings"
)

// Chunks splits a string into segments based on line prefixes specified in a
// slice. If no prefixes are provided, it returns the entire string as a single
// segment. Each segment is trimmed of leading and trailing whitespace.
func Chunks(source string, splitPrefixes []string) []string {
	if len(splitPrefixes) == 0 {
		return []string{source}
	}

	lines := strings.Split(source, "\n")

	var chunks []string
	var currentChunk []string

	appendChunk := func() {
		if len(currentChunk) == 0 {
			return
		}

		chunks = append(chunks, strings.TrimSpace(strings.Join(currentChunk, "\n")))
		currentChunk = currentChunk[:0]
	}

	for _, line := range lines {
		if len(currentChunk) == 0 {
			currentChunk = append(currentChunk, line)
			continue
		}

		for _, prefix := range splitPrefixes {
			if strings.HasPrefix(line, prefix) {
				appendChunk()
				break
			}
		}

		currentChunk = append(currentChunk, line)
	}

	appendChunk()

	return chunks
}
