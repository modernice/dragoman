package openai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/tiktoken-go/tokenizer"
)

const (
	// DefaultModel is the default language model used for generating text when no
	// specific model is set during the client creation.
	DefaultModel = openai.GPT3Dot5Turbo

	// DefaultTemperature is the default value for the temperature parameter in the
	// AI model. It affects the randomness of the model's output.
	DefaultTemperature = 0.5

	// DefaultTopP is the default value for the "Top P" parameter used in OpenAI's
	// language models.
	DefaultTopP = 0.1

	// DefaultTimeout specifies the default duration to wait before timing out
	// requests to the OpenAI API. This value can be changed by using the Timeout
	// option when creating a new client.
	DefaultTimeout = 30 * time.Second
)

var modelTokens = map[string]int{
	openai.GPT3Dot5Turbo:    4096,
	openai.GPT3Dot5Turbo16K: 16384,
	openai.GPT4:             8192,
	openai.GPT432K:          32768,
	"default":               4096,
}

// Client is a configurable interface to the OpenAI API. It allows for the
// generation of text completions using various models, with adjustable
// parameters for token count, temperature, and topP. A specified timeout can be
// set for API requests.
type Client struct {
	model       string
	maxTokens   int
	temperature float32
	topP        float32
	timeout     time.Duration
	client      *openai.Client
}

// Option is a function type used to configure a Client. It allows for setting
// various parameters such as the model, maximum tokens, temperature, topP, and
// timeout. These options are applied to a Client instance during its creation
// with the New function.
type Option func(*Client)

// Model is a function that returns an Option which sets the model string field
// of a Client object when called. The model string represents the specific
// OpenAI model to be used for text generation tasks.
func Model(model string) Option {
	return func(m *Client) {
		m.model = model
	}
}

// MaxTokens sets the maximum number of tokens that the Client's model can
// generate. It is an option that can be passed when creating a new Client.
// If not set, the default max tokens for the selected model will be used.
func MaxTokens(maxTokens int) Option {
	return func(m *Client) {
		m.maxTokens = maxTokens
	}
}

// Temperature sets the temperature parameter for the Client. The temperature
// affects the randomness of the model's output during text generation tasks.
func Temperature(temperature float32) Option {
	return func(m *Client) {
		m.temperature = temperature
	}
}

// TopP sets the topP parameter for the Client.
func TopP(topP float32) Option {
	return func(m *Client) {
		m.topP = topP
	}
}

// Timeout is a function that sets the timeout duration for the Client. It
// returns an Option that, when provided to the New function, modifies the
// timeout duration of the created Client instance. The timeout duration
// determines how long the Client waits for a response before it cancels the
// request.
func Timeout(timeout time.Duration) Option {
	return func(m *Client) {
		m.timeout = timeout
	}
}

// New creates a new Client instance with the specified API token and optional
// configuration options. The Client allows for the generation of text
// completions using various models, with adjustable parameters for token count,
// temperature, and topP. The default values for these parameters are used if
// not explicitly set. The Client also supports setting a timeout duration for
// API requests.
func New(apiToken string, opts ...Option) *Client {
	m := Client{
		temperature: DefaultTemperature,
		topP:        DefaultTopP,
		timeout:     DefaultTimeout,
		client:      openai.NewClient(apiToken),
	}
	for _, opt := range opts {
		opt(&m)
	}

	if m.model == "" {
		m.model = DefaultModel
	}

	var ok bool
	if m.maxTokens, ok = modelTokens[m.model]; !ok {
		m.maxTokens = modelTokens["default"]
	}

	return &m
}

// Chat generates text completion based on the given prompt using the configured
// OpenAI language model. The generated text completion is returned as a string.
func (m *Client) Chat(ctx context.Context, prompt string) (string, error) {
	tok, err := tokenizer.ForModel(tokenizer.Model(m.model))
	if err != nil {
		return "", fmt.Errorf("get tokenizer for %q: %w", m.model, err)
	}

	promptTokens, _, err := tok.Encode(prompt)
	if err != nil {
		return "", fmt.Errorf("encode prompt: %w", err)
	}
	maxTokens := m.maxTokens - len(promptTokens)

	return m.createCompletion(ctx, prompt, maxTokens)
}

func (m *Client) createCompletion(ctx context.Context, prompt string, maxTokens int) (string, error) {
	if m.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.timeout)
		defer cancel()
	}

	if isChatModel(m.model) {
		resp, err := m.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       m.model,
			MaxTokens:   maxTokens,
			Temperature: m.temperature,
			TopP:        m.topP,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "",
				},
			},
		})
		if err != nil {
			return "", err
		}
		return resp.Choices[0].Message.Content, nil
	}

	resp, err := m.client.CreateCompletion(ctx, openai.CompletionRequest{
		Model:       m.model,
		MaxTokens:   maxTokens,
		Temperature: m.temperature,
		TopP:        m.topP,
		Prompt:      prompt,
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Text, nil
}

func isChatModel(model string) bool {
	return strings.HasPrefix(model, "gpt-")
}
