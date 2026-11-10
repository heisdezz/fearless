package events

import (
	"time"

	"fearless/internal/core/model"
)

// RequestQueued announces that a request has been accepted for execution.
//
// It is published before any network I/O so the UI can show a pending state and
// so the request UID is reserved: every later event for this execution carries
// the same UID.
//
// Exactly one terminal event (ResponseReceived or ErrorOccurred) follows.
type RequestQueued struct {
	// UID identifies this execution. It is not the item UID: the same item can
	// be sent repeatedly, and each execution is tracked separately.
	UID string
	// ItemUID is the collection item being sent.
	ItemUID string
	// CollectionUID scopes the item.
	CollectionUID string
	// Environment names the active environment, or "" for none.
	Environment string
	// QueuedAt is when the command accepted the request.
	QueuedAt time.Time
}

// Kind implements Event.
func (RequestQueued) Kind() Kind { return KindRequestQueued }

// RequestSent carries the request exactly as it will go on the wire, after
// variable interpolation and auth application but before transmission.
//
// This is what the UI's "request" view shows and what scripts observe. It may be
// published more than once for a single execution: a redirect hop or an auth
// retry re-sends with a modified request, and each transmission is reported.
type RequestSent struct {
	UID string
	// Attempt counts transmissions for this execution, starting at 1. A value
	// greater than 1 means a redirect or an auth challenge caused a resend.
	Sent model.SentRequest
	// Attempt is 1 for the initial transmission.
	Attempt int
	// Reason explains a resend: "" for the first attempt, otherwise one of
	// "redirect" or "auth".
	Reason string
}

// Kind implements Event.
func (RequestSent) Kind() Kind { return KindRequestSent }

// Reasons a RequestSent may be a resend.
const (
	// ResendRedirect marks a transmission following an HTTP redirect.
	ResendRedirect = "redirect"
	// ResendAuth marks a transmission following an authentication challenge.
	ResendAuth = "auth"
)

// ResponseChunk is a fragment of a streaming response body (SSE, chunked, or an
// incremental download).
//
// Chunks are droppable: a slow subscriber loses fragments rather than blocking
// the producer. A consumer that needs the complete body must read it from the
// BodyRef on ResponseReceived instead of concatenating chunks.
type ResponseChunk struct {
	UID string
	// Data is the fragment. It must not be retained beyond the handler call;
	// the slice may be reused by the producer.
	Data []byte
	// Seq is a monotonically increasing fragment number, so a consumer can
	// detect a dropped chunk by a gap.
	Seq uint64
}

// Kind implements Event.
func (ResponseChunk) Kind() Kind { return KindResponseChunk }

// ResponseReceived is the terminal event for a successful exchange.
//
// It carries a Result whose body is a BodyRef, so no large payload crosses the
// event stream. A Result with a non-empty Err still arrives as
// ResponseReceived when the HTTP exchange completed: an error status such as
// 500 is a result, not a failure.
type ResponseReceived struct {
	UID      string
	ItemUID  string
	Received model.Result
}

// Kind implements Event.
func (ResponseReceived) Kind() Kind { return KindResponseReceived }

// UnresolvedVariables reports template names that could not be resolved.
//
// It is published before sending. Depending on configuration the command either
// aborts (followed by ErrorOccurred) or sends with the placeholders left intact.
//
// Secret values never appear here; only names are reported.
type UnresolvedVariables struct {
	UID string
	// Names lists the unresolved variable names, deduplicated and in the order
	// first encountered.
	Names []string
	// Aborted reports whether the send was cancelled because of these.
	Aborted bool
}

// Kind implements Event.
func (UnresolvedVariables) Kind() Kind { return KindUnresolvedVariables }

// ScriptedRequest reports a request issued from inside a script via
// bru.sendRequest or bru.runRequest.
//
// The parent execution is the script's request; ChildUID identifies the nested
// one, which receives its own RequestQueued and terminal events.
type ScriptedRequest struct {
	// ParentUID is the execution whose script issued the nested request.
	ParentUID string
	// ChildUID is the execution UID reserved for the nested request.
	ChildUID string
	// Source names the API used: "sendRequest" or "runRequest".
	Source string
}

// Kind implements Event.
func (ScriptedRequest) Kind() Kind { return KindScriptedRequest }

// Script output levels.
const (
	LevelLog   = "log"
	LevelWarn  = "warn"
	LevelError = "error"
)

// ScriptOutput is a line written by a script's console.
//
// Like ResponseChunk it is droppable: the console must never stall execution.
type ScriptOutput struct {
	UID string
	// Level is one of "log", "warn" or "error".
	Level string
	// Text is the rendered message.
	Text string
	// At is when the script wrote it.
	At time.Time
}

// Kind implements Event.
func (ScriptOutput) Kind() Kind { return KindScriptOutput }
