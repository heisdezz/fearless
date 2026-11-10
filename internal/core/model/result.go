package model

import "time"

// BodyRef points at response bytes without holding them.
//
// Small bodies are inlined; large ones live in the spool and are referenced by
// Path. This is what keeps large responses out of application state — Bruno
// buffers the whole body in memory and then copies it across IPC.
type BodyRef struct {
	// ID is the content hash, so identical bodies deduplicate.
	ID string `json:"id"`
	// Size is the body length in bytes.
	Size int64 `json:"size"`
	// ContentType is the media type as reported or sniffed.
	ContentType string `json:"contentType,omitempty"`
	// Encoding is the transferred encoding, if any ("gzip", "br", …).
	Encoding string `json:"encoding,omitempty"`
	// Inline holds the bytes when the body is small enough; nil when spilled.
	Inline []byte `json:"inline,omitempty"`
	// Path is the spool file path when the body was spilled.
	Path string `json:"path,omitempty"`
	// Truncated reports that the body was cut off by the size cap.
	Truncated bool `json:"truncated,omitempty"`
}

// Spilled reports whether the body lives on disk.
func (b BodyRef) Spilled() bool { return b.Path != "" }

// TimelinePhase names one stage of request execution.
type TimelinePhase string

const (
	PhaseQueued       TimelinePhase = "queued"
	PhasePreRequest   TimelinePhase = "pre-request"
	PhaseVariables    TimelinePhase = "variables"
	PhaseAuth         TimelinePhase = "auth"
	PhaseDNS          TimelinePhase = "dns"
	PhaseConnect      TimelinePhase = "connect"
	PhaseTLS          TimelinePhase = "tls"
	PhaseRequestSent  TimelinePhase = "request-sent"
	PhaseWaiting      TimelinePhase = "waiting"
	PhaseDownload     TimelinePhase = "download"
	PhasePostResponse TimelinePhase = "post-response"
	PhaseTests        TimelinePhase = "tests"
)

// TimelineEntry records when a phase started and how long it took.
type TimelineEntry struct {
	Phase    TimelinePhase `json:"phase"`
	Started  time.Time     `json:"started"`
	Duration time.Duration `json:"duration"`
	Detail   string        `json:"detail,omitempty"`
}

// Timeline is the ordered phase list for one request.
type Timeline []TimelineEntry

// Total is the wall-clock span from the first start to the last end.
func (t Timeline) Total() time.Duration {
	if len(t) == 0 {
		return 0
	}
	var end time.Time
	start := t[0].Started
	for _, e := range t {
		if e.Started.Before(start) {
			start = e.Started
		}
		if e := e.Started.Add(e.Duration); e.After(end) {
			end = e
		}
	}
	return end.Sub(start)
}

// SentRequest is the request as it actually went on the wire, after variable
// resolution and auth application. It is what the UI shows and what scripts see.
type SentRequest struct {
	Method  string     `json:"method"`
	URL     string     `json:"url"`
	Headers []KeyValue `json:"headers"`
	Body    []byte     `json:"body,omitempty"`
	BodyRef *BodyRef   `json:"bodyRef,omitempty"`
}

// Header returns the first header value with the given name, case-insensitively.
func (s *SentRequest) Header(name string) (string, bool) {
	for _, h := range s.Headers {
		if equalFoldASCII(h.NameOrEmpty(), name) {
			return h.ValueOrEmpty(), true
		}
	}
	return "", false
}

// ResponseHeaders is the received header set, preserving order and duplicates
// (Set-Cookie repeats and must not be collapsed into a map).
type ResponseHeaders []KeyValue

// Values returns every value for name, case-insensitively.
func (h ResponseHeaders) Values(name string) []string {
	var out []string
	for _, kv := range h {
		if equalFoldASCII(kv.NameOrEmpty(), name) {
			out = append(out, kv.ValueOrEmpty())
		}
	}
	return out
}

// Result is the outcome of one request execution.
type Result struct {
	// Status is the HTTP status code; 0 when the request never completed.
	Status int `json:"status"`
	// StatusText is the reason phrase.
	StatusText string `json:"statusText,omitempty"`
	// Headers are the response headers in received order.
	Headers ResponseHeaders `json:"headers,omitempty"`
	// Body is where the payload lives.
	Body BodyRef `json:"body"`
	// Sent is the request as transmitted.
	Sent SentRequest `json:"sent"`
	// Timeline is the phase log.
	Timeline Timeline `json:"timeline,omitempty"`
	// Duration is the total time.
	Duration time.Duration `json:"duration"`
	// ReceivedAt is when the response completed.
	ReceivedAt time.Time `json:"receivedAt"`
	// Err is set when the exchange failed; Status may still be populated.
	Err string `json:"error,omitempty"`
	// Redirects records each followed hop.
	Redirects []Redirect `json:"redirects,omitempty"`
}

// Redirect is one followed redirect hop.
type Redirect struct {
	Status   int    `json:"status"`
	Location string `json:"location"`
	Method   string `json:"method"`
}

// Failed reports whether the exchange ended in a transport error.
func (r *Result) Failed() bool { return r != nil && r.Err != "" }

// StatusClass returns the status in hundreds (2 for 200–299), or 0.
func (r *Result) StatusClass() int {
	if r == nil || r.Status < 100 {
		return 0
	}
	return r.Status / 100
}

// TestResult is one `test(...)` outcome.
type TestResult struct {
	UID         string        `json:"uid"`
	Description string        `json:"description"`
	Status      string        `json:"status"` // pass | fail | skip | error
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
}

// AssertionResult is one `assert` outcome.
type AssertionResult struct {
	UID        string `json:"uid"`
	Expression string `json:"expression"`
	Operator   string `json:"operator,omitempty"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	Passed     bool   `json:"passed"`
	Error      string `json:"error,omitempty"`
}

// equalFoldASCII compares ASCII strings case-insensitively. Header names are
// ASCII by specification, so this avoids pulling in unicode folding.
func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
