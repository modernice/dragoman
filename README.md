# Dragoman

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
go get github.com/bounoable/dragoman/cmd/translate
```

### As a library

```sh
go get github.com/bounoable/dragoman
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
  "github.com/bounoable/dragoman"
  "github.com/bounoable/dragoman/json"
  "github.com/bounoable/dragoman/service/deepl"
)

func translateJSONFile(path, sourceLang, targetLang string) (string, error) {
  trans := dragoman.New(deepl.New(os.Getenv("DEEPL_AUTH_KEY")))

  f, err := os.Open()
  if err != nil {
    return fmt.Errorf("open file: %w", err)
  }
  defer f.Close()

  translated, err := trans.Translate(
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

## Preserve substrings (placeholders)

You can prevent translations of substrings matching against a regular expression by using the `Preserve()` option:

```go
// ...
res, _ := trans.Translate(
  context.Background(),
  strings.NewReader(`{"title": "Hello, {firstName}, how are you?"}`,
  "EN",
  "DE",
  dragoman.Preserve(regexp.MustCompile(`{[a-zA-Z]+?}`)),
))

fmt.Println(res)
// {"title": "Hallo, {firstName}, wie geht es Ihnen?"}
```

**:warning: Note that matched substrings are cut out of the sentence, and the remaining parts are translated independently. Then the cut out parts are reinserted into the sentence. If you have placeholders in sentences with complex grammar the translated sentence may end up grammatically incorrect.**

## License

[MIT](./LICENSE)
