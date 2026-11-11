// Package transport executes HTTP requests.
//
// It owns the parts of the request lifecycle that Bruno implements with axios,
// custom agents and a manual redirect loop:
//
//   - building the client (proxy, TLS, cookies, decompression)
//   - following redirects by hand, so each hop is recorded and each hop gets the
//     right cookies and headers
//   - recording a phase timeline (DNS, connect, TLS, first byte, download)
//   - streaming chunks to a sink while still producing a complete Result
//   - honouring cancellation and timeouts
//
// The package is GUI-free: it takes a fully-built *http.Request plus Options and
// returns a model.Result. See docs/logic/http-engine.md and
// docs/logic/redirects.md for the behaviour this reproduces.
package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"time"

	"fearless/internal/core/cookies"
	enginetls "fearless/internal/core/engine/tls"
	"fearless/internal/core/model"
	"fearless/internal/core/persistence"
)

// Sink receives progressive events while a request runs.
//
// It is an interface rather than a channel so the ENGINE decides what becomes an
// application event; the transport never publishes directly.
type Sink interface {
	// Chunk receives a fragment of the response body as it arrives.
	Chunk(data []byte)
	// Phase records a completed timeline phase.
	Phase(entry model.TimelineEntry)
}

// NopSink discards progressive events.
type NopSink struct{}

// Chunk implements Sink.
func (NopSink) Chunk([]byte) {}

// Phase implements Sink.
func (NopSink) Phase(model.TimelineEntry) {}

// Options configure one execution.
type Options struct {
	// FollowRedirects follows 3xx responses.
	FollowRedirects bool
	// MaxRedirects bounds the redirect chain.
	MaxRedirects int
	// TimeoutMs is the whole-request timeout; 0 means none.
	TimeoutMs int
	// ForwardAuthHeader keeps the Authorization header on a cross-host redirect.
	ForwardAuthHeader bool

	// Proxy is the resolved proxy URL, or nil for a direct connection.
	Proxy *url.URL
	// TLS configures verification and client certificates.
	TLS enginetls.BuildOptions
	// TLSConfig, when non-nil, is the fully resolved TLS configuration supplied
	// by the engine (CA bundles, client certificates, SNI). It takes precedence
	// over the verification switches derived from TLS.
	TLSConfig *tls.Config
	// WrapTransport, when non-nil, wraps the round tripper built by the
	// transport (e.g. for Digest/NTLM challenge-response handshakes). The wrapper
	// is applied once, around the whole client.
	WrapTransport func(http.RoundTripper) http.RoundTripper
	// Jar is the shared cookie jar; nil disables cookie handling.
	Jar *cookies.Jar
	// Spooler stores response bodies.
	Spooler *persistence.Spooler
	// Sink receives chunks and phases.
	Sink Sink
	// UserAgent overrides the default User-Agent.
	UserAgent string
	// MaxResponseBytes caps the body size; 0 uses DefaultMaxResponseBytes.
	MaxResponseBytes int64
	// DisableCompression turns off transparent gzip/br decoding.
	DisableCompression bool
}

// DefaultMaxResponseBytes caps an unbounded download at 512 MiB.
//
// Bruno has no cap and buffers whatever arrives, so a mistyped large URL can
// exhaust memory. The cap plus spooling is a deliberate improvement, recorded in
// research/limitations.md §2.1.
const DefaultMaxResponseBytes int64 = 512 << 20

// DefaultUserAgent identifies the client. Bruno sends `bruno-runtime/<version>`.
const DefaultUserAgent = "fearless/0.1"

// Execute performs the request and returns the complete result.
//
// A transport failure is reported through Result.Err rather than as a returned
// error: the caller always gets a Result so the UI can show what was attempted.
// The error return is reserved for programming errors (a nil request).
func Execute(ctx context.Context, req *http.Request, opts Options) (*model.Result, error) {
	if req == nil {
		return nil, errors.New("transport: nil request")
	}
	if opts.Sink == nil {
		opts.Sink = NopSink{}
	}
	if opts.MaxResponseBytes <= 0 {
		opts.MaxResponseBytes = DefaultMaxResponseBytes
	}

	res := &model.Result{ReceivedAt: time.Now().UTC()}
	tl := newTimeline(opts.Sink)
	start := time.Now()

	client, err := newClient(opts, tl)
	if err != nil {
		res.Err = err.Error()
		res.Duration = time.Since(start)
		return res, nil
	}

	current := req
	// The first hop's request is reported as sent; later hops are redirects.
	res.Sent = sentFromRequest(current)

	maxHops := opts.MaxRedirects
	if maxHops < 0 {
		maxHops = 0
	}

	for hop := 0; ; hop++ {
		hopRes, herr := doHop(ctx, client, current, opts, tl)
		if herr != nil {
			res.Err = herr.Error()
			break
		}

		if !opts.FollowRedirects || !isRedirect(hopRes.status) || hop >= maxHops {
			applyHop(res, hopRes)
			break
		}

		location := hopRes.header("Location")
		if location == "" {
			// A redirect status without Location is not a redirect.
			applyHop(res, hopRes)
			break
		}

		next, rerr := redirectRequest(current, hopRes, location, opts)
		if rerr != nil {
			res.Err = rerr.Error()
			break
		}
		res.Redirects = append(res.Redirects, model.Redirect{
			Status:   hopRes.status,
			Location: location,
			Method:   next.Method,
		})
		current = next
	}

	res.Duration = time.Since(start)
	res.Timeline = tl.entries()
	return res, nil
}

// applyHop copies a completed hop onto the result.
func applyHop(res *model.Result, hop *hopResult) {
	res.Status = hop.status
	res.StatusText = hop.statusText
	res.Headers = hop.headers
	res.Body = hop.bodyRef
}

// hopResult is one response after its body has been collected.
type hopResult struct {
	status     int
	statusText string
	headers    model.ResponseHeaders
	bodyRef    model.BodyRef
}

// header returns the first value of a header, case-insensitively.
func (h *hopResult) header(name string) string {
	if v := h.headers.Values(name); len(v) > 0 {
		return v[0]
	}
	return ""
}

// isRedirect reports whether a status code is a redirect this package follows.
func isRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}

// sentFromRequest captures the request as the UI and scripts see it.
//
// The body is not reproduced here: the transport receives a *http.Request whose
// body may be a stream, and reading it would consume it.
func sentFromRequest(req *http.Request) model.SentRequest {
	sent := model.SentRequest{Method: req.Method}
	if req.URL != nil {
		sent.URL = req.URL.String()
	}
	for name, values := range req.Header {
		for _, v := range values {
			n, val := name, v
			sent.Headers = append(sent.Headers, model.KeyValue{
				Name: &n, Value: &val, Enabled: true,
			})
		}
	}
	return sent
}
