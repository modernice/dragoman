package gcloud_test

import (
	"context"
	"testing"

	"github.com/bounoable/dragoman"
	"github.com/bounoable/dragoman/service/gcloud"
	mock_gcloud "github.com/bounoable/dragoman/service/gcloud/mocks"
	"github.com/golang/mock/gomock"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/cloud/translate/v3"
)

const (
	expectedTranslated = "Übersetzter Text."
)

func TestNewWithClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_gcloud.NewMockClient(ctrl)
	svc := gcloud.NewWithClient(client, "foo")
	assert.Same(t, client, svc.Client())
	assert.Equal(t, "foo", svc.ProjectID())

	defer func() {
		msg := recover()
		assert.Equal(t, "nil client", msg)
	}()

	svc = gcloud.NewWithClient(nil, "foo")
}

func TestNew(t *testing.T) {
	svc := gcloud.New("foo")
	assert.NotNil(t, svc)
	var _ dragoman.Service = svc
}

func TestService_Translate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_gcloud.NewMockClient(ctrl)
	svc := gcloud.NewWithClient(client, "foo")

	text := "This is a sentence."
	sourceLang := "en"
	targetLang := "de"

	client.EXPECT().
		TranslateText(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *translate.TranslateTextRequest, _ ...gax.CallOption) (*translate.TranslateTextResponse, error) {
			assert.Equal(t, "This is a sentence.", req.GetContents()[0])
			assert.Equal(t, "en", req.GetSourceLanguageCode())
			assert.Equal(t, "de", req.GetTargetLanguageCode())
			assert.Equal(t, "text/html", req.GetMimeType())
			assert.Equal(t, "projects/foo", req.GetParent())

			return &translate.TranslateTextResponse{
				Translations: []*translate.Translation{
					{TranslatedText: expectedTranslated},
				},
			}, nil
		})

	translated, err := svc.Translate(context.Background(), text, sourceLang, targetLang)

	assert.Nil(t, err)
	assert.Equal(t, expectedTranslated, translated)
}

func TestService_Translate_requestOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_gcloud.NewMockClient(ctrl)
	svc := gcloud.NewWithClient(client, "foo", gcloud.WithRequestOptions(
		func(req *translate.TranslateTextRequest) {
			req.Parent = "bar"
		},
		func(req *translate.TranslateTextRequest) {
			req.Model = "baz"
		},
	))

	text := "This is a sentence."
	sourceLang := "en"
	targetLang := "de"

	client.EXPECT().
		TranslateText(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *translate.TranslateTextRequest, _ ...gax.CallOption) (*translate.TranslateTextResponse, error) {
			assert.Equal(t, "bar", req.GetParent())
			assert.Equal(t, "baz", req.GetModel())

			return &translate.TranslateTextResponse{
				Translations: []*translate.Translation{
					{TranslatedText: expectedTranslated},
				},
			}, nil
		})

	translated, err := svc.Translate(context.Background(), text, sourceLang, targetLang)

	assert.Nil(t, err)
	assert.Equal(t, expectedTranslated, translated)
}
