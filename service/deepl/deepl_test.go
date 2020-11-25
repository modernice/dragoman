package deepl_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/bounoable/deepl"
	"github.com/bounoable/dragoman"
	deeplsvc "github.com/bounoable/dragoman/service/deepl"
	mock_deepl "github.com/bounoable/dragoman/service/deepl/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	expectedTranslated = "Übersetzter Text."
)

func TestNewWithClient(t *testing.T) {
	client := deepl.New("")
	svc := deeplsvc.NewWithClient(client)
	assert.Same(t, client, svc.Client())
}

func TestNew(t *testing.T) {
	svc := deeplsvc.New("")
	assert.NotNil(t, svc)
	var _ dragoman.Service = svc
}

func TestNew_withClientOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var httpClient http.Client
	svc := deeplsvc.New("", deeplsvc.WithClientOptions(
		deepl.BaseURL("custom-base-url"),
		deepl.HTTPClient(&httpClient),
	))

	deeplClient, ok := svc.Client().(*deepl.Client)

	assert.True(t, ok)
	assert.Equal(t, "custom-base-url", deeplClient.BaseURL())
	assert.Same(t, &httpClient, deeplClient.HTTPClient())
}

func TestService_Translate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_deepl.NewMockClient(ctrl)
	svc := deeplsvc.NewWithClient(client)

	text := "This is a sentence."
	sourceLang := "EN"
	targetLang := "DE"

	usedURLValues := expectClientTranslate(client, text, sourceLang, targetLang)

	translated, err := svc.Translate(context.Background(), text, sourceLang, targetLang)

	assert.Nil(t, err)

	// it returns the translated text
	assert.Equal(t, expectedTranslated, translated)

	// it constructs the correct options
	assert.Len(t, usedURLValues["text"], 1)
	assert.Equal(t, text, usedURLValues.Get("text"))
	assert.Equal(t, sourceLang, usedURLValues.Get("source_lang"))
	assert.Equal(t, targetLang, usedURLValues.Get("target_lang"))
	assert.Equal(t, "1", usedURLValues.Get("preserve_formatting"))
}

func TestService_Translate_withTranslateOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_deepl.NewMockClient(ctrl)
	opts := []deepl.TranslateOption{
		deepl.SplitSentences(deepl.SplitNoNewlines),
		deepl.Formality(deepl.MoreFormal),
		deepl.PreserveFormatting(false),
	}
	svc := deeplsvc.NewWithClient(client, deeplsvc.WithTranslateOptions(opts...))

	text := "This is a sentence."
	sourceLang := "EN"
	targetLang := "DE"

	usedURLValues := expectClientTranslate(client, text, sourceLang, targetLang)

	_, err := svc.Translate(context.Background(), text, sourceLang, targetLang)

	assert.Nil(t, err)

	// it constructs the correct options
	assert.Len(t, usedURLValues["text"], 1)
	assert.Equal(t, text, usedURLValues.Get("text"))
	assert.Equal(t, sourceLang, usedURLValues.Get("source_lang"))
	assert.Equal(t, targetLang, usedURLValues.Get("target_lang"))
	assert.Equal(t, "0", usedURLValues.Get("preserve_formatting"))
	assert.Equal(t, "more", usedURLValues.Get("formality"))
	assert.Equal(t, "nonewlines", usedURLValues.Get("split_sentences"))
}

func TestService_Translate_languageNormalization(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_deepl.NewMockClient(ctrl)
	svc := deeplsvc.NewWithClient(client)

	text := "This is a sentence."
	sourceLang := "en"
	targetLang := "de"

	usedURLValues := expectClientTranslate(client, text, "EN", "DE")

	_, err := svc.Translate(context.Background(), text, sourceLang, targetLang)

	assert.Nil(t, err)
	assert.Equal(t, "EN", usedURLValues.Get("source_lang"))
	assert.Equal(t, "DE", usedURLValues.Get("target_lang"))
}

func expectClientTranslate(client *mock_deepl.MockClient, text, sourceLang, targetLang string) url.Values {
	usedURLValues := url.Values{}
	client.EXPECT().
		Translate(gomock.Any(), text, deepl.Language(targetLang), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			text string,
			targetLang deepl.Language,
			opts ...deepl.TranslateOption,
		) (string, deepl.Language, error) {
			for k, v := range makeURLValues(text, string(targetLang), opts...) {
				usedURLValues[k] = v
			}
			return expectedTranslated, deepl.Language(sourceLang), nil
		})
	return usedURLValues
}

func makeURLValues(text, targetLang string, opts ...deepl.TranslateOption) url.Values {
	vals := url.Values{
		"text":        []string{text},
		"target_lang": []string{string(targetLang)},
	}
	for _, opt := range opts {
		opt(vals)
	}
	return vals
}
