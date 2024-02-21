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
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/modernice/dragoman"
	"github.com/modernice/dragoman/openai"
)

var options struct {
	SourcePath string   `arg:"source" name:"source" optional:"" help:"Source file" type:"path" env:"DRAGOMAN_SOURCE"`
	SourceLang string   `name:"from" short:"f" help:"Source language" env:"DRAGOMAN_SOURCE_LANG" default:"auto"`
	TargetLang string   `name:"to" short:"t" help:"Target language" env:"DRAGOMAN_TARGET_LANG" default:"English"`
	Preserve   []string `short:"p" help:"Preserve the specified terms/words" env:"DRAGOMAN_PRESERVE"`
	Rules      []string `name:"rule" short:"r" help:"Additional rules for the prompt" env:"DRAGOMAN_RULES"`
	Out        string   `short:"o" help:"Output file" type:"path" env:"DRAGOMAN_OUT"`
	Update     bool     `short:"u" help:"Only translate missing fields in output file (requires JSON files)" env:"DRAGOMAN_UPDATE"`
	Dry        bool     `help:"Write the result to stdout" env:"DRAGOMAN_DRY_RUN"`

	OpenAIKey            string  `name:"openai-key" help:"OpenAI API key" env:"OPENAI_KEY"`
	OpenAIModel          string  `name:"openai-model" help:"OpenAI model" env:"OPENAI_MODEL" default:"gpt-3.5-turbo"`
	OpenAITemperature    float32 `name:"temperature" help:"OpenAI temperature" env:"OPENAI_TEMPERATURE" default:"0.3"`
	OpenAITopP           float32 `name:"top-p" help:"OpenAI top_p" env:"OPENAI_TOP_P" default:"0.3"`
	OpenAIResponseFormat string  `name:"format" help:"OpenAI response format ('text' or 'json_object')" env:"OPENAI_RESPONSE_FORMAT" default:"text"`

	Timeout time.Duration `short:"T" help:"Timeout for API requests" env:"DRAGOMAN_TIMEOUT" default:"3m"`
	Verbose bool          `short:"v" help:"Verbose output"`
	Stream  bool          `short:"s" help:"Stream output to stdout"`
}

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

// Run initializes and starts the application, handling command-line parsing,
// signal interrupts, file input/output operations, and invoking the translation
// process using AI language models. It manages errors gracefully, provides
// feedback to the user, and ensures proper resource cleanup.
func (app *App) Run() {
	if options.Update && options.Out == "" {
		app.kong.Fatalf("you must provide the <out> file when using --update")
	}

	if options.Out == "" {
		options.Dry = true
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

	model := openai.New(options.OpenAIKey, opts...)
	translator := dragoman.New(model)

	var (
		source []byte
		err    error
	)
	if options.SourcePath == "" {
		source, err = readAll(os.Stdin)
		if errors.Is(err, errEmptyStdin) {
			app.kong.Fatalf("you must either provide the <source> file or provide the source text via stdin")
		} else {
			app.kong.FatalIfErrorf(err, "failed to read source from stdin")
		}
	} else {
		source, err = os.ReadFile(options.SourcePath)
		app.kong.FatalIfErrorf(err, "failed to read source file %q", options.SourcePath)
	}

	var (
		sourceMap      map[string]any
		originalOutMap map[string]any
	)
	if options.Update {
		err = json.Unmarshal(source, &sourceMap)
		app.kong.FatalIfErrorf(err, "failed to unmarshal source as JSON")

		outFile, err := os.ReadFile(options.Out)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			app.kong.FatalIfErrorf(err, "failed to read target file %q", options.Out)
		} else if err == nil {
			err = json.Unmarshal(outFile, &originalOutMap)
			app.kong.FatalIfErrorf(err, "failed to unmarshal target file %q", options.Out)
		} else {
			originalOutMap = map[string]any{}
		}

		paths, err := dragoman.JSONDiff(sourceMap, originalOutMap)
		app.kong.FatalIfErrorf(err, "failed to diff source and target")

		if len(paths) == 0 {
			if options.Verbose {
				fmt.Fprintf(os.Stderr, "No fields missing in output file %q.\n", options.Out)
			}
			return
		}

		sourceMap, err := dragoman.JSONExtract(source, paths)
		if err != nil {
			app.kong.FatalIfErrorf(err, "failed to extract missing fields from source")
		}

		if source, err = json.Marshal(sourceMap); err != nil {
			app.kong.FatalIfErrorf(err, "failed to marshal source map")
		}
	}

	if options.SourceLang == "auto" {
		options.SourceLang = ""
	}

	result, err := translator.Translate(
		ctx,
		string(source),
		dragoman.Source(options.SourceLang),
		dragoman.Target(options.TargetLang),
		dragoman.Preserve(options.Preserve...),
		dragoman.Rules(options.Rules...),
	)
	app.kong.FatalIfErrorf(err)

	if options.Dry {
		fmt.Fprintf(os.Stdout, "%s\n", result)
		return
	}

	if options.Update {
		var resultMap map[string]any
		if err := json.Unmarshal([]byte(result), &resultMap); err != nil {
			app.kong.FatalIfErrorf(err, "failed to unmarshal result as JSON")
		}
		dragoman.JSONMerge(originalOutMap, resultMap)

		marshaled, err := json.Marshal(originalOutMap)
		if err != nil {
			app.kong.FatalIfErrorf(err, "failed to marshal result map")
		}
		result = string(marshaled)
	}

	f, err := os.Create(options.Out)
	if err != nil {
		app.kong.FatalIfErrorf(err, "failed to create output file %q", options.Out)
		return
	}
	defer f.Close()

	if _, err = fmt.Fprint(f, result); err != nil {
		app.kong.FatalIfErrorf(err, "failed to write to output file %q", options.Out)
		return
	}

	if err = f.Close(); err != nil {
		app.kong.FatalIfErrorf(err, "failed to close output file %q", options.Out)
		return
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
