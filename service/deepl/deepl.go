// Package deepl provides the DeepL-backed translation service.
package deepl

//go:generate mockgen -source=deepl.go -destination=./mocks/deepl.go

import (
	"context"
	"fmt"
	"strings"

	"github.com/bounoable/deepl"
)

// New returns a new DeepL translation service.
//
// Use WithClientOptions() to configure the *deepl.Client:
//	New("auth-key", WithClientOptions(deepl.BaseURL("https://example.com")))
//
// Use WithTranslateOptions() to append deepl.TranslateOptions to every request
// that is made through *Service.Translate():
//	New("auth-key", WithTranslateOptions(deepl.Formality(deepl.MoreFormal)))
func New(authKey string, opts ...Option) *Service {
	client := deepl.New(authKey)
	svc := NewWithClient(client, opts...)
	for _, opt := range svc.clientOpts {
		opt(client)
	}
	return svc
}

// NewWithClient does the same as New(), but accepts an existing *deepl.Client.
//
// WithClientOptions() has no effect in this case.
func NewWithClient(client Client, opts ...Option) *Service {
	svc := &Service{client: client}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// Option is a service option.
type Option func(*Service)

// WithClientOptions configures the created *deepl.Client.
func WithClientOptions(opts ...deepl.ClientOption) Option {
	return func(svc *Service) {
		svc.clientOpts = append(svc.clientOpts, opts...)
	}
}

// WithTranslateOptions adds translation options to every request.
func WithTranslateOptions(opts ...deepl.TranslateOption) Option {
	return func(svc *Service) {
		svc.translateOpts = append(svc.translateOpts, opts...)
	}
}

// Client is an interface for *deepl.Client.
type Client interface {
	Translate(
		ctx context.Context,
		text string,
		targetLang deepl.Language,
		opts ...deepl.TranslateOption,
	) (string, deepl.Language, error)
}

// Client returns the underlying *deepl.Client.
func (svc *Service) Client() Client {
	return svc.client
}

// Service is the DeepL translation service.
//
// It delegates translation requests to the underlying *deepl.Client (https://github.com/bounoable/deepl).
//
// The deepl.SourceLang() and deepl.PreserveFormatting() options will be used automatically.
type Service struct {
	client        Client
	clientOpts    []deepl.ClientOption
	translateOpts []deepl.TranslateOption
}

// Translate translates the given text from sourceLang to targetLang.
func (svc *Service) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	opts := append([]deepl.TranslateOption{
		deepl.SourceLang(deepl.Language(strings.ToUpper(sourceLang))),
		deepl.PreserveFormatting(true),
		deepl.SplitSentences(deepl.SplitNoNewlines),
	}, svc.translateOpts...)

	translated, _, err := svc.client.Translate(ctx, text, deepl.Language(strings.ToUpper(targetLang)), opts...)
	if err != nil {
		return translated, fmt.Errorf("deepl translate: %w", err)
	}

	return translated, nil
}
