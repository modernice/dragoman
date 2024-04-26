package dragoman

import "context"

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
