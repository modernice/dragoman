package cli

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	translator "github.com/bounoable/dragoman"
	"github.com/bounoable/dragoman/format/json"
	"github.com/bounoable/dragoman/service/deepl"
	"github.com/spf13/cobra"
)

// New returns the translator CLI.
func New() *CLI {
	cli := &CLI{Command: cobra.Command{
		Use:   "translate",
		Short: "Translate structured texts.",
	}}
	cli.init()
	return cli
}

// CLI is the translator CLI.
type CLI struct {
	cobra.Command
	trans *translator.Translator
}

var (
	formats = [...]string{
		"json",
	}
	formatShorts = map[string]string{
		"json": "Translate JSON",
	}
)

func (cli *CLI) init() {
	for _, format := range formats[:] {
		cli.AddCommand(formatCommand(format, formatShorts[format]))
	}
}

var (
	deeplAuthKey string
)

var (
	sourceLang string
	targetLang string
	preserve   string
	parallel   int
	outfile    string
)

var (
	out = os.Stdout
)

var (
	sources = [...]string{
		"text",
		"file",
	}

	contents = map[string]string{
		"text": `'{"title": "Hello, {firstName}!"}'`,
		"file": `i18n/en.json`,
	}
)

func formatCommand(
	format string,
	short string,
) *cobra.Command {
	var file *os.File
	cmd := &cobra.Command{
		Use:   format,
		Short: short,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if outfile == "" {
				return nil
			}
			var err error
			if file, err = os.Create(outfile); err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			out = file
			return nil
		},
		PersistentPostRunE: func(*cobra.Command, []string) error {
			if file != nil {
				if err := file.Close(); err != nil {
					return fmt.Errorf("close file: %w", err)
				}
			}
			return nil
		},
	}

	for _, source := range sources[:] {
		cmd.AddCommand(sourceCommand(
			format,
			source,
			fmt.Sprintf("%s %s", short, source),
			contents[source],
		))
	}

	cmd.PersistentFlags().StringVar(&deeplAuthKey, "deepl", "", "DeepL authentication key")
	cmd.PersistentFlags().StringVar(&sourceLang, "from", "en", "Source language")
	cmd.PersistentFlags().StringVar(&targetLang, "into", "en", "Target language")
	cmd.PersistentFlags().StringVar(&preserve, "preserve", "", "Prevent translation of substrings (regular expression)")
	cmd.PersistentFlags().IntVarP(&parallel, "parallel", "p", 1, "Max allowed concurrent translation requests")
	cmd.PersistentFlags().StringVarP(&outfile, "out", "o", "", "Write the result to the specified filepath")

	cmd.MarkPersistentFlagRequired("from")
	cmd.MarkPersistentFlagRequired("into")

	return cmd
}

func sourceCommand(format, source, short, content string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   source,
		Short: short,
		Example: fmt.Sprintf(
			`translate %s %s %s --from=en --into=de --preserve='{[a-z]+?}' --deepl=$DEEPL_AUTH_KEY`,
			format,
			source,
			content,
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newService()
			if err != nil {
				return fmt.Errorf("create service: %w", err)
			}
			trans := translator.New(svc)

			opts := []translator.TranslateOption{
				translator.Parallel(parallel),
			}

			if preserve != "" {
				expr, err := regexp.Compile(preserve)
				if err != nil {
					return fmt.Errorf("compile regexp for `preserve` option: %w", err)
				}
				opts = append(opts, translator.Preserve(expr))
			}

			res, err := trans.Translate(
				cmd.Context(),
				strings.NewReader(args[0]),
				sourceLang,
				targetLang,
				json.Ranger(),
				opts...,
			)

			if err != nil {
				return fmt.Errorf("translate: %w", err)
			}

			if _, err = fmt.Fprintf(out, string(res)); err != nil {
				return fmt.Errorf("write result: %w", err)
			}

			return nil
		},
	}
	return cmd
}

func newService() (translator.Service, error) {
	switch {
	case deeplAuthKey != "":
		return deepl.New(deeplAuthKey), nil
	default:
		return nil, errors.New("missing authentication")
	}
}
