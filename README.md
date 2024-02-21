# Dragoman - Translator for Structured Documents

[![PkgGoDev](https://pkg.go.dev/badge/github.com/modernice/dragoman)](https://pkg.go.dev/github.com/modernice/dragoman) ![Test](https://github.com/modernice/dragoman/workflows/Test/badge.svg)

Dragoman is an AI-powered tool for translating structured documents like JSON,
XML, YAML. The tool's key feature is its ability to maintain the document's 
structure during translation - keeping elements such as JSON keys and placeholders intact.

Dragoman is available as both a CLI tool and a Go library. This means you can
use it directly from your terminal for one-off tasks, or integrate it into your
Go applications for more complex use cases.

<sub>
If you're looking for a version of Dragoman that leverages conventional
translation services like Google Translate or DeepL, check out the
<a href="https://github.com/modernice/dragoman/tree/freeze">freeze</a> branch
of this repository. The previous implementation manually extracted texts from
the input files, translated them using DeepL or Google Translate, and reinserted
the translated pieces back into the original documents.
</sub>

## Installation

Dragoman can be installed directly using Go's built-in package manager:

```bash
go install github.com/modernice/dragoman/cmd/dragoman@latest
```

To add Dragoman to your Go project, install using `go get`:

```bash
go get github.com/modernice/dragoman
```

## Usage

The basic usage of Dragoman is as follows:

```bash
dragoman source.json
```

This command will translate the content of `source.json` to English and print
the translated document to stdout. The source language is automatically detected
by default, but if you want to specify the source or target languages, you need
to use the `--from` or `--to` option.

### Full list of available options

**`-f` or `--from`**

The source language of the document. It can be specified in any format that a
human would understand (like 'English', 'German', 'French', etc.). If not
provided, it defaults to 'auto', meaning the language is automatically detected.

```bash
dragoman source.json --from English
```

**`-t` or `--to`**

The target language to which the document will be translated. It can be
specified in any format that a human would understand (like 'English', 'German',
'French', etc.). If not provided, it defaults to 'English'.

```bash
dragoman source.json --to French
```

**`-o` or `--out`**

The path to the output file where the translated content will be saved. If this
option is not provided, the translated content will be printed to stdout.

```bash
dragoman source.json --out target.json
```

**`-u` or `--update`**

Enable this option to only translate missing fields from the source file that
are missing in the output file. This option requires the source and output files
to be JSON!

```bash
dragoman source.json --out target.json --update
```

#### Example

When you add new translations to your JSON source file, you can use the `--update`
option to only translate the newly added fields and merge them into the output file.

```json
// en.json
{
	"hello": "Hello, world!",
	"contact": {
		"email": "hello@example.com",
		"response": "Thank you for your message."
	}
}
```

```json
// de.json
{
	"hello": "Hallo, Welt!",
	"contact": {
		"email": "hallo@example.com"
	}
}
```

```bash
dragoman en.json --out de.json --update
```

Result:

```json
// de.json
{
	"hello": "Hallo, Welt!",
	"contact": {
		"email": "hallo@example.com",
		"response": "Vielen Dank für deine Nachricht."
	}
}
```

**`-p` or `--preserve`**

This option allows you to specify a list of specific words or phrases, separated by commas, that you want to remain unchanged during the translation process. It's particularly useful for ensuring that certain terms, which may have significance in their original form or are used in specific contexts (like code, trademarks, or names), are not altered. These specified terms will be recognized and preserved whether they appear in isolation or as part of larger strings. This feature is especially handy for content that includes embedded terms within other elements, such as HTML tags. For instance, using --preserve ensures that a term like <span class="font-bold">Drago</span>man retains its original form post-translation. Note that the effectiveness of this feature may vary depending on the language model used, and it is optimized for use with OpenAI's GPT models.

```bash
dragoman source.json --preserve Dragoman
```

**`-v` or `--verbose`**

A flag that, if provided, makes the CLI provide more detailed output about the
process and result of the translation.

```bash
dragoman source.json --verbose
```

**`-h` or `--help`**

A flag that displays a help message detailing how to use the command and its options.

```bash
dragoman --help
```

## Use as Library

Besides the CLI tool, Dragoman can also be used as a Go library in your own
applications. This allows you to build the Dragoman translation capabilities
directly into your own Go programs.

### Example: Basic Translation

In this example, we load a JSON file and translate its content using the default
source and target languages (automatic detection and English, respectively).

```go
package main

import (
	"fmt"
	"io"

	"github.com/modernice/dragoman"
	"github.com/modernice/dragoman/openai"
)

func main() {
	content, _ := io.ReadFile("source.json")
	
	service := openai.New()
	translator := dragoman.New(service)
	
	translated, _ := translator.Translate(context.TODO(), string(content))

	fmt.Println(translated)
}
```

### Example: Translation with Preserved Words

In this example, we translate a JSON file, specifying some preserved words that
should not be translated.

```go
package main

import (
	"fmt"
	"io"

	"github.com/modernice/dragoman"
	"github.com/modernice/dragoman/openai"
)

func main() {
	content, _ := io.ReadFile("source.json")
	
	service := openai.New()
	translator := dragoman.New(service)
	
	translated, _ := translator.Translate(
		context.TODO(),
		string(content),
		dragoman.Preserve([]string{"Dragoman", "OpenAI"}),
	)

	fmt.Println(translated)
}
```

### Example: Translation with Specific Source and Target Languages

In this example, we translate a JSON file from English to French, specifying the
source and target languages.

```go
package main

import (
	"fmt"
	"io"

	"github.com/modernice/dragoman"
	"github.com/modernice/dragoman/openai"
)

func main() {
	content, _ := io.ReadFile("source.json")
	
	service := openai.New()
	translator := dragoman.New(service)
	
	translated, _ := translator.Translate(
		context.TODO(),
		string(content),
		dragoman.Source("English"),
		dragoman.Target("French"),
	)

	fmt.Println(translated)
}
```

## License

[MIT](./LICENSE)
