package openai

import (
	"errors"
	"sync"

	"github.com/tiktoken-go/tokenizer"
)

var (
	codecsMux sync.Mutex
	codecs    = make(map[string]tokenizer.Codec)
)

// PromptTokens computes the number of tokens in the given prompt string for the
// specified model. It returns the token count and any error encountered during
// tokenization.
func PromptTokens(model string, prompt string) (int, error) {
	codec, err := getCodec(model)
	if err != nil {
		return 0, err
	}
	toks, _, err := codec.Encode(prompt)
	return len(toks), err
}

func getCodec(model string) (tokenizer.Codec, error) {
	codecsMux.Lock()
	defer codecsMux.Unlock()

	codec, ok := codecs[model]
	if !ok {
		var err error
		if codec, err = getTokenizer(model); err != nil {
			return nil, err
		}
		codecs[model] = codec
	}

	return codec, nil
}

func getTokenizer(model string) (tokenizer.Codec, error) {
	codec, err := tokenizer.ForModel(tokenizer.Model(model))
	if errors.Is(err, tokenizer.ErrModelNotSupported) {
		return tokenizer.Get(tokenizer.Cl100kBase)
	}
	return codec, err
}
