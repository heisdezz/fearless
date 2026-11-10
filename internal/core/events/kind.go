// Package events defines the typed messages that flow from the application
// layer to the presentation layer.
//
// This package replaces Bruno's untyped `main:*` IPC channels. Every event is a
// value with a concrete type and a stable Kind, so a presenter's dispatch is a
// switch on an integer the compiler can check for completeness, rather than a
// chain of string comparisons against `event.type`.
//
// Rules (see docs/implementation/interface-design.md §4):
//
//   - Events are values. No event holds a pointer into core-owned mutable
//     state, so a consumer cannot mutate the producer's data by accident.
//     Response bodies travel as model.BodyRef, never as bytes.
//   - Every RequestQueued is followed by exactly one terminal event:
//     ResponseReceived or ErrorOccurred. This holds for cancellation too.
//   - Per-request events arrive in order. Global ordering is not guaranteed.
//
// This package depends only on model, which keeps it safe for the bus, the
// commands and the presenters to share.
package events

// Event is the interface every event satisfies.
//
// Kind returns a stable numeric tag. Using an integer rather than a marker
// method lets presenters dispatch on a switch and lets tests assert that every
// Kind has a handler.
type Event interface {
	Kind() Kind
}

// Kind identifies an event type. The zero value is deliberately invalid so an
// uninitialised Kind can never be mistaken for a real event.
type Kind uint8

const (
	// kindInvalid is the zero value; no event reports it.
	kindInvalid Kind = iota

	KindRequestQueued
	KindRequestSent
	KindResponseChunk
	KindResponseReceived
	KindTestResults
	KindAssertionResults
	KindUnresolvedVariables
	KindScriptedRequest
	KindCookiesUpdated
	KindCredentialsUpdated
	KindCollectionChanged
	KindScriptOutput
	KindErrorOccurred
)

// AllKinds lists every defined Kind, excluding the invalid zero value. Tests
// iterate this to assert that each Kind has a concrete event and a handler.
var AllKinds = []Kind{
	KindRequestQueued,
	KindRequestSent,
	KindResponseChunk,
	KindResponseReceived,
	KindTestResults,
	KindAssertionResults,
	KindUnresolvedVariables,
	KindScriptedRequest,
	KindCookiesUpdated,
	KindCredentialsUpdated,
	KindCollectionChanged,
	KindScriptOutput,
	KindErrorOccurred,
}

// Valid reports whether k is a defined Kind.
func (k Kind) Valid() bool { return k >= KindRequestQueued && k <= KindErrorOccurred }

// String returns a stable lower-case name, used in logs and test output.
func (k Kind) String() string {
	switch k {
	case KindRequestQueued:
		return "request-queued"
	case KindRequestSent:
		return "request-sent"
	case KindResponseChunk:
		return "response-chunk"
	case KindResponseReceived:
		return "response-received"
	case KindTestResults:
		return "test-results"
	case KindAssertionResults:
		return "assertion-results"
	case KindUnresolvedVariables:
		return "unresolved-variables"
	case KindScriptedRequest:
		return "scripted-request"
	case KindCookiesUpdated:
		return "cookies-updated"
	case KindCredentialsUpdated:
		return "credentials-updated"
	case KindCollectionChanged:
		return "collection-changed"
	case KindScriptOutput:
		return "script-output"
	case KindErrorOccurred:
		return "error-occurred"
	default:
		return "invalid"
	}
}

// IsTerminal reports whether k ends a request's event stream.
//
// The bus uses this to honour the back-pressure rule: chunk events may be
// dropped when a subscriber falls behind, terminal events never are.
func (k Kind) IsTerminal() bool {
	return k == KindResponseReceived || k == KindErrorOccurred
}

// IsDroppable reports whether k may be discarded under back-pressure.
//
// Only high-frequency, replaceable events qualify. Everything else represents a
// state transition the UI must not miss.
func (k Kind) IsDroppable() bool {
	return k == KindResponseChunk || k == KindScriptOutput
}
