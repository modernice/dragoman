package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bounoable/dragoman"
	"github.com/bounoable/dragoman/cli"
	"github.com/bounoable/dragoman/format/html"
	"github.com/bounoable/dragoman/format/json"
	"github.com/bounoable/dragoman/service/deepl"
	"github.com/bounoable/dragoman/text"
	"github.com/spf13/pflag"
)

func main() {
	var (
		htmlAttrs     []string
		htmlAttrPaths []string
	)

	if err := cli.New(
		cli.WithTranslator(
			cli.Translator{
				Name:        "deepl",
				Description: "DeepL authentication key",
				New: func(authKey string) (dragoman.Service, error) {
					return deepl.New(authKey), nil
				},
			},
		),
		cli.WithFormat(
			cli.Format{
				Name:  "json",
				Ext:   ".json",
				Short: "Translate JSON",
				Ranger: func() (text.Ranger, error) {
					return json.Ranger(), nil
				},
			},
			cli.Format{
				Name:  "html",
				Ext:   ".html",
				Short: "Translate HTML",
				Flags: func(flags *pflag.FlagSet) {
					flags.StringSliceVar(&htmlAttrs, "attr", nil, `HTML tag attributes to be translated (e.g. "alt", "title")`)
					flags.StringSliceVar(&htmlAttrPaths, "attr-path", nil, `HTML tag attribute paths to be translated (e.g. "img.alt", "a.title")`)
				},
				Ranger: func() (text.Ranger, error) {
					var opts []html.Option
					for _, attr := range htmlAttrs {
						opts = append(opts, html.WithAttribute(attr))
					}
					for _, path := range htmlAttrPaths {
						opt, err := html.WithAttributePath(path)
						if err != nil {
							return nil, err
						}
						opts = append(opts, opt)
					}
					return html.Ranger(opts...), nil
				},
			},
		),
		cli.WithSource(
			cli.Source{
				Name: "text",
				Reader: func(val string) (io.Reader, error) {
					return strings.NewReader(val), nil
				},
			},
			cli.Source{
				Name: "file",
				Reader: func(val string) (io.Reader, error) {
					return os.Open(val)
				},
			},
		),
		cli.WithExample("json", "text", `'{"title": "This is an example."}' -o out.json`),
		cli.WithExample("json", "file", `i18n/en.json -o i18n/de.json`),
		cli.WithExample("html", "text", `'<p>This is an example.</p>' -o out.html`),
		cli.WithExample("html", "file", `index.html -o out.html`),
	).Execute(); err != nil {
		msg := err.Error()

		var herr interface {
			HumanError() string
		}

		if errors.As(err, &herr) {
			msg = herr.HumanError()
		}

		fmt.Println(msg)
		os.Exit(1)
	}
}
