package engine

import (
	"context"
	"net/url"

	"fearless/internal/core/cookies"
	enginetls "fearless/internal/core/engine/tls"
	"fearless/internal/core/model"
	"fearless/internal/core/persistence"
	"fearless/internal/core/vars"
)

// Engine executes HTTP requests through the full lifecycle pipeline.
//
// The engine is the sole network boundary and must remain completely decoupled
// from any GUI or image packages. Both the headless CLI and the GUI application
// invoke requests exclusively through this interface.
type Engine interface {
	Do(ctx context.Context, in Input) (*model.Result, error)
}

// Input gathers everything needed to execute a request.
type Input struct {
	Item               *model.Item
	Collection         *model.Collection
	Ancestors          []*model.Item // enclosing folders outermost to innermost
	Scopes             vars.Scopes   // resolved variable scopes
	Settings           model.ItemSettings
	GlobalPreferences  model.Settings // application preferences for defaults
	Hooks              HookSet
	Sink               EventSink
	Jar                *cookies.Jar           // optional shared cookie jar
	Spooler            *persistence.Spooler   // optional body spooler
	ProxyURL           *url.URL               // optional resolved proxy
	TLS                enginetls.BuildOptions // optional TLS configuration
	CollectionPath     string                 // root collection directory path
	UserAgent          string                 // optional custom User-Agent
	DisableCompression bool
}

// Option configures an Engine instance.
type Option func(*defaultEngine)

// WithDefaultUserAgent configures the default User-Agent string for the engine.
func WithDefaultUserAgent(ua string) Option {
	return func(e *defaultEngine) {
		e.defaultUserAgent = ua
	}
}

// WithDefaultSpooler configures a default disk spooler for large responses.
func WithDefaultSpooler(s *persistence.Spooler) Option {
	return func(e *defaultEngine) {
		e.defaultSpooler = s
	}
}

// WithDefaultJar configures a default cookie jar for the engine.
func WithDefaultJar(jar *cookies.Jar) Option {
	return func(e *defaultEngine) {
		e.defaultJar = jar
	}
}

// New creates an Engine instance configured with the provided options.
func New(opts ...Option) Engine {
	e := &defaultEngine{}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

type defaultEngine struct {
	defaultUserAgent string
	defaultSpooler   *persistence.Spooler
	defaultJar       *cookies.Jar
}

// Do executes a request through the full lifecycle pipeline.
func (e *defaultEngine) Do(ctx context.Context, in Input) (*model.Result, error) {
	if in.UserAgent == "" && e.defaultUserAgent != "" {
		in.UserAgent = e.defaultUserAgent
	}
	if in.Spooler == nil && e.defaultSpooler != nil {
		in.Spooler = e.defaultSpooler
	}
	if in.Jar == nil && e.defaultJar != nil {
		in.Jar = e.defaultJar
	}
	return runPipeline(ctx, in)
}
