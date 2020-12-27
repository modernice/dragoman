package gcloud

//go:generate mockgen -source=gcloud.go -destination=./mocks/gcloud.go

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	api "cloud.google.com/go/translate/apiv3"
	"github.com/googleapis/gax-go/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/translate/v3"
)

var (
	// ErrNoCredentials means no credentials are provided to initialize the translation service.
	ErrNoCredentials = errors.New("no credentials provided")
)

// Service is the Google Cloud Translate service.
type Service struct {
	mux            sync.RWMutex
	client         Client
	projectID      string
	scopes         []string
	tokenSource    oauth2.TokenSource
	newTokenSource func(context.Context, ...string) (oauth2.TokenSource, error)
	clientOpts     []option.ClientOption
	requestOpts    []func(*translate.TranslateTextRequest)
}

// Client is an interface for the Google Cloud Translate client.
type Client interface {
	TranslateText(context.Context, *translate.TranslateTextRequest, ...gax.CallOption) (*translate.TranslateTextResponse, error)
}

// Option is a Service option.
type Option func(*Service)

// SetupError is an error that occurred during the initial setup of the translation service.
type SetupError struct {
	Err error
}

// WithClientOptions returns an Option that adds custom option.ClientOptions that will be passed to the Client.
func WithClientOptions(opts ...option.ClientOption) Option {
	return func(svc *Service) {
		svc.clientOpts = append(svc.clientOpts, opts...)
	}
}

// WithRequestOptions returns an Option that modifies translation requests.
func WithRequestOptions(opts ...func(*translate.TranslateTextRequest)) Option {
	return func(svc *Service) {
		svc.requestOpts = append(svc.requestOpts, opts...)
	}
}

// WithTokenSource returns an Option that sets ts as the oauth2.TokenSource
// to be passed as an option.ClientOption to the Client.
//
// Using this option makes the following options no-ops:
// WithTokenSourceFactory(), CredentialsXXX()
func WithTokenSource(ts oauth2.TokenSource) Option {
	return func(svc *Service) {
		svc.tokenSource = ts
	}
}

// WithTokenSourceFactory returns an Option that sets newTokenSource to be used as the oauth2.TokenSource factory.
//
// Using this option makes the following options no-ops:
// CredentialsXXX()
func WithTokenSourceFactory(newTokenSource func(context.Context, ...string) (oauth2.TokenSource, error)) Option {
	return func(svc *Service) {
		svc.newTokenSource = newTokenSource
	}
}

// Scopes returns an Option that specifies the Cloud Translate scopes.
func Scopes(scopes ...string) Option {
	return func(svc *Service) {
		svc.scopes = append(svc.scopes, scopes...)
	}
}

// Credentials returns an Option that authenticates API calls using creds.
func Credentials(creds *google.Credentials) Option {
	return WithTokenSourceFactory(func(context.Context, ...string) (oauth2.TokenSource, error) {
		if creds == nil {
			return nil, errors.New("nil credentials")
		}
		return creds.TokenSource, nil
	})
}

// CredentialsJSON returns an Option that authenticates API calls using the credentials file in jsonKey.
func CredentialsJSON(jsonKey []byte) Option {
	return WithTokenSourceFactory(func(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
		cfg, err := google.JWTConfigFromJSON(jsonKey, scopes...)
		if err != nil {
			return nil, fmt.Errorf("parse credentials: %w", err)
		}
		return cfg.TokenSource(ctx), nil
	})
}

// CredentialsFile returns an Option that authenticates API calls using the credentials file at path.
func CredentialsFile(path string) Option {
	return WithTokenSourceFactory(func(ctx context.Context, scopes ...string) (oauth2.TokenSource, error) {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read credentials file %s: %w", path, err)
		}
		cfg, err := google.JWTConfigFromJSON(b, scopes...)
		if err != nil {
			return nil, fmt.Errorf("parse credentials: %w", err)
		}
		return cfg.TokenSource(ctx), nil
	})
}

// New creates a new Google Cloud Translator service. If no credentials are
// provided, the credentials file at the path in the environment variable
// CLOUD_TRANSLATE_CREDENTIALS is used.
func New(projectID string, opts ...Option) *Service {
	return newService(&Service{projectID: projectID}, opts...)
}

func newService(svc *Service, opts ...Option) *Service {
	if credsPath := os.Getenv("CLOUD_TRANSLATE_CREDENTIALS"); credsPath != "" {
		opts = append([]Option{CredentialsFile(credsPath)}, opts...)
	}
	for _, opt := range opts {
		opt(svc)
	}
	if len(svc.scopes) == 0 {
		svc.scopes = []string{"https://www.googleapis.com/auth/cloud-translation"}
	}
	return svc
}

// NewWithClient creates a new Google Cloud Translator service with a
// pre-initialized Client. If no credentials are provided, the credentials file
// at the path in the environment variable CLOUD_TRANSLATE_CREDENTIALS is used.
func NewWithClient(client Client, projectID string, opts ...Option) *Service {
	return newWithClient(&Service{client: client, projectID: projectID}, client, opts...)
}

func newWithClient(svc *Service, client Client, opts ...Option) *Service {
	if client == nil {
		panic("nil client")
	}
	svc = newService(svc, opts...)
	svc.client = client
	return svc
}

// NewFromCredentials creates a new Google Cloud Translator service using creds
// as the credentials and creds.ProjectID as the project id.
func NewFromCredentials(creds *google.Credentials, opts ...Option) *Service {
	opts = append([]Option{Credentials(creds)}, opts...)
	return New(creds.ProjectID, opts...)
}

func newFromCredentials(svc *Service, creds *google.Credentials, opts ...Option) *Service {
	opts = append([]Option{Credentials(creds)}, opts...)
	svc = newService(svc, opts...)
	svc.projectID = creds.ProjectID
	return svc
}

// NewFromCredentialsFile creates a new Google Cloud Translator service using
// the credentials in the file at path p.
func NewFromCredentialsFile(ctx context.Context, p string, opts ...Option) (*Service, error) {
	var svc Service
	for _, opt := range opts {
		opt(&svc)
	}

	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", p, err)
	}

	creds, err := google.CredentialsFromJSON(ctx, b, svc.scopes...)
	if err != nil {
		return nil, fmt.Errorf("obtain credentials: %w", err)
	}

	opts = append([]Option{CredentialsFile(p)}, opts...)

	return newFromCredentials(&svc, creds, opts...), nil
}

// Client returns the underlying Client.
func (svc *Service) Client() Client {
	return svc.client
}

// ProjectID returns the Google Cloud project id.
func (svc *Service) ProjectID() string {
	return svc.projectID
}

// Translate translates the given text from sourceLang to targetLang.
func (svc *Service) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	if err := svc.ensure(ctx); err != nil {
		return "", err
	}

	req := &translate.TranslateTextRequest{
		Parent:             fmt.Sprintf("projects/%s", svc.projectID),
		MimeType:           "text/html",
		SourceLanguageCode: sourceLang,
		TargetLanguageCode: targetLang,
		Contents:           []string{text},
	}

	for _, opt := range svc.requestOpts {
		opt(req)
	}

	resp, err := svc.client.TranslateText(ctx, req)
	if err != nil {
		return "", fmt.Errorf("cloud translate: %w", err)
	}

	trans := resp.GetTranslations()
	if len(trans) == 0 {
		return "", errors.New("cloud translate: no translations")
	}

	return trans[0].GetTranslatedText(), nil
}

func (svc *Service) ensure(ctx context.Context) error {
	if svc.initialized() {
		return nil
	}

	if err := svc.init(ctx); err != nil {
		return &SetupError{err}
	}

	return nil
}

func (svc *Service) initialized() bool {
	svc.mux.RLock()
	defer svc.mux.RUnlock()
	return svc.client != nil
}

func (svc *Service) init(ctx context.Context) error {
	svc.mux.Lock()
	defer svc.mux.Unlock()

	if svc.tokenSource == nil && svc.newTokenSource != nil {
		ts, err := svc.newTokenSource(ctx, svc.scopes...)
		if err != nil {
			return fmt.Errorf("new token source: %w", err)
		}
		svc.tokenSource = ts
	}

	if svc.tokenSource != nil {
		client, err := svc.newClient(ctx, svc.tokenSource, svc.clientOpts...)
		if err != nil {
			return fmt.Errorf("new client: %w", err)
		}
		svc.client = client
	}

	if svc.client == nil {
		return ErrNoCredentials
	}

	return nil
}

func (svc *Service) newClient(ctx context.Context, ts oauth2.TokenSource, opts ...option.ClientOption) (Client, error) {
	opts = append([]option.ClientOption{option.WithTokenSource(ts)}, opts...)
	client, err := api.NewTranslationClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("new translation client: %w", err)
	}
	return client, nil
}

func (err *SetupError) Unwrap() error {
	return err.Err
}

func (err *SetupError) Error() string {
	return fmt.Sprintf("setup: %s", err.Err.Error())
}
