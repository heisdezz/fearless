package build

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"fearless/internal/core/model"
)

// HeaderEntry represents a single HTTP header preserving original key casing and order.
type HeaderEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// BuildInput encapsulates everything required to prepare and synthesize an outbound HTTP request.
type BuildInput struct {
	// Item is the request item being executed (draft or saved).
	Item *model.Item

	// Collection is the parent collection, used for ancestor inheritance and relative paths.
	Collection *model.Collection

	// Variables represents the merged active variable scope (environment, collection, runtime, prompt, etc.).
	Variables map[string]any

	// CollectionPath is the base filesystem directory for resolving relative file bodies.
	CollectionPath string

	// Settings overrides or defaults item settings.
	Settings *model.ItemSettings
}

// BuiltRequest represents the fully interpolated, synthesized, and prepared outbound request.
type BuiltRequest struct {
	// Method is the HTTP method (e.g. GET, POST).
	Method string

	// URL is the parsed and fully synthesized target URL.
	URL *url.URL

	// RawURL is the final URL string before parsing.
	RawURL string

	// Headers is the standard Go http.Header map.
	Headers http.Header

	// HeaderEntries preserves exact header casing and ordering (allowing duplicates).
	HeaderEntries []HeaderEntry

	// Body is the rendered request body bytes.
	Body []byte

	// BodyReader provides an io.ReadCloser for the body payload (or nil if body mode is none).
	BodyReader io.ReadCloser

	// ContentLength is the calculated byte length of the body.
	ContentLength int64

	// ContentType is the determined Content-Type header value.
	ContentType string

	// UnresolvedVars lists all variable placeholders that could not be resolved during interpolation.
	UnresolvedVars []string

	// HTTPRequest is a standard net/http.Request ready for dispatch.
	HTTPRequest *http.Request
}

// Builder defines the interface for preparing an executable request.
type Builder interface {
	Build(ctx context.Context, in BuildInput) (*BuiltRequest, error)
}
