package openai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

const (
	// DefaultModel is the default language model used for generating text when no
	// specific model is set during the client creation.
	DefaultModel = openai.GPT3Dot5Turbo

	// DefaultTemperature is the default value for the temperature parameter in the
	// AI model. It affects the randomness of the model's output.
	DefaultTemperature = 0.3

	// DefaultTopP is the default value for the "Top P" parameter used in OpenAI's
	// language models.
	DefaultTopP = 0.3

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
	verbose     bool
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

// Verbose sets the verbosity level of the Client instance. If set to true,
// debug logs will be printed during API requests.
func Verbose(verbose bool) Option {
	return func(m *Client) {
		m.verbose = verbose
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

	m.debug("Model: %s", m.model)
	m.debug("Temperature: %f", m.temperature)
	m.debug("TopP: %f", m.topP)
	m.debug("Max tokens: %d", m.maxTokens)

	return &m
}

// Chat is a method of the Client type that generates a text completion based on
// the provided prompt. The generated text completion is returned as a string.
func (m *Client) Chat(ctx context.Context, prompt string) (string, error) {
	resp, err := m.createCompletion(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp), nil
}

func (m *Client) createCompletion(ctx context.Context, prompt string) (string, error) {
	if m.timeout > 0 {
		m.debug("Setting timeout to %s", m.timeout)

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.timeout)
		defer cancel()
	}

	if isChatModel(m.model) {
		m.debug("Creating chat completion with prompt:\n\n%s", prompt)

		msgs := []openai.ChatCompletionMessage{{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		}}

		promptTokens, err := ChatTokens(m.model, msgs)
		if err != nil {
			return "", err
		}

		// -1 because "This model's maximum context length is 8192 tokens. However, you requested 8192 tokens" ???
		maxTokens := m.maxTokens - promptTokens - 1

		resp, err := m.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       m.model,
			MaxTokens:   maxTokens,
			Temperature: m.temperature,
			TopP:        m.topP,
			Messages:    msgs,
		})
		if err != nil {
			return "", err
		}
		return resp.Choices[0].Message.Content, nil
	}

	m.debug("Creating completion with prompt:\n\n%s", prompt)

	promptTokens, err := PromptTokens(m.model, prompt)
	if err != nil {
		return "", fmt.Errorf("compute prompt tokens: %w", err)
	}

	// -1 because "This model's maximum context length is 8192 tokens. However, you requested 8192 tokens" ???
	maxTokens := m.maxTokens - promptTokens - 1

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

func (m *Client) debug(format string, args ...interface{}) {
	if m.verbose {
		log.Printf("[OpenAI] %s", fmt.Sprintf(format, args...))
	}
}

func isChatModel(model string) bool {
	return strings.HasPrefix(model, "gpt-")
}
