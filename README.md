# Dragoman - Translator for Structured Documents

Dragoman is a command line tool designed for translating structured documents
including but not limited to formats like JSON, XML, YAML. The tool's key
feature is its ability to maintain the document's structure during translation -
keeping vital elements such as JSON keys and placeholders intact.

Dragoman is available as both a CLI tool and a Go library. This means you can
use it directly from your terminal for one-off tasks, or integrate it into your
Go applications for more complex use cases.

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

<div style="background: rgba(130, 40, 40, 1); padding: 0.5rem 1rem; border-radius: 4px; color: #fff;">
<p style="margin: 0;"><strong>Warning:</strong> This option only works reliably
with the <strong>gpt-4</strong> model.</p>
</div>

A comma-separated list of words or terms that should not be translated.
The preserved words will be recognized not only as stand-alone words but also as
part of larger expressions. This could be useful, for example, when the known
word is embedded within HTML tags or combined with other words. 

Example:

```bash
dragoman source.json --preserve "Dragoman,OpenAI"
```

In this example, a term like `<span class="font-bold">Drago</span>man` will not
be translated due to the flexibility of the --known option.

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

## Examples

Translate a JSON file from a detected language to English and print the result
to stdout:

```bash
dragoman source.json
```

Translate a JSON file from English to French and save the result to a file:

```bash
dragoman source.json --from English --to French --output target.json
```

Translate a JSON file from a detected language to French, with certain known
words not translated:

```bash
dragoman source.json --to French --known "Dragoman,OpenAI"
```

## Go Library

Besides the CLI tool, Dragoman can also be used as a Go library in your own
applications. This allows you to build the Dragoman translation capabilities
directly into your own Go programs.

Here are some examples of how you can use the Dragoman library:

### Example 1: Basic Translation

In this example, we load a JSON file and translate its content using the default
source and target languages (automatic detection and English, respectively).

```go
package main

import (
	"fmt"
	"io/ioutil"

	"github.com/modernice/dragoman"
	"github.com/modernice/dragoman/openai"
)

func main() {
	content, _ := ioutil.ReadFile("source.json")
	
	service := openai.New()
	translator := dragoman.New(service)
	
	translated, err := translator.Translate(context.TODO(), string(content))
	if err != nil {
			fmt.Println("Error in translation:", err)
	}

	fmt.Println(translated)
}
```

### Example 2: Translation with Preserved Words

In this example, we translate a JSON file, specifying some preserved words that
should not be translated.

```go
package main

import (
	"fmt"
	"io/ioutil"

	"github.com/modernice/dragoman"
	"github.com/modernice/dragoman/openai"
)

func main() {
	content, _ := ioutil.ReadFile("source.json")
	
	service := openai.New()
	translator := dragoman.New(service)
	
	translated, err := translator.Translate(
		context.TODO(),
		string(content),
		dragoman.Preserve([]string{"Dragoman", "OpenAI"}),
	)
	if err != nil {
			fmt.Println("Error in translation:", err)
	}

	fmt.Println(translated)
}
```

### Example 3: Translation with Specific Source and Target Languages

In this example, we translate a JSON file from English to French, specifying the
source and target languages.

```go
package main

import (
	"fmt"
	"io/ioutil"

	"github.com/modernice/dragoman"
	"github.com/modernice/dragoman/openai"
)

func main() {
	content, _ := ioutil.ReadFile("source.json")
	
	service := openai.New()
	translator := dragoman.New(service)
	
	translated, err := translator.Translate(
		context.TODO(),
		string(content),
		dragoman.Source("English"),
		dragoman.Target("French"),
	)
	if err != nil {
			fmt.Println("Error in translation:", err)
	}

	fmt.Println(translated)
}
```

## License

[MIT](./LICENSE)
