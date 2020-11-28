package dragoman_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/bounoable/dragoman"
	"github.com/bounoable/dragoman/format/html"
	"github.com/bounoable/dragoman/format/json"
	"github.com/bounoable/dragoman/service/deepl"
)

func ExampleTranslator_Translate_json() {
	svc := deepl.New(os.Getenv("DEEPL_AUTH_KEY"))
	dm := dragoman.New(svc)

	res, err := dm.Translate(
		context.TODO(),
		strings.NewReader(`{"title": "This is a title."}`),
		"en",
		"de",
		json.Ranger(),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(res))
}

func ExampleTranslator_Translate_jsonWithPlaceholder() {
	svc := deepl.New(os.Getenv("DEEPL_AUTH_KEY"))
	dm := dragoman.New(svc)

	res, err := dm.Translate(
		context.TODO(),
		strings.NewReader(`{"greeting": "Hello, {firstName}!"}`),
		"en",
		"de",
		json.Ranger(),
		dragoman.Preserve(regexp.MustCompile(`{[a-zA-Z]+?}`)),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res)
}

func ExampleTranslator_Translate_html() {
	svc := deepl.New(os.Getenv("DEEPL_AUTH_KEY"))
	dm := dragoman.New(svc)

	res, err := dm.Translate(
		context.TODO(),
		strings.NewReader(`<p>This is an example.</p>`),
		"en",
		"de",
		html.Ranger(),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res)
}

func ExampleTranslator_Translate_htmlWithPlaceholder() {
	svc := deepl.New(os.Getenv("DEEPL_AUTH_KEY"))
	dm := dragoman.New(svc)

	res, err := dm.Translate(
		context.TODO(),
		strings.NewReader(`<p>Hello, {firstName}, this is an example.</p>`),
		"en",
		"de",
		html.Ranger(),
		dragoman.Preserve(regexp.MustCompile(`{[a-zA-Z]+?}`)),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res)
}

func ExampleTranslator_Translate_htmlWithAttributes() {
	svc := deepl.New(os.Getenv("DEEPL_AUTH_KEY"))
	dm := dragoman.New(svc)

	res, err := dm.Translate(
		context.TODO(),
		strings.NewReader(`<p title="A title tag.">Hello, here is an <img src="someimage.jpeg" alt="An alternate description."></p>`),
		"en",
		"de",
		html.Ranger(
			html.WithAttribute("title", "alt"), // allow all `title` and `alt` attributes to be translated
			// OR
			html.MustAttributePath("p.title", "img.alt"), // allow `title` attribute of `p` tags and `alt` attribute of `img` tags to be translated
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res)
}

func ExampleTranslator_Translate_htmlWithPlaceholderAndAttributes() {
	svc := deepl.New(os.Getenv("DEEPL_AUTH_KEY"))
	dm := dragoman.New(svc)

	res, err := dm.Translate(
		context.TODO(),
		strings.NewReader(`<p title="A title tag.">Hello, {firstName}, here is an <img src="someimage.jpeg" alt="An alternate description."></p>`),
		"en",
		"de",
		html.Ranger(
			html.MustAttributePath("p.title", "img.alt"), // allow `title` attribute of `p` tags and `alt` attribute of `img` tags to be translated
		),
		dragoman.Preserve(regexp.MustCompile(`{[a-zA-Z]+?}`)),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res)
}
