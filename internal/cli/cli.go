package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	Out        string   `short:"o" help:"Output file" type:"path" env:"DRAGOMAN_OUT"`

	OpenAIKey         string  `name:"openai-key" help:"OpenAI API key" env:"OPENAI_KEY"`
	OpenAIModel       string  `name:"openai-model" help:"OpenAI model" env:"OPENAI_MODEL" default:"gpt-3.5-turbo"`
	OpenAITemperature float32 `name:"temperature" help:"OpenAI temperature" env:"OPENAI_TEMPERATURE" default:"0.3"`
	OpenAITopP        float32 `name:"top-p" help:"OpenAI top_p" env:"OPENAI_TOP_P" default:"0.3"`

	Timeout time.Duration `short:"T" help:"Timeout for API requests" env:"DRAGOMAN_TIMEOUT" default:"3m"`
	Verbose bool          `short:"v" help:"Verbose output"`
	Stream  bool          `short:"s" help:"Stream output to stdout"`
}

// App represents a command-line application for translating structured text
// using AI language models. It reads the source text either from a file
// specified by the user or from the standard input. The translated text is then
// printed to the standard output. The application uses the OpenAI API for
// translation and supports various configuration options, such as specifying
// the OpenAI API key and model.
type App struct {
	version string
	kong    *kong.Context
}

// New creates a new instance of the Dragoman application. It takes a version
// string as input and returns a pointer to the created App object. The App
// object represents a command-line application for translating structured text
// using AI language models.
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

// Run runs the Dragoman application. It translates structured text using an AI
// language model. It reads the source text either from a specified file or from
// stdin, translates it using the OpenAI language model, and prints the
// translated result to stdout. The application can be interrupted by an
// interrupt signal (SIGINT) or a termination signal (SIGTERM).
func (app *App) Run() {
	fmt.Printf("dragoman %s\n", app.version)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	opts := []openai.Option{
		openai.Model(options.OpenAIModel),
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

	if options.SourceLang == "auto" {
		options.SourceLang = ""
	}

	result, err := translator.Translate(
		ctx,
		string(source),
		dragoman.Source(options.SourceLang),
		dragoman.Target(options.TargetLang),
		dragoman.Preserve(options.Preserve...),
	)
	app.kong.FatalIfErrorf(err)

	if options.Out == "" {
		fmt.Fprintf(os.Stdout, "%s\n", result)
		return
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
