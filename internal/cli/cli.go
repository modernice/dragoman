package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/modernice/dragoman"
	"github.com/modernice/dragoman/internal/chunks"
	"github.com/modernice/dragoman/openai"
)

type cliOptions struct {
	Translate struct {
		SourcePath   string   `arg:"source" name:"source" optional:"" help:"Source file" type:"path" env:"DRAGOMAN_SOURCE"`
		SourceLang   string   `name:"from" short:"f" help:"Source language" env:"DRAGOMAN_SOURCE_LANG" default:"auto"`
		TargetLang   string   `name:"to" short:"t" help:"Target language" env:"DRAGOMAN_TARGET_LANG" default:"English"`
		Preserve     []string `short:"p" help:"Preserve the specified terms/words" env:"DRAGOMAN_PRESERVE"`
		Instructions []string `name:"instruct" short:"i" help:"Additional instructions for the prompt" env:"DRAGOMAN_INSTRUCT"`
		Out          string   `short:"o" help:"Output file" type:"path" env:"DRAGOMAN_OUT"`
		Update       bool     `short:"u" help:"Only translate missing fields in output file (requires JSON files)" env:"DRAGOMAN_UPDATE"`
		SplitChunks  []string `name:"split-chunks" help:"Chunk source file at lines that start with one of the provided prefixes" env:"DRAGOMAN_SPLIT_CHUNKS"`
		Dry          bool     `help:"Write the result to stdout" env:"DRAGOMAN_DRY_RUN"`
	} `cmd:"translate" default:"withargs"`

	Improve struct {
		SourcePath   string             `arg:"source" name:"source" optional:"" help:"Source file" type:"path" env:"DRAGOMAN_SOURCE"`
		Out          string             `short:"o" help:"Output file" type:"path" env:"DRAGOMAN_OUT"`
		SplitChunks  []string           `name:"split-chunks" help:"Chunk source file at lines that start with one of the provided prefixes" env:"DRAGOMAN_SPLIT_CHUNKS"`
		Formality    dragoman.Formality `name:"formality" help:"Formality of the text" env:"DRAGOMAN_FORMALITY"`
		Instructions []string           `name:"instruct" short:"i" help:"Additional instructions for the prompt" env:"DRAGOMAN_INSTRUCT"`
		Keywords     []string           `name:"keywords" help:"Keywords to optimize for" env:"DRAGOMAN_KEYWORDS"`
		Language     string             `name:"language" short:"l" help:"Write the text in the given language" env:"DRAGOMAN_LANGUAGE"`
		Dry          bool               `help:"Write the result to stdout" env:"DRAGOMAN_DRY_RUN"`
	} `cmd:"improve"`

	OpenAIKey            string  `name:"openai-key" help:"OpenAI API key" env:"OPENAI_KEY"`
	OpenAIModel          string  `name:"openai-model" help:"OpenAI model" env:"OPENAI_MODEL" default:"gpt-3.5-turbo"`
	OpenAITemperature    float32 `name:"temperature" help:"OpenAI temperature" env:"OPENAI_TEMPERATURE" default:"0.3"`
	OpenAITopP           float32 `name:"top-p" help:"OpenAI top_p" env:"OPENAI_TOP_P" default:"0.3"`
	OpenAIResponseFormat string  `name:"format" help:"OpenAI response format ('text' or 'json_object')" env:"OPENAI_RESPONSE_FORMAT" default:"text"`
	OpenAIChunkTimeout   string  `name:"chunk-timeout" help:"Timeout for each token chunk" env:"OPENAI_CHUNK_TIMEOUT"`

	Timeout time.Duration `short:"T" help:"Timeout for API requests" env:"DRAGOMAN_TIMEOUT" default:"3m"`
	Verbose bool          `short:"v" help:"Verbose output"`
	Stream  bool          `short:"s" help:"Stream output to stdout"`
}

var options cliOptions

// App coordinates the translation of structured text using AI language models.
// It sets up a command-line interface with various options to specify source
// and target languages, preserve certain terms, apply translation rules, and
// handle input/output configurations. App encapsulates the logic for reading
// source content, invoking translation services with the specified parameters,
// and writing the translated result to either a file or standard output,
// respecting user-defined timeouts and verbosity settings. It also gracefully
// handles termination signals to ensure proper cleanup during unexpected exits.
type App struct {
	version string
	kong    *kong.Context
}

// New creates a new instance of App with the provided version and sets up its
// command-line interface context. It returns a pointer to the created App.
func New(version string) *App {
	app := App{version: version}
	app.kong = kong.Parse(
		&options,
		kong.Name("dragoman"),
		kong.Description("Dragoman is a translator for structured text, powered by AI language models."),
		kong.Help(func(opts kong.HelpOptions, ctx *kong.Context) error {
			ctx.Stdout.Write([]byte(fmt.Sprintf("dragoman %s\n", version)))
			return kong.DefaultHelpPrinter(opts, ctx)
		}),
	)
	return &app
}

// Run starts the application based on the command-line arguments provided. It
// determines the operation mode (translate or improve), executes the
// corresponding function, and handles default behavior if no specific command
// is recognized.
func (app *App) Run() {
	switch app.kong.Command() {
	case "translate <source>":
		app.translate()
	case "improve <source>":
		app.improve()
	default:
		app.kong.PrintUsage(false)
	}
}

func (app *App) translate() {
	if options.Translate.Update && options.Translate.Out == "" {
		app.kong.Fatalf("you must provide the <out> file when using --update")
	}

	if options.Translate.Out == "" {
		options.Translate.Dry = true
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	opts := []openai.Option{
		openai.Model(options.OpenAIModel),
		openai.ResponseFormat(options.OpenAIResponseFormat),
		openai.Temperature(options.OpenAITemperature),
		openai.TopP(options.OpenAITopP),
		openai.Timeout(options.Timeout),
		openai.Verbose(options.Verbose),
	}

	if options.Stream {
		opts = append(opts, openai.Stream(os.Stdout))
	}

	if options.OpenAIChunkTimeout != "" {
		chunkTimeout, err := time.ParseDuration(options.OpenAIChunkTimeout)
		if err != nil {
			app.kong.Fatalf("invalid chunk timeout: %v", err)
		}
		opts = append(opts, openai.ChunkTimeout(chunkTimeout))
	}

	model := openai.New(options.OpenAIKey, opts...)
	translator := dragoman.NewTranslator(model)

	var (
		source []byte
		err    error
	)
	if options.Translate.SourcePath == "" {
		source, err = readAll(os.Stdin)
		if errors.Is(err, errEmptyStdin) {
			app.kong.Fatalf("you must either provide the <source> file or provide the source text via stdin")
		} else {
			app.kong.FatalIfErrorf(err, "failed to read source from stdin")
		}
	} else {
		source, err = os.ReadFile(options.Translate.SourcePath)
		app.kong.FatalIfErrorf(err, "failed to read source file %q", options.Translate.SourcePath)
	}

	var (
		sourceMap      map[string]any
		originalOutMap map[string]any
	)
	if options.Translate.Update {
		err = json.Unmarshal(source, &sourceMap)
		app.kong.FatalIfErrorf(err, "failed to unmarshal source as JSON")

		outFile, err := os.ReadFile(options.Translate.Out)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			app.kong.FatalIfErrorf(err, "failed to read target file %q", options.Translate.Out)
		} else if err == nil {
			err = json.Unmarshal(outFile, &originalOutMap)
			app.kong.FatalIfErrorf(err, "failed to unmarshal target file %q", options.Translate.Out)
		} else {
			originalOutMap = map[string]any{}
		}

		paths, err := dragoman.JSONDiff(sourceMap, originalOutMap)
		app.kong.FatalIfErrorf(err, "failed to diff source and target")

		if len(paths) == 0 {
			if options.Verbose {
				fmt.Fprintf(os.Stderr, "No fields missing in output file %q.\n", options.Translate.Out)
			}
			return
		}

		sourceMap, err := dragoman.JSONExtract(source, paths)
		if err != nil {
			app.kong.FatalIfErrorf(err, "failed to extract missing fields from source")
		}

		if source, err = jsonMarshal(sourceMap); err != nil {
			app.kong.FatalIfErrorf(err, "failed to marshal source map")
		}
	}

	if options.Translate.SourceLang == "auto" {
		options.Translate.SourceLang = ""
	}

	chunks := getChunks(string(source), options.Translate.SplitChunks, options.Verbose)

	var results []string
	for _, chunk := range chunks {
		chunkResult, err := translator.Translate(
			ctx,
			dragoman.TranslateParams{
				Document:     chunk,
				Source:       options.Translate.SourceLang,
				Target:       options.Translate.TargetLang,
				Preserve:     options.Translate.Preserve,
				Instructions: options.Translate.Instructions,
			},
		)
		app.kong.FatalIfErrorf(err)
		results = append(results, chunkResult)
	}

	result := strings.Join(results, "\n\n")

	if options.Translate.Dry {
		fmt.Fprintf(os.Stdout, "%s\n", result)
		return
	}

	if options.Translate.Update {
		var resultMap map[string]any
		if err := json.Unmarshal([]byte(result), &resultMap); err != nil {
			app.kong.FatalIfErrorf(err, "failed to unmarshal result as JSON")
		}
		dragoman.JSONMerge(originalOutMap, resultMap)

		marshaled, err := jsonMarshal(originalOutMap)
		if err != nil {
			app.kong.FatalIfErrorf(err, "failed to marshal result map")
		}
		result = string(marshaled)
	}

	f, err := os.Create(options.Translate.Out)
	if err != nil {
		app.kong.FatalIfErrorf(err, "failed to create output file %q", options.Translate.Out)
		return
	}
	defer f.Close()

	if _, err = fmt.Fprint(f, result); err != nil {
		app.kong.FatalIfErrorf(err, "failed to write to output file %q", options.Translate.Out)
		return
	}

	if err = f.Close(); err != nil {
		app.kong.FatalIfErrorf(err, "failed to close output file %q", options.Translate.Out)
		return
	}
}

func (app *App) improve() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	opts := []openai.Option{
		openai.Model(options.OpenAIModel),
		openai.ResponseFormat(options.OpenAIResponseFormat),
		openai.Temperature(options.OpenAITemperature),
		openai.TopP(options.OpenAITopP),
		openai.Timeout(options.Timeout),
		openai.Verbose(options.Verbose),
	}

	if options.Stream {
		opts = append(opts, openai.Stream(os.Stdout))
	}

	model := openai.New(options.OpenAIKey, opts...)
	improver := dragoman.NewImprover(model)

	var (
		source []byte
		err    error
	)
	if options.Improve.SourcePath == "" {
		source, err = readAll(os.Stdin)
		if errors.Is(err, errEmptyStdin) {
			app.kong.Fatalf("you must either provide the <source> file or provide the source text via stdin")
		} else {
			app.kong.FatalIfErrorf(err, "failed to read source from stdin")
		}
	} else {
		source, err = os.ReadFile(options.Improve.SourcePath)
		app.kong.FatalIfErrorf(err, "failed to read source file %q", options.Improve.SourcePath)
	}

	result, err := improver.Improve(ctx, dragoman.ImproveParams{
		Document:     string(source),
		SplitChunks:  options.Improve.SplitChunks,
		Formality:    options.Improve.Formality,
		Instructions: options.Improve.Instructions,
		Keywords:     options.Improve.Keywords,
		Language:     options.Improve.Language,
	})
	if err != nil {
		app.kong.FatalIfErrorf(err, "failed to improve document")
	}

	if options.Improve.Dry {
		fmt.Fprintf(os.Stdout, "%s\n", result)
		return
	}

	f, err := os.Create(options.Improve.Out)
	if err != nil {
		app.kong.FatalIfErrorf(err, "failed to create output file %q", options.Improve.Out)
		return
	}
	defer f.Close()

	if _, err = fmt.Fprint(f, result); err != nil {
		app.kong.FatalIfErrorf(err, "failed to write to output file %q", options.Improve.Out)
		return
	}

	if err = f.Close(); err != nil {
		app.kong.FatalIfErrorf(err, "failed to close output file %q", options.Improve.Out)
		return
	}

	if options.Improve.Dry {
		fmt.Fprintf(os.Stdout, "%s\n", result)
	}

	if options.Improve.Out != "" {
		if err := os.WriteFile(options.Improve.Out, []byte(result), 0644); err != nil {
			app.kong.FatalIfErrorf(err, "failed to write output to %q", options.Improve.Out)
		}
	}
}

var errEmptyStdin = errors.New("stdin is empty")

func readAll(r io.Reader) (out []byte, err error) {
	defer func() { out = bytes.TrimSpace(out) }()

	var buf bytes.Buffer
	var checked bool

	chunk := make([]byte, 64)
	for {
		var (
			n   int
			err error
		)

		if !checked {
			timer := time.NewTimer(time.Second)

			var read = make(chan struct{})

			go func() {
				defer close(read)
				n, err = r.Read(chunk)
			}()

			select {
			case <-timer.C:
				timer.Stop()
				return buf.Bytes(), errEmptyStdin
			case <-read:
				timer.Stop()
				checked = true
			}
		} else {
			n, err = r.Read(chunk)
		}

		buf.Write(chunk[:n])

		if errors.Is(err, io.EOF) {
			return buf.Bytes(), nil
		}

		if err != nil {
			return buf.Bytes(), err
		}
	}
}

func jsonMarshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(v)
	return buf.Bytes(), err
}

func getChunks(source string, splitChunks []string, verbose bool) []string {
	if len(splitChunks) == 0 {
		return []string{string(source)}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Splitting source file at lines with prefixes: %v\n", splitChunks)
	}

	return chunks.Chunks(string(source), splitChunks)
}
