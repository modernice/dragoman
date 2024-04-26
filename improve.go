package dragoman

import (
	"context"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/modernice/dragoman/internal/chunks"
)

// Improver enhances the content of a document by making it more engaging,
// informative, and optimized for search engine visibility while preserving its
// structural integrity. It takes into account various parameters such as
// formality, language, and specific keywords to ensure the output is tailored
// to specific needs. The enhanced content is achieved by processing each
// segment of the document separately when necessary, allowing for large
// documents to be handled effectively.
type Improver struct {
	model Model
}

// NewImprover creates a new instance of [Improver] using the provided [Model].
func NewImprover(svc Model) *Improver {
	return &Improver{
		model: svc,
	}
}

// ImproveParams configures the enhancement of a document by specifying its
// content, how to split it for processing, the desired formality tone, SEO
// keywords to incorporate, specific instructions for adjustment, and the
// language in which improvements should be made. It is used by an [Improver] to
// adjust a document's appeal, readability, and search engine optimization.
type ImproveParams struct {
	Document string

	// SplitChunks is a list of strings that should be used to split the document
	// into chunks. If the document is split into chunks, each chunk will be
	// improved separately, allowing to fit large documents into the model's
	// context window.
	SplitChunks []string

	// Formality specifies the formality (formal address) to use in the improved document.
	Formality Formality

	// Keywords are SEO keywords that should be used in the improved document.
	Keywords []string

	// Instructions are raw instructions that should be included in the prompt.
	Instructions []string

	// Language is the language the improved document should be written in.
	Language string
}

// Improve enhances the content of a document based on specified parameters to
// increase engagement, clarity, and search engine optimization. It splits the
// document into manageable chunks if necessary, processes each chunk
// independently according to the improvement criteria including language,
// formality, keywords, and additional instructions, and then reassembles the
// improved chunks into a cohesive output.
func (imp *Improver) Improve(ctx context.Context, params ImproveParams) (string, error) {
	docChunks := []string{params.Document}

	if len(params.SplitChunks) > 0 {
		docChunks = chunks.Chunks(params.Document, params.SplitChunks)
	}

	var result []string

	for _, chunk := range docChunks {
		translated, err := imp.improveChunk(ctx, chunk, params)
		if err != nil {
			return "", err
		}
		result = append(result, translated)
	}

	return addNewline(strings.Join(result, "\n\n")), nil
}

func (imp *Improver) improveChunk(ctx context.Context, chunk string, params ImproveParams) (string, error) {
	optimizeKeywords := "Identify and utilize keywords naturally derived from the document's content."
	if len(params.Keywords) > 0 {
		optimizeKeywords = fmt.Sprintf("Incorporate the following keywords effectively throughout the document: %s", strings.Join(mapSlice(params.Keywords, quote), ", "))
	}

	prompt := strings.TrimSpace(heredoc.Docf(`
		Task: Improve the document provided below. The objective is to enhance the content to be more engaging, informative, and optimized for search engine visibility.

		Instructions:
		1. Preserve Document Elements:
			- Maintain the original formatting elements such as headings, lists, code blocks, and embedded HTML or Markdown tags. These elements are crucial for preserving the structural integrity of the document.
			- Modify or reorganize these elements only to enhance clarity, engagement, or SEO effectiveness, but ensure that the document’s fundamental layout and component functions are not altered.
		2. Content Optimization:
			- Engagement: Increase the text's appeal and readability by refining dense or uninviting sentences. Adjust titles and headings to be more compelling and clear.
			- SEO: Optimize the text for search engines. Incorporate provided keywords effectively throughout the document. %s
		3. You may introduce new sections or headings and reorganize existing content. Ensure these changes enhance the document’s overall message and readability while using the predefined formatting elements mentioned.
		4. Return only the revised document text. Exclude any additional commentary or discussion about the changes made.
	`, optimizeKeywords))

	language := "5. Write in the same language as the original document."
	if params.Language != "" {
		language = fmt.Sprintf("5. Write in the following language: %s", params.Language)
	}

	prompt += fmt.Sprintf("\n%s", language)

	additionalInstructions := make([]string, len(params.Instructions))
	for i, instruction := range params.Instructions {
		additionalInstructions[i] = fmt.Sprintf("%d. %s", i+6, instruction)
	}

	if params.Formality.IsSpecified() {
		additionalInstructions = append(additionalInstructions, fmt.Sprintf("%d. %s", len(additionalInstructions)+6, params.Formality.instruction()))
	}

	if len(additionalInstructions) > 0 {
		prompt += "\n" + strings.Join(additionalInstructions, "\n")
	}

	prompt += fmt.Sprintf("\n\nImprove the following document:\n---<DOC_BEGIN>---\n%s\n---<DOC_END>---", chunk)

	response, err := imp.model.Chat(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("llm error: %w", err)
	}

	return trimDividers(response), nil
}

func quote(s string) string {
	return fmt.Sprintf("%q", s)
}
