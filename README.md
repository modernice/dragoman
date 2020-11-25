# Structured Text Translator (working title)

Translate texts in structured formats.

## TL;DR – Translate JSON files, but preserve key names!

Translate the file `i18n/en.json` from `English` into `German` via `DeepL` while preserving placeholders, saving the result into `i18n/de.json`:

```sh
translate json file i18n/en.json -o i18n/de.json --from en --into de --preserve '{[a-zA-Z]+?}' --deepl $DEEPL_AUTH_KEY
```

```js
// i18n/en.json
{
  "title": "Hello, {firstName}! This is a title."
  "body": "And this is another sentence."
}
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
translate json file en.json -o de.json --from en --into de --deepl $DEEPL_AUTH_KEY
```

The syntax for translating files looks like this:

```sh
translate FORMAT SOURCE CONTENT -opt1 -opt2 ...
```

### Formats

- [x] `json`
- [ ] `plain`
- [ ] `html`
- [ ] `markdown`

### Sources

- [x] `text`
- [x] `file`
- [ ] `webpage`

## Use as library

```go
import (
  "github.com/bounoable/translator"
  "github.com/bounoable/translator/json"
  "github.com/bounoable/translator/service/deepl"
)

func translateJSONFile(path, sourceLang, targetLang string) (string, error) {
  translator := translator.New(deepl.New(os.Getenv("DEEPL_AUTH_KEY")))

  f, err := os.Open()
  if err != nil {
    return fmt.Errorf("open file: %w", err)
  }
  defer f.Close()

  translated, err := translator.Translate(
    context.TODO(),
    f,
    sourceLang,
    targetLang,
    json.Ranger(),
    // options ...
  )
  if err != nil {
    return fmt.Errorf("translate file: %w", err)
  }

  return string(translated), nil
}
```

## License

[MIT](./LICENSE)
