package dragoman

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/modernice/dragoman/internal/chunks"
)

// Translator provides facilities for converting text from one language to
// another while optionally preserving specific terms and adhering to additional
// translation instructions. It supports translating large documents by
// splitting them into manageable chunks based on specified delimiters. The
// process respects the contextual nuances of the source and target languages,
// ensuring that the structural integrity and formatting of the original
// document are maintained. Errors during the translation process are handled
// gracefully, providing detailed error messages that facilitate
// troubleshooting.
type Translator struct {
	model Model
}

// TranslateParams specifies the parameters for translating text from one
// language to another, including instructions on how text should be handled
// during translation and any terms that should be preserved unchanged. It also
// defines how to segment the text for translation if necessary.
type TranslateParams struct {
	Document string

	// Source is the language of the document to translate.
	Source string

	// Target is the language to translate the document to.
	Target string

	// Preserve is a list of terms that should not be translated. Useful for
	// preserving brand names.
	Preserve []string

	// Instructions are raw instructions that should be included in the prompt.
	Instructions []string

	// SplitChunks is a list of strings that should be used to split the document
	// into chunks. If the document is split into chunks, each chunk will be
	// translated separately, allowing to fit large documents into the model's

	SplitChunks []string
}

// NewTranslator creates a new instance of a translator, initializing it with a
// provided model for language translation tasks. It returns a [*Translator].
func NewTranslator(svc Model) *Translator {
	return &Translator{
		model: svc,
	}
}

// Translate converts the content of a document from one language to another
// according to specified parameters. It processes the document in potentially
// multiple segments, preserving specified terms and formatting instructions.
// The function returns the translated text or an error if the translation
// fails. Input parameters and context are provided by a [TranslateParams] and
// [context.Context], respectively.
func (t *Translator) Translate(ctx context.Context, params TranslateParams) (string, error) {
	if params.Target == "" {
		params.Target = "English"
	}

	docChunks := chunks.Chunks(params.Document, params.SplitChunks)
	result := make([]string, 0, len(docChunks))
	for _, chunk := range docChunks {
		translated, err := t.translateChunk(ctx, chunk, params)
		if err != nil {
			return "", fmt.Errorf("translate chunk: %w", err)
		}
		result = append(result, translated)
	}

	return addNewline(strings.Join(result, "\n\n")), nil
}

func (t *Translator) translateChunk(ctx context.Context, chunk string, params TranslateParams) (string, error) {
	var from string
	if params.Source != "" {
		from = fmt.Sprintf("from %s ", params.Source)
	}

	instructions := append([]string{
		"Preserve the original document structure and formatting.",
		"Preserve code blocks, placeholders, HTML tags and other structures.",
	}, params.Instructions...)

	if len(params.Preserve) > 0 {
		instructions = append(instructions, fmt.Sprintf("Do not translate the following terms: %s", strings.Join(params.Preserve, ", ")))
	}

	prompt := heredoc.Docf(`
		Translate the following document %sto %s:
		---<DOC_BEGIN>---
		%s
		---<DOC_END>---

		%s

		Output only the translated document, no chat.
	`,
		from,
		params.Target,
		chunk,
		strings.Join(instructions, "\n"),
	)

	response, err := t.model.Chat(ctx, prompt)
	if err != nil {
		return "", err
	}

	response = trimDividers(response)

	return response, nil
}

func trimDividers(text string) string {
	lines := strings.Split(text, "\n")
	out := slices.Clone(lines)

	if len(out) < 1 {
		return text
	}

	if out[0] == "---<DOC_BEGIN>---" {
		out = out[1:]
	}

	if len(out) > 0 && out[len(out)-1] == "---<DOC_END>---" {
		out = out[:len(out)-1]
	}

	return strings.TrimSpace(strings.Join(out, "\n"))
}

func addNewline(text string) string {
	if text == "" {
		return text
	}

	if !strings.HasSuffix(text, "\n") {
		return text + "\n"
	}

	return text
}
