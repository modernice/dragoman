package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/bounoable/dragoman"
	"github.com/bounoable/dragoman/text"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// New returns the translator CLI, configured by opts.
func New(opts ...Option) *CLI {
	cli := &CLI{
		Command: cobra.Command{
			Use:   "translate",
			Short: "Translate structured documents",
		},
		examples:       make(map[string]map[string]string),
		translatorArgs: make(map[string]*string),
	}
	for _, opt := range opts {
		opt(cli)
	}
	cli.init()
	return cli
}

// Option is a CLI option.
type Option func(*CLI)

// CLI is the translator CLI.
type CLI struct {
	cobra.Command
	translators []Translator
	formats     []Format
	sources     []Source
	examples    map[string]map[string]string // map[FORMAT]map[SOURCE]EXAMPLE

	// flags
	translatorArgs map[string]*string
	sourceLang     string
	targetLang     string
	out            string
	preserve       string
	parallel       int

	translator *dragoman.Translator
}

// Translator is a translation service configuration.
type Translator struct {
	// Name will be used as the flag name for the translation service.
	// For example the name "deepl" creates the CLI flag "--deepl".
	Name string
	// Description is the usage message for the CLI flag.
	Description string
	// New accepts the flag value (authentication key) and creates the translation service.
	New func(string) (dragoman.Service, error)
}

// Format is a format cofiguration.
type Format struct {
	// Name will be used as the CLI command name.
	Name string
	// Short is the short CLI description.
	Short string
	// Flags is an optional function that accepts the flag set for the command.
	// Used to add additional flags specific to the format.
	Flags func(*pflag.FlagSet)
	// Ranger creates the text ranger for the format.
	Ranger func() (text.Ranger, error)
}

// Source is a file source configuration.
type Source struct {
	// Name will be used as the CLI command name.
	Name string
	// Reader creates the io.Reader from the first CLI argument.
	// If the reader is also an io.Closer, it will be automatically closed after execution.
	Reader func(string) (io.Reader, error)
}

// WithTranslator adds translation services to the CLI.
func WithTranslator(trans ...Translator) Option {
	return func(cli *CLI) {
		cli.translators = append(cli.translators, trans...)
	}
}

// WithFormat adds formats to the CLI.
func WithFormat(formats ...Format) Option {
	return func(cli *CLI) {
		cli.formats = append(cli.formats, formats...)
	}
}

// WithSource adds sources to the CLI.
func WithSource(sources ...Source) Option {
	return func(cli *CLI) {
		cli.sources = append(cli.sources, sources...)
	}
}

// WithExample adds an example for the given format <-> source combination.
func WithExample(format, source, example string) Option {
	return func(cli *CLI) {
		examples, ok := cli.examples[format]
		if !ok {
			examples = map[string]string{}
		}
		examples[source] = example
		cli.examples[format] = examples
	}
}

func (cli *CLI) init() {
	cli.initTranslator()

	for _, format := range cli.formats {
		cmd := &cobra.Command{
			Use:   format.Name,
			Short: format.Short,
		}

		cmd.PersistentFlags().StringVar(&cli.sourceLang, "from", "en", "Source language")
		cmd.PersistentFlags().StringVar(&cli.targetLang, "into", "en", "Target language")
		cmd.PersistentFlags().StringVar(&cli.preserve, "preserve", "", "Prevent translation of substrings (regular expression)")
		cmd.PersistentFlags().StringVarP(&cli.out, "out", "o", "", "Write the result to the specified filepath")
		cmd.PersistentFlags().IntVarP(&cli.parallel, "parallel", "p", 1, "Max concurrent translation requests")

		if format.Flags != nil {
			format.Flags(cmd.PersistentFlags())
		}

		for _, source := range cli.sources {
			cli.sourceCommand(cmd, source, format)
		}

		cli.AddCommand(cmd)
	}
}

func (cli *CLI) initTranslator() {
	for _, translator := range cli.translators {
		var arg string
		cli.PersistentFlags().StringVar(&arg, translator.Name, "", translator.Description)
		cli.translatorArgs[translator.Name] = &arg
	}

	cli.PersistentPreRunE = func(*cobra.Command, []string) error {
		svc, err := cli.newService()
		if err != nil {
			return fmt.Errorf("make translation service: %w", err)
		}
		cli.translator = dragoman.New(svc)
		return nil
	}
}

func (cli *CLI) newService() (dragoman.Service, error) {
	for name, arg := range cli.translatorArgs {
		if *arg == "" {
			continue
		}

		trans, ok := cli.translatorConfig(name)
		if !ok {
			continue
		}

		svc, err := trans.New(*arg)
		if err != nil {
			return svc, fmt.Errorf("Translator.New(%v) failed: %w", *arg, err)
		}

		return svc, nil
	}
	return nil, cli.missingServiceError()
}

func (cli *CLI) translatorConfig(name string) (Translator, bool) {
	for _, trans := range cli.translators {
		if trans.Name == name {
			return trans, true
		}
	}
	return Translator{}, false
}

func (cli *CLI) missingServiceError() error {
	var b strings.Builder
	b.WriteString("Missing translation service. Select one with the one of the following options:\n")
	for _, translator := range cli.translators {
		b.WriteString(fmt.Sprintf("      --%s string\n", translator.Name))
	}
	return humanError{
		err:     errors.New("missing translation service"),
		message: b.String(),
	}
}

func (cli *CLI) sourceCommand(formatCmd *cobra.Command, source Source, format Format) {
	cmd := &cobra.Command{
		Use:     source.Name,
		Short:   fmt.Sprintf("%s (%s)", formatCmd.Short, source.Name),
		Example: cli.example(format.Name, source.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := source.Reader(args[0])
			if err != nil {
				return fmt.Errorf("Source.Reader(%v) failed: %w", args[0], err)
			}
			if c, ok := r.(io.Closer); ok {
				defer c.Close()
			}

			opts := []dragoman.TranslateOption{
				dragoman.Parallel(cli.parallel),
			}

			if cli.preserve != "" {
				expr, err := regexp.Compile(cli.preserve)
				if err != nil {
					return fmt.Errorf("compile regexp (%v): %w", cli.preserve, err)
				}
				opts = append(opts, dragoman.Preserve(expr))
			}

			ranger, err := format.Ranger()
			if err != nil {
				return fmt.Errorf("make ranger: %w", err)
			}

			res, err := cli.translator.Translate(
				cmd.Context(),
				r,
				cli.sourceLang,
				cli.targetLang,
				ranger,
				opts...,
			)
			if err != nil {
				return err
			}

			out := os.Stdout
			var f *os.File

			if cli.out != "" {
				if f, err = os.Create(cli.out); err != nil {
					return fmt.Errorf("create outfile (%v): %w", cli.out, err)
				}
				out = f
			}

			if _, err = fmt.Fprint(out, string(res)); err != nil {
				return fmt.Errorf("write result: %w", err)
			}

			if f != nil {
				if err = f.Close(); err != nil {
					return fmt.Errorf("close outfile: %w", err)
				}
			}

			return nil
		},
	}

	formatCmd.AddCommand(cmd)
}

func (cli *CLI) example(format, source string) string {
	examples, ok := cli.examples[format]
	if !ok {
		return cli.defaultExample(format, source)
	}
	content, ok := examples[source]
	if !ok {
		return cli.defaultExample(format, source)
	}
	return fmt.Sprintf("translate %s %s %s --from en --into de --deepl $DEEPL_AUTH_KEY", format, source, content)
}

func (cli *CLI) defaultExample(format, source string) string {
	return fmt.Sprintf("translate %s %s CONTENT -o out.%s --from en --into de --deepl $DEEPL_AUTH_LEY", format, source, format)
}

type humanError struct {
	err     error
	message string
}

func (err humanError) Error() string {
	return err.err.Error()
}

func (err humanError) HumanError() string {
	return err.message
}
