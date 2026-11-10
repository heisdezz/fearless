package engine

import (
	"context"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

// HookSet gathers the optional extension hooks for request execution.
//
// Hooks allow scripting, assertions, and tests to hook into the lifecycle
// at their prescribed phases without coupling the core engine to a specific
// script engine (such as goja).
type HookSet struct {
	// PreRequest hooks run before request build/interpolation.
	// A failure here halts execution before any bytes are transmitted.
	PreRequest []PreRequestHook

	// PostResponse hooks run after the response has been received.
	PostResponse []PostResponseHook

	// Assertions run after post-response scripts to check assertions.
	Assertions []AssertHook

	// Tests run after assertions to execute tests.
	Tests []TestHook
}

// PreRequestHook is executed before building and transmitting a request.
type PreRequestHook interface {
	RunPreRequest(ctx context.Context, item *model.Item, scopes *vars.Scopes, sink EventSink) error
}

// PreRequestFunc is an adapter allowing a plain function to act as a PreRequestHook.
type PreRequestFunc func(ctx context.Context, item *model.Item, scopes *vars.Scopes, sink EventSink) error

func (f PreRequestFunc) RunPreRequest(ctx context.Context, item *model.Item, scopes *vars.Scopes, sink EventSink) error {
	return f(ctx, item, scopes, sink)
}

// PostResponseHook is executed after the response has been received.
type PostResponseHook interface {
	RunPostResponse(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink EventSink) error
}

// PostResponseFunc is an adapter allowing a plain function to act as a PostResponseHook.
type PostResponseFunc func(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink EventSink) error

func (f PostResponseFunc) RunPostResponse(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink EventSink) error {
	return f(ctx, item, res, scopes, sink)
}

// AssertHook executes assertions against the completed response.
type AssertHook interface {
	RunAssertions(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink EventSink) (*events.AssertionResults, error)
}

// AssertFunc is an adapter allowing a plain function to act as an AssertHook.
type AssertFunc func(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink EventSink) (*events.AssertionResults, error)

func (f AssertFunc) RunAssertions(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink EventSink) (*events.AssertionResults, error) {
	return f(ctx, item, res, scopes, sink)
}

// TestHook executes test blocks against the completed response.
type TestHook interface {
	RunTests(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink EventSink) (*events.TestResults, error)
}

// TestFunc is an adapter allowing a plain function to act as a TestHook.
type TestFunc func(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink EventSink) (*events.TestResults, error)

func (f TestFunc) RunTests(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink EventSink) (*events.TestResults, error) {
	return f(ctx, item, res, scopes, sink)
}
