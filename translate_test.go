package translator_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/bounoable/translator"
	mock_translator "github.com/bounoable/translator/mocks"
	"github.com/bounoable/translator/text"
	mock_text "github.com/bounoable/translator/text/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Translator.Translate", func() {
	var (
		ctrl            *gomock.Controller
		service         *mock_translator.MockService
		provider        *mock_text.MockRanger
		trans           *translator.Translator
		source          io.Reader
		sourceLang      string
		targetLang      string
		translateResult []byte
		translateError  error
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		service = mock_translator.NewMockService(ctrl)
		provider = mock_text.NewMockRanger(ctrl)
		trans = translator.New(service)
		source = nil
		sourceLang = "EN"
		targetLang = "DE"
		translateResult = nil
		translateError = nil
	})

	AfterEach(func() {
		defer ctrl.Finish()
	})

	JustBeforeEach(func(done Done) {
		translateResult, translateError = trans.Translate(
			context.Background(),
			source,
			sourceLang,
			targetLang,
			provider,
		)
		close(done)
	})

	Context("plain json", func() {
		setupJSONTest(&source, "./testdata/json/plain.json")

		BeforeEach(func() {
			By("calling the ranger to get the ranges that need to be translated", func() {
				provider.EXPECT().
					Ranges(gomock.Any(), gomock.Any()).
					DoAndReturn(func(context.Context, io.Reader) (<-chan text.Range, <-chan error) {
						ranges := make(chan text.Range, 2)
						ranges <- text.Range{14, 30} // "This is a title."
						ranges <- text.Range{51, 73} // "This is a description."
						close(ranges)
						return ranges, make(chan error)
					})
			})

			By("translating the text in those ranges through the translator service", func() {
				service.EXPECT().
					Translate(gomock.Any(), "This is a title.", sourceLang, targetLang).
					Return("Dies ist ein Titel.", nil)

				service.EXPECT().
					Translate(gomock.Any(), "This is a description.", sourceLang, targetLang).
					Return("Dies ist eine Beschreibung.", nil)
			})
		})

		It("doesn't return an error", func() {
			Ω(translateError).ShouldNot(HaveOccurred())
		})

		It("translates just the JSON values", func() {
			type document struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			}

			var doc document
			err := json.Unmarshal(translateResult, &doc)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(doc.Title).Should(Equal("Dies ist ein Titel."))
			Ω(doc.Description).Should(Equal("Dies ist eine Beschreibung."))
		})
	})
})

func setupJSONTest(source *io.Reader, path string) {
	BeforeEach(func() {
		f, err := os.Open(path)
		defer f.Close()
		Ω(err).ShouldNot(HaveOccurred())

		b, err := ioutil.ReadAll(f)
		Ω(err).ShouldNot(HaveOccurred())

		*source = bytes.NewReader(b)
	})
}
