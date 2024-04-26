package dragoman_test

import (
	"context"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/modernice/dragoman"
)

func TestTranslator_Translate(t *testing.T) {
	source := heredoc.Docf(`{
		"hallo": "Hallo Welt!"
	}`)

	wantPrompt := heredoc.Docf(`
		Translate the following document to English:
		---<DOC_BEGIN>---
		%s
		---<DOC_END>---

		Preserve the original document structure and formatting.
		Preserve code blocks, placeholders, HTML tags and other structures.

		Output only the translated document, no chat.
	`, source)

	prompt(wantPrompt).expect(t, dragoman.TranslateParams{Document: source})
}

func TestSource(t *testing.T) {
	source := heredoc.Docf(`{
		"hallo": "Hallo Welt!"
	}`)

	wantPrompt := heredoc.Docf(`
		Translate the following document from French to English:
		---<DOC_BEGIN>---
		%s
		---<DOC_END>---

		Preserve the original document structure and formatting.
		Preserve code blocks, placeholders, HTML tags and other structures.

		Output only the translated document, no chat.
	`, source)

	prompt(wantPrompt).expect(t, dragoman.TranslateParams{Document: source, Source: "French"})
}

func TestTarget(t *testing.T) {
	source := heredoc.Docf(`{
		"hallo": "Hallo Welt!"
	}`)

	wantPrompt := heredoc.Docf(`
		Translate the following document to French:
		---<DOC_BEGIN>---
		%s
		---<DOC_END>---

		Preserve the original document structure and formatting.
		Preserve code blocks, placeholders, HTML tags and other structures.

		Output only the translated document, no chat.
	`, source)

	prompt(wantPrompt).expect(t, dragoman.TranslateParams{Document: source, Target: "French"})
}

func TestPreserve(t *testing.T) {
	source := heredoc.Docf(`{
		"hallo": "Hallo, ich bin der HalloWeltBot!"
	}`)

	wantPrompt := heredoc.Docf(`
		Translate the following document to English:
		---<DOC_BEGIN>---
		%s
		---<DOC_END>---

		Preserve the original document structure and formatting.
		Preserve code blocks, placeholders, HTML tags and other structures.
		Do not translate the following terms: HalloWeltBot

		Output only the translated document, no chat.
	`, source)

	prompt(wantPrompt).expect(t, dragoman.TranslateParams{Document: source, Preserve: []string{"HalloWeltBot"}})
}

func TestPreserve_multiple(t *testing.T) {
	source := heredoc.Docf(`{
		"hallo": "Hallo, ich bin der HalloWeltBot aus der WeltFabrik!"
	}`)

	wantPrompt := heredoc.Docf(`
		Translate the following document to English:
		---<DOC_BEGIN>---
		%s
		---<DOC_END>---

		Preserve the original document structure and formatting.
		Preserve code blocks, placeholders, HTML tags and other structures.
		Do not translate the following terms: HalloWeltBot, WeltFabrik

		Output only the translated document, no chat.
	`, source)

	prompt(wantPrompt).expect(t, dragoman.TranslateParams{Document: source, Preserve: []string{"HalloWeltBot", "WeltFabrik"}})
}

type prompt string

func (p prompt) expect(t *testing.T, params dragoman.TranslateParams) {
	t.Helper()

	var providedPrompt string
	model := dragoman.ModelFunc(func(_ context.Context, prompt string) (string, error) {
		providedPrompt = prompt
		return "", nil
	})

	trans := dragoman.NewTranslator(model)

	if _, err := trans.Translate(context.Background(), params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if providedPrompt != string(p) {
		t.Errorf("expected prompt to be\n\n%s\n\nbut prompt was\n\n%s", p, providedPrompt)
	}
}
