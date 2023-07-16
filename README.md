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

This command will translate the content of `source.json` and print the
translated document to stdout. The source language is automatically detected by
default, but if you want to specify the source or target languages, you need to
use the `--from` or `--to` option.

### Full list of available options

**`-f` or `--from`**

The source language of the document. It can be specified in any format that a
human would understand (like 'English', 'German', 'French', etc.). If not
provided, it defaults to 'auto', meaning the language is automatically detected.

Example:

```bash
dragoman source.json --from English --to French
```

**`-t` or `--to`**

The target language to which the document will be translated. It can be
specified in any format that a human would understand (like 'English', 'German',
'French', etc.). If not provided, it defaults to 'English'.

Example:

```bash
dragoman source.json --to French
```

**`-o` or `--output`**

The path to the output file where the translated content will be saved. If this
option is not provided, the translated content will be printed to stdout.

Example:

```bash
dragoman source.json --output target.json
```

**`-p` or `--preserve`**

A comma-separated list of words or terms that should not be translated.
The preserved words will be recognized not only as stand-alone words but also as
part of larger expressions. This could be useful, for example, when the known
word is embedded within HTML tags or combined with other words. 

Example:

```bash
dragoman source.json --preserve Dragoman
```

In this example, a term like `<span class="font-bold">Drago</span>man` will not
be translated.

**`-v` or `--verbose`**

A flag that, if provided, makes the CLI provide more detailed output about the
process and result of the translation.

Example:

```bash
dragoman source.json --verbose
```

**`-h` or `--help`**

A flag that displays a help message detailing how to use the command and its options.

Example:

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
