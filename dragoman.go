package dragoman

import (
	"context"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
)

// Model is an interface that represents a chat-based translation model. It
// provides a method called Chat, which takes a context and a prompt string as
// input and returns the translated text and any error that occurred during
// translation.
type Model interface {
	// Chat function takes a context and a prompt as input and returns a string and
	// an error. It uses the provided context and prompt to initiate a chat session
	// and retrieve a response.
	Chat(context.Context, string) (string, error)
}

// ModelFunc is a type that represents a function that can be used as a model
// for chat translation. It implements the Model interface and allows for chat
// translation by calling the function with a context and prompt string.
type ModelFunc func(context.Context, string) (string, error)

// Chat is a function that initiates a conversation with the model to translate
// a document. It takes a context and a prompt as input parameters, and returns
// the translated document as a string along with any errors encountered.
func (chat ModelFunc) Chat(ctx context.Context, prompt string) (string, error) {
	return chat(ctx, prompt)
}

// Translator is a type that represents a translator service. It provides
// methods to translate documents from one language to another. The Translate
// method takes a document and optional translation options, such as the source
// and target languages, and terms to preserve. It returns the translated
// document as a string.
type Translator struct {
	model Model
}

// TranslateOption is a type that represents an option for configuring
// translation parameters. It allows users to specify the source language,
// target language, and terms to preserve during translation. Users can create
// TranslateOption instances using the Source, Target, and Preserve functions
// provided by the Translator type. These options can then be passed to the
// Translate method of the Translator to customize the translation process.
type TranslateOption func(*parameters)

type parameters struct {
	source   string
	target   string
	preserve []string
}

// New creates a new Translator with the provided Model.
func New(svc Model) *Translator {
	return &Translator{
		model: svc,
	}
}

// Source sets the source language for translation. It is an option that can be
// passed to the Translate function of the Translator type. The source language
// determines the language in which the original document is written.
func Source(lang string) TranslateOption {
	return func(p *parameters) {
		p.source = lang
	}
}

// Target sets the target language for translation. The Translate function will
// translate the document to the specified target language.
func Target(lang string) TranslateOption {
	return func(p *parameters) {
		p.target = lang
	}
}

// Preserve is a function that allows you to specify terms to be preserved
// during translation. These terms will not be translated and will be kept in
// the original language.
func Preserve(terms ...string) TranslateOption {
	return func(p *parameters) {
		p.preserve = append(p.preserve, terms...)
	}
}

// Translate method translates a given document from a specified source language
// to a target language using the provided translation options. It preserves the
// original document structure and formatting, excludes translation of code
// blocks, placeholders, and HTML tags. Additionally, it allows specifying terms
// to be preserved and not translated. The translated document is returned as
// the output.
func (t *Translator) Translate(ctx context.Context, document string, opts ...TranslateOption) (string, error) {
	var params parameters
	for _, opt := range opts {
		opt(&params)
	}

	if params.target == "" {
		params.target = "English"
	}

	var from string
	if params.source != "" {
		from = fmt.Sprintf("from %s ", params.source)
	}

	rules := []string{
		"Preserve the original document structure and formatting.",
		"Do not translate any code blocks, placeholders, or HTML tags.",
	}

	if len(params.preserve) > 0 {
		rules = append(rules, fmt.Sprintf("Do not translate the following terms: %s", strings.Join(params.preserve, ", ")))
	}

	prompt := heredoc.Docf(`
		Translate the following document %sto %s:
		--------------------------------------------
		%s
		--------------------------------------------

		%s

		Output only the translated document and nothing else.
	`,
		from,
		params.target,
		document,
		strings.Join(rules, "\n"),
	)

	return t.model.Chat(ctx, prompt)
}
