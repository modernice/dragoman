package openai

import (
	"context"
	"fmt"
	"io"
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
	DefaultTimeout = 3 * time.Minute

	// DefaultChunkTimeout specifies the default duration for waiting on a chunk of
	// data during streaming operations before timing out. This value can be
	// adjusted to control how long the system will wait for a chunk before
	// considering the operation timed out.
	DefaultChunkTimeout = 5 * time.Second
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
	model          string
	responseFormat openai.ChatCompletionResponseFormatType
	maxTokens      int
	temperature    float32
	topP           float32
	timeout        time.Duration
	chunkTimeout   time.Duration
	verbose        bool
	stream         io.Writer
	client         *openai.Client
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

// ResponseFormat configures the format of the response received from the OpenAI
// API when generating text completions. It specifies how the response should be
// structured, which can be either plain text or a structured format that
// includes additional metadata. This option is passed to a Client instance
// during its creation and influences how the Client processes and returns
// generated content. The function accepts format types that can be either a
// string or an openai.ChatCompletionResponseFormatType.
func ResponseFormat[Format string | openai.ChatCompletionResponseFormatType](format Format) Option {
	return func(m *Client) {
		m.responseFormat = openai.ChatCompletionResponseFormatType(format)
	}
}

// MaxTokens configures the maximum number of tokens that the Client can use for
// generating text completions. It accepts an integer value and returns an
// [Option] to modify a [Client] instance.
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

// ChunkTimeout sets the maximum duration a Client should wait for a chunk of
// data during streaming operations before timing out. This is configured as an
// Option that modifies the chunkTimeout field of a Client instance.
func ChunkTimeout(timeout time.Duration) Option {
	return func(m *Client) {
		m.chunkTimeout = timeout
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

// Stream is an option function that sets the writer to which the generated text
// completions will be streamed. This allows for real-time processing and
// display of the generated text.
func Stream(stream io.Writer) Option {
	return func(m *Client) {
		m.stream = stream
	}
}

// New creates a new Client instance with the specified API token and optional
// configuration options. The Client allows for the generation of text
// completions using various models, with adjustable parameters for token count,
// temperature, and topP. The default values for these parameters are used if
// not explicitly set. The Client also supports setting a timeout duration for
// API requests.
func New(apiToken string, opts ...Option) *Client {
	c := Client{
		temperature:  DefaultTemperature,
		topP:         DefaultTopP,
		timeout:      DefaultTimeout,
		chunkTimeout: DefaultChunkTimeout,
		client:       openai.NewClient(apiToken),
	}
	for _, opt := range opts {
		opt(&c)
	}

	if c.model == "" {
		c.model = DefaultModel
	}

	var ok bool
	if c.maxTokens, ok = modelTokens[c.model]; !ok {
		c.maxTokens = modelTokens["default"]
	}

	c.debug("Model: %s", c.model)
	c.debug("Temperature: %f", c.temperature)
	c.debug("TopP: %f", c.topP)

	if c.maxTokens > 0 {
		c.debug("Max tokens: %d", c.maxTokens)
	}

	return &c
}

// Chat is a method of the Client type that generates a text completion based on
// the provided prompt. The generated text completion is returned as a string.
func (c *Client) Chat(ctx context.Context, prompt string) (string, error) {
	resp, err := c.createCompletion(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp), nil
}

func (c *Client) createCompletion(ctx context.Context, prompt string) (string, error) {
	if c.timeout > 0 {
		c.debug("Setting timeout to %s", c.timeout)

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	if isChatModel(c.model) {
		c.debug("Creating chat completion with prompt:\n\n%s", prompt)

		msgs := []openai.ChatCompletionMessage{{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		}}

		if c.responseFormat == "json_object" {
			msgs = append([]openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a translator for JSON files. You only translate text fields, preserving the JSON structure and keys.",
				},
			}, msgs...)
		}

		var responseFormat *openai.ChatCompletionResponseFormat
		if c.responseFormat != "" {
			responseFormat = &openai.ChatCompletionResponseFormat{Type: c.responseFormat}
		}

		stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:          c.model,
			MaxTokens:      c.maxTokens,
			Temperature:    c.temperature,
			TopP:           c.topP,
			Messages:       msgs,
			ResponseFormat: responseFormat,
		})
		if err != nil {
			return "", err
		}
		return streamReader(c, stream, c.chunkTimeout).read(ctx, func(stream *openai.ChatCompletionStream) (chunk, error) {
			resp, err := stream.Recv()
			if err != nil {
				return chunk{}, err
			}
			return chunk{
				text:         resp.Choices[0].Delta.Content,
				finishReason: string(resp.Choices[0].FinishReason),
			}, nil
		})
	}

	c.debug("Creating completion with prompt:\n\n%s", prompt)

	promptTokens, err := PromptTokens(c.model, prompt)
	if err != nil {
		return "", fmt.Errorf("compute prompt tokens: %w", err)
	}

	// -1 because "This model's maximum context length is 8192 tokens. However, you requested 8192 tokens" ???
	maxTokens := c.maxTokens - promptTokens - 1

	stream, err := c.client.CreateCompletionStream(ctx, openai.CompletionRequest{
		Model:       c.model,
		MaxTokens:   maxTokens,
		Temperature: c.temperature,
		TopP:        c.topP,
		Prompt:      prompt,
	})
	if err != nil {
		return "", err
	}
	return streamReader(c, stream, c.chunkTimeout).read(ctx, func(stream *openai.CompletionStream) (chunk, error) {
		resp, err := stream.Recv()
		if err != nil {
			return chunk{}, err
		}
		return chunk{
			text:         resp.Choices[0].Text,
			finishReason: resp.Choices[0].FinishReason,
		}, nil
	})
}

type chunk struct {
	text         string
	finishReason string
}

func (m *Client) debug(format string, args ...interface{}) {
	if m.verbose {
		log.Printf("[OpenAI] %s", fmt.Sprintf(format, args...))
	}
}

func isChatModel(model string) bool {
	return strings.HasPrefix(model, "gpt-")
}

type chunkReader[Stream any] struct {
	client  *Client
	stream  Stream
	timeout time.Duration
}

func streamReader[Stream any](client *Client, stream Stream, timeout time.Duration) *chunkReader[Stream] {
	return &chunkReader[Stream]{
		client:  client,
		stream:  stream,
		timeout: timeout,
	}
}

func (r *chunkReader[Stream]) read(ctx context.Context, getChunk func(Stream) (chunk, error)) (string, error) {
	var text strings.Builder

	if r.client.stream != nil {
		fmt.Fprint(r.client.stream, "\n")
	}

	for {
		timeout := time.NewTimer(r.timeout)

		chunkC := make(chan chunk)
		errC := make(chan error)

		fail := func(err error) {
			select {
			case <-ctx.Done():
				return
			case errC <- err:
				return
			}
		}

		go func() {
			chunk, err := getChunk(r.stream)
			if err != nil {
				fail(err)
				return
			}
			select {
			case <-ctx.Done():
				return
			case chunkC <- chunk:
			}
		}()

		select {
		case <-ctx.Done():
			timeout.Stop()
			return text.String(), ctx.Err()
		case <-timeout.C:
			return text.String(), fmt.Errorf("token-chunk timeout")
		case err := <-errC:
			timeout.Stop()
			return text.String(), err
		case chunk := <-chunkC:
			timeout.Stop()
			text.WriteString(chunk.text)

			if chunk.text != "" && r.client.stream != nil {
				fmt.Fprint(r.client.stream, chunk.text)
			}

			if chunk.finishReason == string(openai.FinishReasonStop) {
				return text.String(), nil
			}

			if chunk.finishReason == string(openai.FinishReasonLength) {
				return text.String(), fmt.Errorf("max tokens exceeded")
			}
		}
	}
}
