package dragoman_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/bounoable/dragoman"
	"github.com/bounoable/dragoman/format/json"
	mock_dragoman "github.com/bounoable/dragoman/mocks"
	"github.com/golang/mock/gomock"
)

func TestTranslator_Translate_Preserve(t *testing.T) {
	ctrl, ctx := gomock.WithContext(context.Background(), t)
	svc := mock_dragoman.NewMockService(ctrl)

	source := `{"msg": "Hello, {firstName}. Today is {day}."}`

	svc.EXPECT().Translate(gomock.Any(), "Hello,", "en", "de").Return("Hallo,", nil)
	svc.EXPECT().Translate(gomock.Any(), ". Today is", "en", "de").Return(". Heute ist", nil)

	translator := dragoman.New(svc)
	ranger := json.Ranger()

	translated, err := translator.Translate(ctx, strings.NewReader(source), "en", "de", ranger, dragoman.Preserve(regexp.MustCompile("{.+?}")))
	if err != nil {
		t.Fatal(err)
	}

	result := string(translated)
	want := `{"msg": "Hallo, {firstName}. Heute ist {day}."}`

	if result != want {
		t.Errorf("result == %q; want %q", result, want)
	}
}

func TestTranslator_Translate_Preserve_multiple(t *testing.T) {
	ctrl, ctx := gomock.WithContext(context.Background(), t)
	svc := mock_dragoman.NewMockService(ctrl)

	source := `{"msg": "Hello, {firstName}. Today is {day}. PreservedWord is here."}`

	svc.EXPECT().Translate(gomock.Any(), "Hello,", "en", "de").Return("Hallo,", nil)
	svc.EXPECT().Translate(gomock.Any(), ". Today is", "en", "de").Return(". Heute ist", nil)
	svc.EXPECT().Translate(gomock.Any(), "is here.", "en", "de").Return("ist hier.", nil)

	translator := dragoman.New(svc)
	ranger := json.Ranger()

	translated, err := translator.Translate(
		ctx,
		strings.NewReader(source),
		"en",
		"de",
		ranger,
		dragoman.Preserve(regexp.MustCompile("{.+?}")),
		dragoman.Preserve(regexp.MustCompile("PreservedWord")),
	)
	if err != nil {
		t.Fatal(err)
	}

	result := string(translated)
	want := `{"msg": "Hallo, {firstName}. Heute ist {day}. PreservedWord ist hier."}`

	if result != want {
		t.Errorf("result == %q; want %q", result, want)
	}
}
