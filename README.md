# Structured Text Translator (working title)

Translate texts in structured formats.

## TL;DR – Translate JSON files, but preserve key names!

```js
// i18n/en.json
{
  "title": "Hello, {firstName}! This is a title."
  "body": "And this is another sentence."
}
```

```sh
translate json file i18n/en.json -o i18n/de.json -from en -into de -preserve '{[a-zA-Z]+?}' -deepl $DEEPL_AUTH_KEY
```

File gets translated, but property names and placeholder variables are preserved:

```js
// i18n/de.json
{
  "title": "Hallo, {firstName}! Dies ist ein Titel."
  "body": "Und dies ist ein weiterer Satz."
}
```

## Installation

### CLI

```sh
go get github.com/bounoable/translator/cmd/translate
```

### As a library

```sh
go get github.com/bounoable/translator
```

## Usage with CLI

At the time of this writing only DeepL is implemented as a translation service, so you need a DeepL Pro Account and your authentication key.

Then you just run the following command to translate the JSON file `en.json` from English into German:

```sh
translate json file en.json -o de.json -from en -into de -deepl $DEEPL_AUTH_KEY
```

The syntax for translating files via the CLI looks like this:

```sh
translate FORMAT SOURCE CONTENT -opt1 -opt2 ...
```

### Formats

- `plain` [ ]
- `json` [ ]
- `html` [ ]
- `markdown` [ ]

### Sources

- `text` [ ]
- `file` [ ]
- `web` [ ]

## Use as library

```go
func translateJSONFile(path, sourceLang, targetLang string) (string, error) {
  translator := translator.New(deepl.New(os.Getenv("DEEPL_AUTH_KEY")))

  f, err := os.Open()
  if err != nil {
    return fmt.Errorf("open file: %w", err)
  }
  defer f.Close()

  translated, err := translator.Translate(context.TODO(), f, sourceLang, targetLang)
  if err != nil {
    return fmt.Errorf("translate: %w", err)
  }

  return string(translated), nil
}
```

## License

[MIT](./LICENSE)
