package dragoman_test

import (
	"context"
	"io"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bounoable/dragoman"
	mock_dragoman "github.com/bounoable/dragoman/mocks"
	"github.com/bounoable/dragoman/text"
	mock_text "github.com/bounoable/dragoman/text/mocks"
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
					{13, 29}, // This is a title.
					{49, 71}, // This is a description.
				}, func(ranger text.Ranger) {
					WithTranslations(ctrl, "EN", "DE", map[string]string{
						"This is a title.":       "Dies ist ein Titel.",
						"This is a description.": "Dies ist eine Beschreibung.",
					}, func(svc dragoman.Service) {
						trans := dragoman.New(svc)
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
	"meta": {
		"title": "Hello, {firstName}, how are you {day}?",
		"description": "This is a sentence with a {placeholder} variable."
	}
}`)

			Convey("Given a JSON ranger", WithRanges(ctrl, []text.Range{
				{25, 63},  // "Hello, {firstName}, how are you {day}?"
				{84, 133}, // "This is a sentence with a {placeholder} variable."
			}, func(ranger text.Ranger) {
				Convey("When the text gets translated with the `Preserve()` option", WithTranslations(
					ctrl, "EN", "DE",
					map[string]string{
						"Hello,":                    "Hallo,",
						", how are you":             ", wie geht es Ihnen",
						"This is a sentence with a": "Dies ist ein Satz mit einer",
						"variable.":                 "Variable.",
					},
					func(svc dragoman.Service) {
						trans := dragoman.New(svc)
						result, err := trans.Translate(
							context.Background(),
							input,
							"EN",
							"DE",
							ranger,
							dragoman.Preserve(regexp.MustCompile("{[a-zA-Z]+?}")),
						)

						Convey("There should be no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("The string values should be translated, but the placeholders not", func() {
							So(string(result), ShouldEqual, `{
	"meta": {
		"title": "Hallo, {firstName}, wie geht es Ihnen {day}?",
		"description": "Dies ist ein Satz mit einer {placeholder} Variable."
	}
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
					time.Millisecond*500,
					func(svc dragoman.Service, maxActive *int64) {
						trans := dragoman.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, dragoman.Preserve(expr))

						Convey("Only 1 translations should have been active at a time", func() {
							So(*maxActive, ShouldEqual, 1)
						})
					}),
				)

				Convey("When the `Parallel(1)` option is used", WithParallelTranslations(
					ctrl,
					time.Millisecond*500,
					func(svc dragoman.Service, maxActive *int64) {
						trans := dragoman.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, dragoman.Preserve(expr), dragoman.Parallel(1))

						Convey("Only 1 translations should have been active at a time", func() {
							So(*maxActive, ShouldEqual, 1)
						})
					}),
				)

				Convey("When the `Parallel(2)` option is used", WithParallelTranslations(
					ctrl,
					time.Millisecond*500,
					func(svc dragoman.Service, maxActive *int64) {
						trans := dragoman.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, dragoman.Preserve(expr), dragoman.Parallel(2))

						Convey("2 translations should have been active at a time", func() {
							So(*maxActive, ShouldEqual, min(2, runtime.NumCPU()))
						})
					}),
				)

				Convey("When the `Parallel(0)` option is used", WithParallelTranslations(
					ctrl,
					time.Millisecond*500,
					func(svc dragoman.Service, maxActive *int64) {
						trans := dragoman.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, dragoman.Preserve(expr), dragoman.Parallel(0))

						Convey("no translation should have been made", func() {
							So(*maxActive, ShouldEqual, 0)
						})
					}),
				)

				Convey("When `Parallel()` option is used with a negative value", WithParallelTranslations(
					ctrl,
					time.Millisecond*500,
					func(svc dragoman.Service, maxActive *int64) {
						trans := dragoman.New(svc)
						trans.Translate(context.Background(), input, sourceLang, targetLang, ranger, dragoman.Preserve(expr), dragoman.Parallel(-1))

						Convey("no translation should have been made", func() {
							So(*maxActive, ShouldEqual, 0)
						})
					}),
				)
			}))
		})
	})

	Convey("Punctuation handling", t, func() {
		ctrl := gomock.NewController(t)
		Reset(ctrl.Finish)

		Convey("Given an input that's only punctuation", func() {
			input := "!-/:-@[-`{-~" // [[:punct:]] named ASCII character class

			Convey("And a text ranger", WithRanges(ctrl, []text.Range{{0, 12}}, func(ranger text.Ranger) {
				Convey("Then no translation request should be made", WithTranslations(ctrl, "", "", map[string]string{}, func(svc dragoman.Service) {
					trans := dragoman.New(svc)
					res, err := trans.Translate(context.Background(), strings.NewReader(input), "EN", "DE", ranger)

					So(err, ShouldBeNil)
					So(string(res), ShouldEqual, input)
				}))
			}))
		})
	})

	Convey("Whitespace handling", t, func() {
		ctrl := gomock.NewController(t)
		Reset(ctrl.Finish)

		Convey("Given an input with some placeholders", func() {
			input := `Hello, {firstName}! How are you {day}?`

			Convey("And a text ranger", WithRanges(ctrl, []text.Range{{0, 38}}, func(ranger text.Ranger) {
				Convey("Then the whitespace should be trimmed before making the translation request", WithTranslations(
					ctrl,
					"EN", "EN",
					map[string]string{
						"Hello,":        "Hello,",
						"! How are you": "! How are you",
					},
					func(svc dragoman.Service) {
						Convey("And the translated text should contain the trimmed whitespace", func() {
							trans := dragoman.New(svc)
							res, err := trans.Translate(
								context.Background(),
								strings.NewReader(input),
								"EN", "EN",
								ranger,
								dragoman.Preserve(regexp.MustCompile(`{[a-zA-Z]+?}`)),
							)

							So(err, ShouldBeNil)
							So(string(res), ShouldEqual, input)
						})
					}),
				)
			}))
		})
	})

	Convey("Double quote handling", t, func() {
		ctrl := gomock.NewController(t)
		Reset(ctrl.Finish)

		Convey("Given a JSON string with double quotes", func() {
			input := `"\"one\", \"two\", \"three\""`

			Convey("And a text ranger", WithRanges(ctrl, []text.Range{{1, 28}}, func(ranger text.Ranger) {
				Convey("Then the translated text should also escape the double quotes", WithTranslations(
					ctrl,
					"EN", "EN",
					map[string]string{
						`\"one\", \"two\", \"three\"`: `"one", "two", "three"`,
					},
					func(svc dragoman.Service) {
						trans := dragoman.New(svc)
						res, err := trans.Translate(
							context.Background(),
							strings.NewReader(input),
							"EN", "EN",
							ranger,
							dragoman.EscapeDoubleQuotes(true),
						)

						So(err, ShouldBeNil)
						So(string(res), ShouldEqual, `"\"one\", \"two\", \"three\""`)
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
	f func(dragoman.Service),
) func() {
	return func() {
		svc := mock_dragoman.NewMockService(ctrl)
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
	f func(dragoman.Service, *int64),
) func() {
	return func() {
		var active int64
		var maxActive int64
		svc := mock_dragoman.NewMockService(ctrl)

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

func min(nums ...int) int {
	if len(nums) == 0 {
		return 0
	}
	min := nums[0]
	for _, n := range nums[1:] {
		if n < min {
			min = n
		}
	}
	return min
}
