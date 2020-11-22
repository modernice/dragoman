package translator_test

import (
	"context"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/bounoable/translator"
	mock_translator "github.com/bounoable/translator/mocks"
	"github.com/bounoable/translator/text"
	mock_text "github.com/bounoable/translator/text/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTranslator_Translate(t *testing.T) {
	Convey("Feature: translate JSON", t, func() {
		ctrl := gomock.NewController(t)
		Reset(ctrl.Finish)

		Convey("Given a plain JSON file", func() {
			input := strings.NewReader(`{
	"title": "This is a title.",
	"description": "This is a description."
}`)

			Convey(
				"When the text gets translated",
				WithRanges(ctrl, []text.Range{
					{13, 29}, // "This is a title."
					{49, 71}, // "This is a description."
				}, func(ranger text.Ranger) {
					WithTranslations(ctrl, "EN", "DE", map[string]string{
						"This is a title.":       "Dies ist ein Titel.",
						"This is a description.": "Dies ist eine Beschreibung.",
					}, func(svc translator.Service) {
						trans := translator.New(svc)
						result, err := trans.Translate(context.Background(), input, "EN", "DE", ranger)

						Convey("There should be no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("The string values should be translated", func() {
							So(string(result), ShouldEqual, `{
	"title": "Dies ist ein Titel.",
	"description": "Dies ist eine Beschreibung."
}`)
						})
					})()
				}),
			)
		})

		Convey("Given a JSON file with placeholders", func() {
			input := strings.NewReader(`{
	"title": "Hello, {firstName}, how are you {day}?",
	"description": "This is a sentence with a {placeholder} variable."
}`)

			Convey("Given a JSON ranger", WithRanges(ctrl, []text.Range{
				{13, 51},  // "Hello, {firstName}, how are you {day}?"
				{71, 120}, // "This is a sentence with a {placeholder} variable."
			}, func(ranger text.Ranger) {
				Convey("When the text gets translated with the `Preserve()` option", WithTranslations(
					ctrl, "EN", "DE",
					map[string]string{
						"Hello, ":                    "Hallo, ",
						", how are you ":             ", wie geht es Ihnen ",
						"?":                          "?",
						"This is a sentence with a ": "Dies ist ein Satz mit einer ",
						" variable.":                 " Variable.",
					},
					func(svc translator.Service) {
						trans := translator.New(svc)
						result, err := trans.Translate(
							context.Background(),
							input,
							"EN",
							"DE",
							ranger,
							translator.Preserve(regexp.MustCompile("{[a-zA-Z]+?}")),
						)

						Convey("There should be no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("The string values should be translated, but the placeholders not", func() {
							So(string(result), ShouldEqual, `{
	"title": "Hallo, {firstName}, wie geht es Ihnen {day}?",
	"description": "Dies ist ein Satz mit einer {placeholder} Variable."
}`)
						})
					},
				))
			}))
		})
	})
}

func WithRanges(ctrl *gomock.Controller, ranges []text.Range, f func(text.Ranger)) func() {
	return func() {
		ranger := mock_text.NewMockRanger(ctrl)
		ranger.EXPECT().
			Ranges(gomock.Any(), gomock.Any()).
			DoAndReturn(func(context.Context, io.Reader) (<-chan text.Range, <-chan error) {
				result := make(chan text.Range, len(ranges))
				for _, r := range ranges {
					result <- r
				}
				close(result)
				return result, make(chan error)
			})
		f(ranger)
	}
}

func WithTranslations(ctrl *gomock.Controller, sourceLang, targetLang string, m map[string]string, f func(translator.Service)) func() {
	return func() {
		svc := mock_translator.NewMockService(ctrl)
		for i, o := range m {
			o := o
			svc.EXPECT().
				Translate(gomock.Any(), i, sourceLang, targetLang).
				DoAndReturn(func(context.Context, string, string, string) (string, error) {
					return o, nil
				})
		}
		f(svc)
	}
}
