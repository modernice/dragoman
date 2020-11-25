package translator_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/bounoable/translator"
	"github.com/bounoable/translator/json"
	"github.com/bounoable/translator/service/deepl"
)

func ExampleTranslator_Translate_json() {
	svc := deepl.New(os.Getenv("DEEPL_AUTH_KEY"))
	trans := translator.New(svc)

	res, err := trans.Translate(
		context.TODO(),
		strings.NewReader(`{"title": "This is a title."}`),
		"EN",
		"DE",
		json.Ranger(),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res)
}

func ExampleTranslator_Translate_jsonWithPlaceholder() {
	svc := deepl.New(os.Getenv("DEEPL_AUTH_KEY"))
	trans := translator.New(svc)

	res, err := trans.Translate(
		context.TODO(),
		strings.NewReader(`{"greeting": "Hello, {firstName}!"}`),
		"EN",
		"DE",
		json.Ranger(),
		translator.Preserve(regexp.MustCompile(`{[a-zA-Z]+?}`)),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res)
}
