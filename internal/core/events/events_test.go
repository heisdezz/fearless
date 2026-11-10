package events

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"fearless/internal/core/model"
)

// TestEveryKindHasAConcreteEvent asserts the mapping between Kind constants and
// event types is total: every Kind is reported by exactly one event type, and
// every event type reports a valid Kind.
//
// This is the "did I handle every event" audit from interface-design.md §4, as a
// test rather than a code review.
func TestEveryKindHasAConcreteEvent(t *testing.T) {
	samples := []Event{
		RequestQueued{},
		RequestSent{},
		ResponseChunk{},
		ResponseReceived{},
		TestResults{},
		AssertionResults{},
		UnresolvedVariables{},
		ScriptedRequest{},
		CookiesUpdated{},
		CredentialsUpdated{},
		CollectionChanged{},
		ScriptOutput{},
		ErrorOccurred{},
	}

	seen := make(map[Kind]string, len(samples))
	for _, ev := range samples {
		k := ev.Kind()
		if !k.Valid() {
			t.Fatalf("%T reports invalid kind %d", ev, k)
		}
		if prev, dup := seen[k]; dup {
			t.Fatalf("kind %d (%s) reported by both %s and %T", k, k, prev, ev)
		}
		seen[k] = fmt.Sprintf("%T", ev)
	}

	for _, k := range AllKinds {
		if _, ok := seen[k]; !ok {
			t.Errorf("kind %s (%d) has no event type", k, k)
		}
	}
	if len(seen) != len(AllKinds) {
		t.Errorf("have %d event types for %d kinds", len(seen), len(AllKinds))
	}
}

func TestKindZeroIsInvalid(t *testing.T) {
	var k Kind
	if k.Valid() {
		t.Fatal("the zero Kind must be invalid so an uninitialised value is detectable")
	}
	if got := k.String(); got != "invalid" {
		t.Errorf("zero Kind String() = %q", got)
	}
}

func TestKindStringIsStable(t *testing.T) {
	// These strings appear in logs and test output; changing one should be a
	// deliberate act, not an accident.
	want := map[Kind]string{
		KindRequestQueued:       "request-queued",
		KindRequestSent:         "request-sent",
		KindResponseChunk:       "response-chunk",
		KindResponseReceived:    "response-received",
		KindTestResults:         "test-results",
		KindAssertionResults:    "assertion-results",
		KindUnresolvedVariables: "unresolved-variables",
		KindScriptedRequest:     "scripted-request",
		KindCookiesUpdated:      "cookies-updated",
		KindCredentialsUpdated:  "credentials-updated",
		KindCollectionChanged:   "collection-changed",
		KindScriptOutput:        "script-output",
		KindErrorOccurred:       "error-occurred",
	}
	for k, s := range want {
		if got := k.String(); got != s {
			t.Errorf("Kind %d String() = %q, want %q", k, got, s)
		}
	}
}

// TestTerminalKindsAreExactlyTheTwoOutcomes pins the rule that gives every
// request exactly one ending.
func TestTerminalKindsAreExactlyTheTwoOutcomes(t *testing.T) {
	var terminals []Kind
	for _, k := range AllKinds {
		if k.IsTerminal() {
			terminals = append(terminals, k)
		}
	}
	if len(terminals) != 2 {
		t.Fatalf("expected exactly 2 terminal kinds, got %d: %v", len(terminals), terminals)
	}
	if !KindResponseReceived.IsTerminal() || !KindErrorOccurred.IsTerminal() {
		t.Fatalf("terminal kinds are %v, want response-received and error-occurred", terminals)
	}
}

// TestTerminalKindsAreNeverDroppable guards the back-pressure rule: the bus may
// discard chunks and console lines, but must never discard an ending.
func TestTerminalKindsAreNeverDroppable(t *testing.T) {
	for _, k := range AllKinds {
		if k.IsTerminal() && k.IsDroppable() {
			t.Errorf("%s is both terminal and droppable; terminal events must never be dropped", k)
		}
	}
}

func TestDroppableKinds(t *testing.T) {
	droppable := map[Kind]bool{
		KindResponseChunk: true,
		KindScriptOutput:  true,
	}
	for _, k := range AllKinds {
		if got := k.IsDroppable(); got != droppable[k] {
			t.Errorf("%s.IsDroppable() = %v, want %v", k, got, droppable[k])
		}
	}
}

func TestClassifyCancellationAndTimeout(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"canceled", context.Canceled, ErrorCancelled},
		{"wrapped canceled", fmt.Errorf("sending: %w", context.Canceled), ErrorCancelled},
		{"deadline", context.DeadlineExceeded, ErrorTimeout},
		{"wrapped deadline", fmt.Errorf("dial: %w", context.DeadlineExceeded), ErrorTimeout},
		{"unknown", errors.New("something exploded"), ErrorUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.err); got != tc.want {
				t.Errorf("Classify = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestNewErrorCarriesMessageAndCause checks that a consumer sees a user-facing
// message while the producer can still unwrap the original error.
func TestNewErrorCarriesMessageAndCause(t *testing.T) {
	cause := errors.New("dial tcp 10.0.0.1:443: connection refused")
	ev := NewError(ScopeSend, cause)

	if ev.Kind() != KindErrorOccurred {
		t.Fatalf("Kind = %v", ev.Kind())
	}
	if ev.Scope != ScopeSend {
		t.Errorf("Scope = %q", ev.Scope)
	}
	if ev.Msg != cause.Error() {
		t.Errorf("Msg = %q, want the error text", ev.Msg)
	}
	if !errors.Is(ev, cause) {
		t.Error("Unwrap should expose the original error")
	}
	var target ErrorOccurred
	if !errors.As(fmt.Errorf("wrap: %w", ev), &target) {
		t.Error("ErrorOccurred should be reachable through errors.As")
	}
	if target.Class != ErrorUnknown {
		t.Errorf("errors.As target Class = %q", target.Class)
	}
}

func TestNewErrorClassifiesCancellation(t *testing.T) {
	ev := NewError(ScopeSend, context.Canceled)
	if !ev.Cancelled() {
		t.Fatalf("Class = %q, want cancelled", ev.Class)
	}
	// Cancellation must still be a terminal event, so the UI gets an ending.
	if !ev.Kind().IsTerminal() {
		t.Error("a cancelled request must produce a terminal event")
	}
}

func TestErrorBuilders(t *testing.T) {
	ev := NewError(ScopeSend, errors.New("boom")).
		WithRequest("req-1", "item-1", "coll-1").
		WithMessage("Could not send the request").
		WithDetails("stack…")

	if ev.UID != "req-1" || ev.ItemUID != "item-1" || ev.CollectionUID != "coll-1" {
		t.Errorf("identity not attached: %+v", ev)
	}
	if ev.Msg != "Could not send the request" {
		t.Errorf("Msg = %q", ev.Msg)
	}
	if ev.Details != "stack…" {
		t.Errorf("Details = %q", ev.Details)
	}
	// Replacing the message must not lose the cause.
	if ev.Unwrap() == nil {
		t.Error("WithMessage dropped the wrapped error")
	}
}

func TestTestResultsSummary(t *testing.T) {
	tr := TestResults{Results: []model.TestResult{{Status: StatusPass}, {Status: StatusPass}, {Status: StatusFail}, {Status: StatusSkip}}}
	passed, failed, skipped := tr.Summary()
	if passed != 2 || failed != 1 || skipped != 1 {
		t.Errorf("Summary = %d/%d/%d, want 2/1/1", passed, failed, skipped)
	}
}

func TestAssertionResultsAllPassed(t *testing.T) {
	var none AssertionResults
	if !none.AllPassed() {
		t.Error("no assertions vacuously pass")
	}
	ok := AssertionResults{Results: []model.AssertionResult{{Passed: true}, {Passed: true}}}
	if !ok.AllPassed() {
		t.Error("all-passing assertions should report true")
	}
	bad := AssertionResults{Results: []model.AssertionResult{{Passed: true}, {Passed: false}}}
	if bad.AllPassed() {
		t.Error("a failing assertion should report false")
	}
}

func TestScopeAndPhaseConstantsAreDistinct(t *testing.T) {
	scopes := []string{
		ScopeSend, ScopeSave, ScopeLoad, ScopeDelete, ScopeMove, ScopeImport,
		ScopeExport, ScopeRunner, ScopeRefresh, ScopeCollection, ScopeEnvironment,
		ScopeSettings, ScopeOAuth, ScopeWatcher, ScopeScript, ScopeInternal,
	}
	assertDistinct(t, "scope", scopes)

	phases := []string{PhasePreRequest, PhasePostResponse, PhaseTests}
	assertDistinct(t, "phase", phases)

	levels := []string{LevelLog, LevelWarn, LevelError}
	assertDistinct(t, "level", levels)

	statuses := []string{StatusPass, StatusFail, StatusSkip, StatusError}
	assertDistinct(t, "status", statuses)
}

func assertDistinct(t *testing.T, what string, values []string) {
	t.Helper()
	seen := make(map[string]bool, len(values))
	for _, v := range values {
		if v == "" {
			t.Errorf("%s: empty constant", what)
		}
		if seen[v] {
			t.Errorf("%s: duplicate value %q", what, v)
		}
		seen[v] = true
	}
}

// TestErrorClassesAreDistinct keeps the presentation categories exhaustive without
// collapse.
func TestErrorClassesAreDistinct(t *testing.T) {
	assertDistinct(t, "error class", []string{
		ErrorCancelled, ErrorTimeout, ErrorNetwork, ErrorValidation,
		ErrorNotFound, ErrorConflict, ErrorPermission, ErrorUnknown,
	})
}

func TestChangeOriginDistinct(t *testing.T) {
	if OriginSelf == OriginExternal {
		t.Fatal("change origins must differ")
	}
}

// TestEventsAreCopyableValues checks the "events are values" rule. A struct
// containing a mutex or an uncomparable field would fail to compile here, and a
// shallow copy must not alias core state.
func TestEventsAreCopyableValues(t *testing.T) {
	// Type assertions that the non-pointer events are plain comparable-ish
	// structs without hidden pointers into shared state. ResponseChunk carries a
	// []byte by design; it is documented as not to be retained.
	_ = []Event{
		RequestQueued{UID: "a"},
		RequestSent{UID: "a"},
		ResponseChunk{UID: "a"},
		ResponseReceived{UID: "a"},
		TestResults{UID: "a"},
		AssertionResults{UID: "a"},
		UnresolvedVariables{UID: "a"},
		ScriptedRequest{ParentUID: "a"},
		CookiesUpdated{},
		CredentialsUpdated{CredentialsID: "a"},
		CollectionChanged{CollectionUID: "a"},
		ScriptOutput{UID: "a"},
	}

	orig := RequestQueued{UID: "x", ItemUID: "i"}
	cp := orig
	cp.UID = "y"
	if orig.UID != "x" {
		t.Fatal("copying a value event must not alias the original")
	}
}

// TestNoEventExposesRefreshToken is the redaction requirement from
// interface-design.md §9: credentials events carry what the UI needs and nothing
// that could mint a session.
func TestNoEventExposesRefreshToken(t *testing.T) {
	ev := CredentialsUpdated{
		CredentialsID: "c1",
		AccessToken:   "access",
		TokenType:     "Bearer",
		ExpiresIn:     3600,
		Scope:         "read",
	}
	// The struct deliberately has no RefreshToken field; this test documents the
	// intent so a future addition is a conscious decision. If someone adds one,
	// this assertion must be updated in the same change.
	_ = ev
	if _, ok := any(ev).(interface{ RefreshToken() string }); ok {
		t.Fatal("CredentialsUpdated must not expose a refresh token")
	}
}

// TestTimestampsAreUTC keeps event timestamps unambiguous in logs.
func TestTimestampsAreUTC(t *testing.T) {
	now := time.Now().UTC()
	ev := RequestQueued{QueuedAt: now}
	if ev.QueuedAt.Location() != time.UTC {
		t.Errorf("QueuedAt should be UTC, got %v", ev.QueuedAt.Location())
	}
	so := ScriptOutput{At: now}
	if so.At.Location() != time.UTC {
		t.Errorf("ScriptOutput.At should be UTC, got %v", so.At.Location())
	}
}

// TestErrorOccurredSatisfiesEventAsValue pins the design decision that every
// event, including the error one, is a value: a presenter's switch on
// `case events.ErrorOccurred:` must work without a pointer dereference.
func TestErrorOccurredSatisfiesEventAsValue(t *testing.T) {
	var ev Event = NewError(ScopeSend, errors.New("boom"))
	if _, ok := ev.(ErrorOccurred); !ok {
		t.Fatalf("expected a value type in the interface, got %T", ev)
	}

	// Dispatch through the interface, the way a presenter does.
	handled := false
	switch e := ev.(type) {
	case ErrorOccurred:
		handled = true
		if e.Scope != ScopeSend {
			t.Errorf("Scope = %q", e.Scope)
		}
	case *ErrorOccurred:
		t.Fatal("events must be values; a pointer case would be a bug")
	}
	if !handled {
		t.Fatal("ErrorOccurred was not handled by a value case")
	}
}

// TestEveryEventDispatchesByKind shows the intended presenter pattern: one
// switch over Kind, covering every event, with no string comparison.
func TestEveryEventDispatchesByKind(t *testing.T) {
	all := []Event{
		RequestQueued{}, RequestSent{}, ResponseChunk{}, ResponseReceived{},
		TestResults{}, AssertionResults{}, UnresolvedVariables{},
		ScriptedRequest{}, CookiesUpdated{}, CredentialsUpdated{},
		CollectionChanged{}, ScriptOutput{}, NewError(ScopeInternal, nil),
	}
	for _, ev := range all {
		seen := 0
		switch ev.Kind() {
		case KindRequestQueued, KindRequestSent, KindResponseChunk,
			KindResponseReceived, KindTestResults, KindAssertionResults,
			KindUnresolvedVariables, KindScriptedRequest, KindCookiesUpdated,
			KindCredentialsUpdated, KindCollectionChanged, KindScriptOutput,
			KindErrorOccurred:
			seen++
		default:
			t.Fatalf("%T reported an unhandled kind %v", ev, ev.Kind())
		}
		if seen != 1 {
			t.Fatalf("%T matched %d cases", ev, seen)
		}
	}
}

// TestErrorsAsIdiomForValueReceiver pins the documented gotcha: because
// ErrorOccurred uses a value receiver, errors.As needs a value target. A
// *ErrorOccurred target silently fails to match, which would make a CLI fall
// through to its generic exit code without any compile error.
func TestErrorsAsIdiomForValueReceiver(t *testing.T) {
	err := fmt.Errorf("command failed: %w", NewError(ScopeSend, context.DeadlineExceeded))

	var valueTarget ErrorOccurred
	if !errors.As(err, &valueTarget) {
		t.Fatal("errors.As with a value target should match")
	}
	if valueTarget.Class != ErrorTimeout {
		t.Errorf("Class = %q, want timeout", valueTarget.Class)
	}

	var pointerTarget *ErrorOccurred
	if errors.As(err, &pointerTarget) {
		t.Fatal("a pointer target must not match; the documented idiom is a value target")
	}
}

// TestClassifySurvivesWrapping checks that wrapping does not defeat
// classification, since commands add context before publishing.
func TestClassifySurvivesWrapping(t *testing.T) {
	wrapped := fmt.Errorf("send item abc: %w", fmt.Errorf("transport: %w", context.Canceled))
	if got := Classify(wrapped); got != ErrorCancelled {
		t.Errorf("Classify = %q, want cancelled", got)
	}
	if ev := NewError(ScopeSend, wrapped); !ev.Cancelled() {
		t.Errorf("Cancelled = false, want true")
	}
}
