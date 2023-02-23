# Dragoman - Translate structured documents

[![PkgGoDev](https://pkg.go.dev/badge/github.com/bounoable/dragoman)](https://pkg.go.dev/github.com/bounoable/dragoman) ![Test](https://github.com/bounoable/dragoman/workflows/Test/badge.svg)

## TL;DR – Translate JSON files, but preserve keys!

Translate the file `i18n/en.json` from `English` into `German` via `DeepL` while preserving placeholders and write the result to `i18n/de.json`:

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
go install github.com/bounoable/dragoman/cmd/translate@latest
```

### API

```sh
go get github.com/bounoable/dragoman
```

## CLI

### Authentication

Choose and authenticate the underlying translation service by providing either one of these options:

- `--deepl $DEEPL_AUTH_KEY` to use [DeepL](https://deepl.com) with your DeepL API key
- `--gcloud $CREDENTIALS_FILE` to use [Google Cloud Translation](https://cloud.google.com/translate) and authenticate with a credentials file

**DeepL:**
```sh
translate json text '{"foo": "Hello, my friend."}' --from en --into de --deepl $DEEPL_AUTH_KEY

# Output: {"foo": "Hallo, mein Freund."}
```

**Google Cloud Translation:**
```sh
translate json text '{"foo": "Hello, my friend."}' --from en --into de --gcloud ./credentials.json

# Output: {"foo": "Hallo, mein Freund."}
```

### Translate files

The following example translates the JSON file `en.json` from English into German and writes the result to `de.json`:

```sh
translate json file en.json -o de.json --from en --into de --deepl $DEEPL_AUTH_KEY
```

### Translate directories

The following example translates all JSON files in the directory `i18n/en` from English into German and writes the result to `i18n/de`:

```sh
translate json dir i18n/en -o i18n/de --from en --into de --deepl $DEEPL_AUTH_KEY
```

### Preserve substrings (placeholders)

```sh
translate json text '{"foo": "Hello, {firstName}."}' --from en --into de --preserve '{[a-zA-Z]+?}' --deepl $DEEPL_AUTH_KEY

# Output: {"foo": "Hallo, {firstName}."}
```

### Supported formats

- [x] `json`
- [x] `html`

### Supported sources

- [x] `text`
- [x] `file`
- [x] `dir`
- [ ] `url`

## API

### Authentication

**Deepl:**
```go
import (
  "github.com/bounoable/dragoman"
  "github.com/bounoable/dragoman/service/deepl"
)

dm := dragoman.New(deepl.New(os.Getenv("DEEPL_AUTH_KEY")))
```

**Google Cloud Translation:**
```go
import (
  "github.com/bounoable/dragoman"
  "github.com/bounoable/dragoman/service/gcloud"
)

svc, err := gcloud.NewFromCredentialsFile(context.TODO(), "./credentials.json")
// handle err
dm := dragoman.New(svc)
```

### Translate files

```go
import (
  "github.com/bounoable/dragoman"
  "github.com/bounoable/dragoman/format/json"
  "github.com/bounoable/dragoman/service/deepl"
)

func translateJSONFile(path, sourceLang, targetLang string) (string, error) {
  dm := dragoman.New(deepl.New(os.Getenv("DEEPL_AUTH_KEY")))

  f, err := os.Open()
  if err != nil {
    return fmt.Errorf("open file: %w", err)
  }
  defer f.Close()

  translated, err := dm.Translate(
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

For more examples visit [**pkg.go.dev**](https://pkg.go.dev/github.com/bounoable/dragoman) or [**example_test.go**](./example_test.go).

## Preserve substrings (placeholders)

You can prevent translations of substrings matching a regular expression by using the `Preserve()` option:

```go
// ...
res, _ := dm.Translate(
  context.Background(),
  strings.NewReader(`{"title": "Hello, {firstName}, how are you?"}`,
  "en", "de",
  dragoman.Preserve(regexp.MustCompile(`{[a-zA-Z]+?}`)),
))

fmt.Println(res)
// {"title": "Hallo, {firstName}, wie geht es Ihnen?"}
```

> :warning: Note that matched substrings are cut out of the sentence, and the remaining parts are translated independently. The cut out parts are then reinserted between the translated strings, so if you have placeholders in sentences with complex grammar, **the translated result may end up grammatically incorrect.**

## License

[MIT](./LICENSE)
