package markdown

import (
	"regexp"
	"strings"
)

// Chunks partitions the given string into sections based on specified heading
// levels, returning these sections as a slice of strings. Each section includes
// lines from one heading level to the next matching one in the list or to the
// end of the string if no further matching heading is found.
func Chunks(source string, splitLevels []int) []string {
	if len(splitLevels) == 0 {
		return []string{source}
	}

	lines := strings.Split(source, "\n")

	splitLevelREs := make([]*regexp.Regexp, 0, len(splitLevels))
	for _, level := range splitLevels {
		if level < 1 {
			continue
		}

		splitLevelREs = append(splitLevelREs, regexp.MustCompile("^"+strings.Repeat("#", level)+`\s`))
	}

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

		for _, level := range splitLevelREs {
			if level.MatchString(line) {
				appendChunk()
				break
			}
		}

		currentChunk = append(currentChunk, line)
	}

	appendChunk()

	return chunks
}
