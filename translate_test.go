package translator_test

import (
	"context"
	"io"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

	Convey("Feature: parallel translations", t, func() {
		ctrl := gomock.NewController(t)
		Reset(ctrl.Finish)

		Convey("Given a JSON file with 2 ranges", func() {
			input := strings.NewReader(`{
	"title": "Hello, {firstName}, how are you {day}?",
	"description": "This is a sentence with a {placeholder} variable."
}`)
			expr := regexp.MustCompile("{[a-z-A-Z0-9]+?}")
			sourceLang := "EN"
			targetLang := "DE"

			Convey("Given a text ranger", WithRanges(ctrl, []text.Range{{13, 51}, {71, 120}}, func(ranger text.Ranger) {
				Convey("When the `Parallel()` option is not used", WithParallelTranslations(
					ctrl,
					time.Millisecond*200,
					func(svc translator.Service, maxActive *int64) {
						trans := translator.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, translator.Preserve(expr))

						Convey("Only 1 translations should have been active at a time", func() {
							So(*maxActive, ShouldEqual, 1)
						})
					}),
				)

				Convey("When the `Parallel(1)` option is used", WithParallelTranslations(
					ctrl,
					time.Millisecond*200,
					func(svc translator.Service, maxActive *int64) {
						trans := translator.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, translator.Preserve(expr), translator.Parallel(1))

						Convey("Only 1 translations should have been active at a time", func() {
							So(*maxActive, ShouldEqual, 1)
						})
					}),
				)

				Convey("When the `Parallel(2)` option is used", WithParallelTranslations(
					ctrl,
					time.Millisecond*200,
					func(svc translator.Service, maxActive *int64) {
						trans := translator.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, translator.Preserve(expr), translator.Parallel(2))

						Convey("2 translations should have been active at a time", func() {
							So(*maxActive, ShouldEqual, 2)
						})
					}),
				)

				Convey("When the `Parallel(0)` option is used", WithParallelTranslations(
					ctrl,
					time.Millisecond*200,
					func(svc translator.Service, maxActive *int64) {
						trans := translator.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, translator.Preserve(expr), translator.Parallel(0))

						Convey("no translation should have been made", func() {
							So(*maxActive, ShouldEqual, 0)
						})
					}),
				)

				Convey("When `Parallel()` option is used with a negative value", WithParallelTranslations(
					ctrl,
					time.Millisecond*500,
					func(svc translator.Service, maxActive *int64) {
						trans := translator.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, translator.Preserve(expr), translator.Parallel(-1))

						Convey("no translation should have been made", func() {
							So(*maxActive, ShouldEqual, 0)
						})
					}),
				)
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
				go func() {
					defer close(result)
					for _, r := range ranges {
						result <- r
					}
				}()
				return result, make(chan error)
			})
		f(ranger)
	}
}

func WithTranslations(
	ctrl *gomock.Controller,
	sourceLang, targetLang string,
	m map[string]string,
	f func(translator.Service),
) func() {
	return func() {
		svc := mock_translator.NewMockService(ctrl)
		for i, o := range m {
			svc.EXPECT().
				Translate(gomock.Any(), i, sourceLang, targetLang).
				Return(o, nil)
		}
		f(svc)
	}
}

func WithParallelTranslations(
	ctrl *gomock.Controller,
	d time.Duration,
	f func(translator.Service, *int64),
) func() {
	return func() {
		var active int64
		var maxActive int64
		svc := mock_translator.NewMockService(ctrl)

		svc.EXPECT().
			Translate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(context.Context, string, string, string) (string, error) {
				a := atomic.AddInt64(&active, 1)
				defer atomic.AddInt64(&active, -1)

				if a > atomic.LoadInt64(&maxActive) {
					atomic.StoreInt64(&maxActive, a)
				}

				time.Sleep(d)
				return "", nil
			}).
			AnyTimes()

		f(svc, &maxActive)
	}
}
