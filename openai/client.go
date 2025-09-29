package openai

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/responses"
)

const (
	// DefaultModel is the default language model used for generating text when no
	// specific model is set during the client creation.
	DefaultModel = "gpt-5-mini"

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

// Client is a configurable interface to the OpenAI API. It allows for the
// generation of text completions using various models, with adjustable
// parameters for token count and topP. A specified timeout can be
// set for API requests.
type Client struct {
	model            string
	responseFormat   string
	reasoningEffort  string
	reasoningSummary openai.ReasoningSummary
	maxTokens        int
	topP             float32
	timeout          time.Duration
	chunkTimeout     time.Duration
	verbose          bool
	stream           io.Writer
	client           openai.Client
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
func ResponseFormat(format string) Option {
	return func(m *Client) {
		m.responseFormat = format
	}
}

// ReasoningEffort configures the reasoning effort for Responses API ("low", "medium", or "high").
func ReasoningEffort(effort string) Option {
	return func(m *Client) {
		m.reasoningEffort = strings.ToLower(strings.TrimSpace(effort))
	}
}

// ReasoningSummary configures the reasoning summary for the Responses API by
// setting the [openai.ReasoningSummary] on a [Client].
func ReasoningSummary(summary openai.ReasoningSummary) Option {
	return func(m *Client) {
		m.reasoningSummary = summary
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
		topP:         DefaultTopP,
		timeout:      DefaultTimeout,
		chunkTimeout: DefaultChunkTimeout,
		client:       openai.NewClient(option.WithAPIKey(apiToken)),
	}
	for _, opt := range opts {
		opt(&c)
	}

	if c.model == "" {
		c.model = DefaultModel
	}

	c.debug("Model: %s", c.model)
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

	c.debug("Creating response with prompt:\n\n%s", prompt)

	// Use typed Responses API
	reqOpts := []option.RequestOption{}
	if c.maxTokens > 0 {
		reqOpts = append(reqOpts, option.WithJSONSet("max_output_tokens", c.maxTokens))
	}
	if c.responseFormat == "json_object" {
		reqOpts = append(reqOpts, option.WithJSONSet("response_format", map[string]any{"type": "json_object"}))
	}
	if c.reasoningEffort != "" {
		reqOpts = append(reqOpts, option.WithJSONSet("reasoning.effort", c.reasoningEffort))
	}
	if c.reasoningSummary != "" {
		reqOpts = append(reqOpts, option.WithJSONSet("reasoning.summary", c.reasoningSummary))
	}
	// Only include top_p for models that support it
	if c.topP > 0 && supportsTopP(c.model) {
		reqOpts = append(reqOpts, option.WithJSONSet("top_p", c.topP))
	}

	var responseText string

	if c.stream != nil {
		// Use streaming API
		stream := c.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
			Model: c.model,
			Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(prompt)},
		}, reqOpts...)

		// Accumulate the full response while streaming
		var fullText strings.Builder
		for stream.Next() {
			event := stream.Current()

			var text string
			switch event.Type {
			case "response.reasoning_summary_text.delta":
				if c.reasoningSummary != "" {
					text = event.Delta
				}
			case "response.reasoning_summary_text.done":
				if c.reasoningSummary != "" {
					text = event.Text
				}
			case "response.output_text.delta":
				text = event.Delta
			case "response.output_text.done":
				text = event.Text
			}

			if text != "" {
				fullText.WriteString(text)

				if _, err := c.stream.Write([]byte(text)); err != nil {
					return "", fmt.Errorf("failed to write to stream: %w", err)
				}
			}
		}

		if err := stream.Err(); err != nil {
			return "", err
		}

		responseText = fullText.String()
	} else {
		// Use synchronous API
		resp, err := c.client.Responses.New(ctx, responses.ResponseNewParams{
			Model: c.model,
			Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(prompt)},
		}, reqOpts...)
		if err != nil {
			return "", err
		}

		responseText = resp.OutputText()
	}

	if text := strings.TrimSpace(responseText); text != "" {
		return text, nil
	}

	return "", fmt.Errorf("empty response")
}

func (m *Client) debug(format string, args ...interface{}) {
	if m.verbose {
		log.Printf("[OpenAI] %s", fmt.Sprintf(format, args...))
	}
}

// isGPT5Model returns true if the model string refers to a GPT-5 family model.
func isGPT5Model(model string) bool {
	return strings.HasPrefix(model, "gpt-5")
}

// supportsTopP returns whether top_p should be sent for the given model.
// GPT-5 family models do not support top_p; omit for those.
func supportsTopP(model string) bool {
	return !isGPT5Model(model)
}
