package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	"fearless/internal/core/model"
)

// doHop sends one request and collects its body.
func doHop(ctx context.Context, client *http.Client, req *http.Request, opts Options, tl *timeline) (*hopResult, error) {
	// The trace must be attached to this hop's context, not the caller's, so a
	// redirect's phases do not merge with the first hop's.
	traceCtx := httptrace.WithClientTrace(ctx, newTrace(tl))
	req = req.WithContext(traceCtx)

	tl.begin(model.PhaseRequestSent)

	// Cookies are applied per hop, not through http.Client.Jar: our Jar exposes an
	// explicit API (HeaderFor/AddRaw) rather than net/http's CookieJar interface,
	// which lets it apply Bruno's trustworthy-origin rule and record every change.
	if opts.Jar != nil {
		if hdr, cerr := opts.Jar.HeaderFor(req.URL); cerr == nil && hdr != "" {
			req.Header.Set("Cookie", mergeCookieHeader(req.Header.Get("Cookie"), hdr))
		}
	}

	resp, err := client.Do(req)
	// Trace callbacks ran while Do was in flight; apply them now, before the
	// logical phases that follow, so the recorded order matches reality.
	tl.drain()
	if err != nil {
		tl.end(model.PhaseRequestSent, err.Error())
		return nil, classifyError(err)
	}
	defer resp.Body.Close()

	// Store any cookies the server set before the body is read, so a redirect hop
	// sees them.
	if opts.Jar != nil {
		for _, sc := range resp.Header.Values("Set-Cookie") {
			_ = opts.Jar.AddRaw(sc, req.URL)
		}
	}

	// The send phase ends when the response headers arrive; the download phase
	// that follows measures the body.
	tl.end(model.PhaseRequestSent, "")
	tl.end(model.PhaseWaiting, "")

	out := &hopResult{
		status:     resp.StatusCode,
		statusText: resp.Status,
	}
	for name, values := range resp.Header {
		for _, v := range values {
			n, val := name, v
			out.headers = append(out.headers, model.KeyValue{
				Name: &n, Value: &val, Enabled: true,
			})
		}
	}

	encoding := resp.Header.Get("Content-Encoding")
	contentType := resp.Header.Get("Content-Type")

	// Non-followed redirects and 204/304 have no body worth spooling.
	if resp.Body == nil || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotModified {
		out.bodyRef = model.BodyRef{ContentType: contentType}
		return out, nil
	}

	tl.begin(model.PhaseDownload)
	ref, serr := spoolBody(resp.Body, opts, contentType, encoding)
	tl.end(model.PhaseDownload, "")
	out.bodyRef = ref
	if serr != nil {
		return nil, serr
	}
	return out, nil
}

// mergeCookieHeader combines an explicit Cookie header with jar cookies.
//
// An explicit header wins only for the names it already sets, which is what
// CookieJar.MergeHeaderFor does inside the jar; doing it here keeps a user-set
// cookie from being silently replaced by a stale one.
func mergeCookieHeader(explicit, fromJar string) string {
	if explicit == "" {
		return fromJar
	}
	if fromJar == "" {
		return explicit
	}
	set := make(map[string]bool)
	for _, part := range strings.Split(explicit, ";") {
		if kv := strings.SplitN(strings.TrimSpace(part), "=", 2); len(kv) > 0 {
			set[kv[0]] = true
		}
	}
	var extra []string
	for _, part := range strings.Split(fromJar, ";") {
		name := strings.TrimSpace(part)
		if kv := strings.SplitN(name, "=", 2); len(kv) > 0 {
			name = kv[0]
		}
		if !set[name] {
			extra = append(extra, strings.TrimSpace(part))
		}
	}
	if len(extra) == 0 {
		return explicit
	}
	return explicit + "; " + strings.Join(extra, "; ")
}

// spoolBody streams the body to the spooler, emitting chunks as they arrive.
//
// A tee keeps the streaming behaviour without buffering the whole body: the
// spooler owns the bytes, the sink sees fragments.
func spoolBody(body io.Reader, opts Options, contentType, encoding string) (model.BodyRef, error) {
	reader := &chunkReader{r: body, sink: opts.Sink}
	if opts.Spooler == nil {
		// No spooler: read into memory so the caller still gets a body. This is the
		// path used by unit tests that do not set up persistence.
		data, err := io.ReadAll(io.LimitReader(reader, opts.MaxResponseBytes))
		if err != nil {
			return model.BodyRef{}, err
		}
		return model.BodyRef{
			Size:        int64(len(data)),
			ContentType: contentType,
			Encoding:    encoding,
			Inline:      data,
		}, nil
	}
	return opts.Spooler.Spool(reader, contentType, encoding, opts.MaxResponseBytes)
}

// chunkReader forwards read data to the sink.
type chunkReader struct {
	r    io.Reader
	sink Sink
}

// Read implements io.Reader.
func (c *chunkReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.sink.Chunk(p[:n])
	}
	return n, err
}

// classifyError turns a transport error into a message that names the cause.
//
// The categories match the events package's error classes, so the UI can render
// a timeout differently from a DNS failure without string matching.
func classifyError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("request cancelled: %w", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("request timed out: %w", err)
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return fmt.Errorf("%s: %w", urlErr.Op, urlErr.Err)
	}
	return err
}

// redirectRequest builds the next hop.
//
// Method rewriting follows RFC 7231 §6.4, which is what Bruno does:
//
//	301/302 → GET for POST (and drop the body)
//	303     → GET always
//	307/308 → preserve method and body
//
// Only a HEAD request keeps HEAD on a 303.
func redirectRequest(prev *http.Request, hop *hopResult, location string, opts Options) (*http.Request, error) {
	base := prev.URL
	ref, err := url.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect location %q: %w", location, err)
	}
	target := base.ResolveReference(ref)

	method := prev.Method
	dropBody := false
	switch hop.status {
	case http.StatusMovedPermanently, http.StatusFound:
		if method == http.MethodPost {
			method = http.MethodGet
			dropBody = true
		}
	case http.StatusSeeOther:
		if method != http.MethodHead {
			method = http.MethodGet
			dropBody = true
		}
	case http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		// Preserve method and body.
	}

	var body io.Reader
	if !dropBody {
		// The previous body has been consumed; ask the caller for a fresh one.
		if resetter, ok := prev.Body.(interface{ Reset() (io.ReadCloser, error) }); ok {
			fresh, rerr := resetter.Reset()
			if rerr != nil {
				return nil, fmt.Errorf("redirect body could not be replayed: %w", rerr)
			}
			body = fresh
		}
	}

	next, err := http.NewRequestWithContext(prev.Context(), method, target.String(), body)
	if err != nil {
		return nil, err
	}
	next.Header = cloneHeader(prev.Header)

	// A cross-host redirect must not carry credentials unless explicitly allowed.
	if !opts.ForwardAuthHeader && !sameHost(base, target) {
		next.Header.Del("Authorization")
		next.Header.Del("Cookie")
	}
	if dropBody {
		next.Header.Del("Content-Type")
		next.Header.Del("Content-Length")
		next.ContentLength = 0
	}
	return next, nil
}

// sameHost reports whether two URLs share a host and port.
func sameHost(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Host, b.Host)
}

// cloneHeader copies a header map.
func cloneHeader(h http.Header) http.Header {
	out := make(http.Header, len(h))
	for k, v := range h {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// modelPhaseDNS and friends alias the model phase constants used by the trace
// callbacks, which run in client.go where importing model for two identifiers
// would be noise.
const (
	modelPhaseDNS     = model.PhaseDNS
	modelPhaseConnect = model.PhaseConnect
	modelPhaseTLS     = model.PhaseTLS
	modelPhaseWaiting = model.PhaseWaiting
)

// traceEvent is one httptrace callback, delivered through a channel.
type traceEvent struct {
	phase  model.TimelinePhase
	begin  bool
	detail string
}

// timeline records phases and forwards completed ones to the sink.
type timeline struct {
	entries_ []model.TimelineEntry
	sink     Sink
	open     map[model.TimelinePhase]time.Time
	// trace carries events from httptrace callbacks, which run on connection
	// goroutines and therefore cannot write here directly.
	trace chan traceEvent
}

// newTimeline creates a timeline that reports to sink.
func newTimeline(sink Sink) *timeline {
	return &timeline{
		sink:  sink,
		open:  make(map[model.TimelinePhase]time.Time),
		trace: make(chan traceEvent, 16),
	}
}

// drain applies any trace events that arrived during the hop.
func (t *timeline) drain() {
	if t.trace == nil {
		return
	}
	for {
		select {
		case ev := <-t.trace:
			if ev.begin {
				t.begin(ev.phase)
			} else {
				t.end(ev.phase, ev.detail)
			}
		default:
			return
		}
	}
}

// begin opens a phase.
func (t *timeline) begin(phase model.TimelinePhase) {
	t.open[phase] = time.Now()
}

// end closes a phase and records it.
func (t *timeline) end(phase model.TimelinePhase, detail string) {
	start, ok := t.open[phase]
	if !ok {
		start = time.Now()
	}
	delete(t.open, phase)
	entry := model.TimelineEntry{
		Phase:    phase,
		Started:  start,
		Duration: time.Since(start),
		Detail:   detail,
	}
	t.entries_ = append(t.entries_, entry)
	t.sink.Phase(entry)
}

// mark records an instantaneous phase.
func (t *timeline) mark(phase model.TimelinePhase, detail string) {
	now := time.Now()
	entry := model.TimelineEntry{Phase: phase, Started: now, Duration: 0, Detail: detail}
	t.entries_ = append(t.entries_, entry)
	t.sink.Phase(entry)
}

// entries returns the recorded phases.
func (t *timeline) entries() model.Timeline {
	return model.Timeline(t.entries_)
}
