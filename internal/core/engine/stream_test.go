package engine

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// recorder collects events for assertions.
type recorder struct {
	mu     sync.Mutex
	events []Event
}

func (r *recorder) HandleEvent(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recorder) all() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Event(nil), r.events...)
}

func (r *recorder) ofKind(kind Kind) []Event {
	var out []Event
	for _, e := range r.all() {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

func (r *recorder) messages() []Message {
	var out []Message
	for _, e := range r.ofKind(KindMessage) {
		out = append(out, e.Message)
	}
	return out
}

// parse feeds a whole document and closes.
func parse(t *testing.T, doc string) (*recorder, *Parser) {
	t.Helper()
	rec := &recorder{}
	p := NewParser(rec, 0)
	if err := p.Write([]byte(doc)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return rec, p
}

// TestParseSimpleMessages covers the basic field set.
func TestParseSimpleMessages(t *testing.T) {
	rec, _ := parse(t, "data: hello\n\ndata: world\n\n")

	msgs := rec.messages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %+v", msgs)
	}
	if msgs[0].Data != "hello" || msgs[1].Data != "world" {
		t.Errorf("data = %q, %q", msgs[0].Data, msgs[1].Data)
	}
	// An event with no `event:` field is a "message", per the spec.
	if msgs[0].Event != "message" {
		t.Errorf("event type = %q", msgs[0].Event)
	}
	// Sequence numbers start at 1 and increase.
	if msgs[0].Seq != 1 || msgs[1].Seq != 2 {
		t.Errorf("seq = %d, %d", msgs[0].Seq, msgs[1].Seq)
	}
}

// TestParseMultiLineData joins data lines with newlines, as the spec requires.
func TestParseMultiLineData(t *testing.T) {
	rec, _ := parse(t, "data: line one\ndata: line two\ndata: line three\n\n")

	msgs := rec.messages()
	if len(msgs) != 1 {
		t.Fatalf("messages = %d", len(msgs))
	}
	// The spec removes the final newline, so three lines join with two newlines.
	if want := "line one\nline two\nline three"; msgs[0].Data != want {
		t.Errorf("data = %q, want %q", msgs[0].Data, want)
	}
}

// TestParseEventTypeAndID covers the other fields.
func TestParseEventTypeAndID(t *testing.T) {
	rec, _ := parse(t, "event: update\nid: 42\ndata: x\n\n")

	msgs := rec.messages()
	if len(msgs) != 1 {
		t.Fatalf("messages = %d", len(msgs))
	}
	if msgs[0].Event != "update" {
		t.Errorf("event = %q", msgs[0].Event)
	}
	// The id persists across messages, because it is the last id seen.
	if msgs[0].ID != "42" {
		t.Errorf("id = %q", msgs[0].ID)
	}
}

// TestParseIDPersistsAcrossMessages is the reconnection contract.
func TestParseIDPersistsAcrossMessages(t *testing.T) {
	rec, _ := parse(t, "id: 1\ndata: a\n\ndata: b\n\n")

	msgs := rec.messages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %d", len(msgs))
	}
	if msgs[1].ID != "1" {
		t.Errorf("second message id = %q, want the last id seen", msgs[1].ID)
	}
}

// TestParseRetry covers the reconnection delay.
func TestParseRetry(t *testing.T) {
	rec, _ := parse(t, "retry: 3000\ndata: x\n\n")

	retries := rec.ofKind(KindRetry)
	if len(retries) != 1 {
		t.Fatalf("retry events = %d", len(retries))
	}

	msgs := rec.messages()
	if len(msgs) != 1 || msgs[0].Retry != 3*time.Second {
		t.Errorf("retry = %v", msgs[0].Retry)
	}

	// A non-numeric retry is ignored rather than treated as zero.
	rec2, _ := parse(t, "retry: soon\ndata: x\n\n")
	if got := rec2.messages()[0].Retry; got != 0 {
		t.Errorf("invalid retry produced %v", got)
	}
}

// TestParseComments covers keep-alive comments.
func TestParseComments(t *testing.T) {
	rec, _ := parse(t, ": keep-alive\n\ndata: real\n\n")

	comments := rec.ofKind(KindComment)
	if len(comments) != 1 || comments[0].Text != " keep-alive" {
		t.Fatalf("comments = %+v", comments)
	}

	// A comment alone must not produce a message: the data buffer is empty.
	msgs := rec.messages()
	if len(msgs) != 1 {
		t.Fatalf("messages = %+v", msgs)
	}
	if msgs[0].Data != "real" {
		t.Errorf("data = %q", msgs[0].Data)
	}
}

// TestParseLineEndings covers all three forms plus a split CRLF.
func TestParseLineEndings(t *testing.T) {
	cases := map[string]string{
		"LF":    "data: a\n\ndata: b\n\n",
		"CRLF":  "data: a\r\n\r\ndata: b\r\n\r\n",
		"CR":    "data: a\r\rdata: b\r\r",
		"mixed": "data: a\r\n\n",
	}
	for name, doc := range cases {
		rec, _ := parse(t, doc)
		msgs := rec.messages()
		if len(msgs) == 0 {
			t.Errorf("%s: no messages parsed", name)
			continue
		}
		if name != "mixed" && len(msgs) != 2 {
			t.Errorf("%s: messages = %d, want 2", name, len(msgs))
		}
	}
}

// TestParseCrLfSplitAcrossFragments is why the CR state exists: a fragment can
// end in CR while the LF arrives in the next one.
func TestParseCrLfSplitAcrossFragments(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)

	for _, fragment := range []string{"data: a\r", "\n\r", "\ndata: b\n\n"} {
		if err := p.Write([]byte(fragment)); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	msgs := rec.messages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %+v", msgs)
	}
	if msgs[0].Data != "a" || msgs[1].Data != "b" {
		t.Errorf("data = %q, %q", msgs[0].Data, msgs[1].Data)
	}
}

// TestParseByteAtATime is the strongest fragmentation case: every fragment is one
// byte, so no line is ever complete within a write.
func TestParseByteAtATime(t *testing.T) {
	doc := "event: tick\nid: 7\ndata: one\ndata: two\n\n"
	rec := &recorder{}
	p := NewParser(rec, 0)

	for i := 0; i < len(doc); i++ {
		if err := p.Write([]byte{doc[i]}); err != nil {
			t.Fatalf("write at %d: %v", i, err)
		}
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	msgs := rec.messages()
	if len(msgs) != 1 {
		t.Fatalf("messages = %+v", msgs)
	}
	if msgs[0].Data != "one\ntwo" || msgs[0].Event != "tick" || msgs[0].ID != "7" {
		t.Errorf("message = %+v", msgs[0])
	}
}

// TestParseLeadingBOM covers a provider that prefixes a BOM.
func TestParseLeadingBOM(t *testing.T) {
	rec, _ := parse(t, "\uFEFFdata: x\n\n")

	msgs := rec.messages()
	if len(msgs) != 1 || msgs[0].Data != "x" {
		t.Fatalf("messages = %+v", msgs)
	}
}

// TestParseNoSpaceAfterColon covers `data:x`, which is legal.
func TestParseNoSpaceAfterColon(t *testing.T) {
	rec, _ := parse(t, "data:x\n\n")
	if got := rec.messages()[0].Data; got != "x" {
		t.Errorf("data = %q", got)
	}
}

// TestParseFieldWithoutColon covers `data` alone, which means an empty value.
func TestParseFieldWithoutColon(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)
	// `data` with no colon is an empty data line, then a blank line dispatches it.
	if err := p.Write([]byte("data\n\n")); err != nil {
		t.Fatal(err)
	}
	_ = p.Close()

	msgs := rec.messages()
	if len(msgs) != 1 {
		t.Fatalf("messages = %+v", msgs)
	}
	// An empty data line still counts as data present, so a message is dispatched
	// with empty content rather than none at all.
	if msgs[0].Data != "" {
		t.Errorf("data = %q, want empty", msgs[0].Data)
	}
}

// TestParseUnknownFieldIgnored covers forward compatibility.
func TestParseUnknownFieldIgnored(t *testing.T) {
	rec, _ := parse(t, "future: value\ndata: x\n\n")
	msgs := rec.messages()
	if len(msgs) != 1 || msgs[0].Data != "x" {
		t.Errorf("messages = %+v", msgs)
	}
}

// TestParseUnterminatedFinalEvent covers Close dispatching a pending event, which
// the spec requires for a stream that ends without a blank line.
func TestParseUnterminatedFinalEvent(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)
	if err := p.Write([]byte("data: last")); err != nil {
		t.Fatal(err)
	}
	// No blank line: the event is still pending.
	if len(rec.messages()) != 0 {
		t.Fatal("a message was dispatched before the stream ended")
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}

	msgs := rec.messages()
	if len(msgs) != 1 || msgs[0].Data != "last" {
		t.Fatalf("messages = %+v", msgs)
	}
}

// TestParseNulIDIgnored covers the spec's rule for an id containing NUL.
func TestParseNulIDIgnored(t *testing.T) {
	rec, _ := parse(t, "id: a\x00b\ndata: x\n\n")
	if got := rec.messages()[0].ID; got != "" {
		t.Errorf("id = %q, want it ignored", got)
	}
}

// TestParseLineLimit covers the malformed-stream bound.
func TestParseLineLimit(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)
	p.MaxLineBytes = 16

	// No newline at all, so the line buffers forever without the bound.
	err := p.Write([]byte(strings.Repeat("x", 64)))
	if err == nil {
		t.Fatal("expected a line-limit error")
	}

	errorsSeen := rec.ofKind(KindError)
	if len(errorsSeen) != 1 || !strings.Contains(errorsSeen[0].Text, "line exceeds") {
		t.Errorf("error events = %+v", errorsSeen)
	}

	// Further writes are ignored rather than emitting more errors.
	if err := p.Write([]byte("more")); err != nil {
		t.Errorf("write after failure = %v", err)
	}
}

// TestParseMessageLimit covers the accumulated-data bound.
func TestParseMessageLimit(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)
	p.MaxMessageBytes = 32

	var doc strings.Builder
	for i := 0; i < 20; i++ {
		doc.WriteString("data: " + strings.Repeat("y", 10) + "\n")
	}
	_ = p.Write([]byte(doc.String()))

	errorsSeen := rec.ofKind(KindError)
	if len(errorsSeen) != 1 || !strings.Contains(errorsSeen[0].Text, "message exceeds") {
		t.Errorf("error events = %+v", errorsSeen)
	}
}

// TestOpenAndDoneBracketTheStream covers the lifecycle markers.
func TestOpenAndDoneBracketTheStream(t *testing.T) {
	rec, _ := parse(t, "data: x\n\n")

	events := rec.all()
	if len(events) < 3 {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Kind != KindOpen {
		t.Errorf("first event = %s", events[0].Kind)
	}
	if events[len(events)-1].Kind != KindDone {
		t.Errorf("last event = %s", events[len(events)-1].Kind)
	}
}

// TestProgressEventsAccumulate covers byte accounting.
func TestProgressEventsAccumulate(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 100)

	_ = p.Write([]byte("data: a\n\n"))
	_ = p.Write([]byte("data: b\n\n"))
	_ = p.Close()

	bytesEvents := rec.ofKind(KindBytes)
	if len(bytesEvents) != 2 {
		t.Fatalf("byte events = %d", len(bytesEvents))
	}
	if bytesEvents[0].Total != 9 || bytesEvents[1].Total != 18 {
		t.Errorf("totals = %d, %d", bytesEvents[0].Total, bytesEvents[1].Total)
	}
	// The expected length is carried on every progress event so the UI can show a
	// fraction without tracking it separately.
	if bytesEvents[0].Expected != 100 {
		t.Errorf("expected = %d", bytesEvents[0].Expected)
	}

	done := rec.ofKind(KindDone)
	if len(done) != 1 || done[0].Total != 18 {
		t.Errorf("done = %+v", done)
	}
}

// TestFragmentSizeMatchesInputNotLines is the "does not buffer the whole body"
// property: an event is dispatched as soon as its terminator arrives, without
// waiting for the stream to end.
func TestFragmentSizeMatchesInputNotLines(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)

	// First event, then a second that never terminates.
	_ = p.Write([]byte("data: first\n\n"))
	if got := len(rec.messages()); got != 1 {
		t.Fatalf("after the first event, messages = %d", got)
	}

	_ = p.Write([]byte("data: second\n"))
	// Still only the first: the second is unterminated.
	if got := len(rec.messages()); got != 1 {
		t.Errorf("an unterminated event was dispatched early")
	}

	_ = p.Close()
	if got := len(rec.messages()); got != 2 {
		t.Errorf("messages after close = %d, want 2", got)
	}
}

// cancelOnRead cancels the context from inside Read, so the cancel lands between
// the source producing a fragment and Pump deciding what to do with it. It is the
// narrow window B-018 was about, made deterministic.
type cancelOnRead struct {
	mu        sync.Mutex
	fragments []string
	index     int
	cancel    context.CancelFunc
}

func (r *cancelOnRead) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.index >= len(r.fragments) {
		return 0, io.EOF
	}
	n := copy(p, r.fragments[r.index])
	r.index++
	r.cancel()
	return n, nil
}

// chunkReader yields pre-set fragments, optionally blocking on a channel.
type chunkReader struct {
	mu        sync.Mutex
	fragments []string
	index     int
	// block, when non-nil, is waited on before each read, letting a test cancel
	// between fragments.
	block chan struct{}
	// err is returned once the fragments run out.
	err error
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.block != nil {
		<-r.block
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.index >= len(r.fragments) {
		if r.err != nil {
			return 0, r.err
		}
		return 0, io.EOF
	}
	n := copy(p, r.fragments[r.index])
	r.index++
	return n, nil
}

// TestPumpDeliversEventsIncrementally covers the streaming path end to end.
func TestPumpDeliversEventsIncrementally(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)

	src := &chunkReader{fragments: []string{
		"event: start\ndata: {}\n\n",
		": keep-alive\n\n",
		"data: partial",
		" rest\n\n",
	}}

	if err := Pump(context.Background(), p, src); err != nil {
		t.Fatalf("pump: %v", err)
	}

	msgs := rec.messages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %+v", msgs)
	}
	if msgs[0].Event != "start" {
		t.Errorf("first event = %q", msgs[0].Event)
	}
	// The fourth fragment continues the third, which is what makes fragmentation
	// invisible to the consumer.
	if msgs[1].Data != "partial rest" {
		t.Errorf("second data = %q", msgs[1].Data)
	}
	if len(rec.ofKind(KindComment)) != 1 {
		t.Errorf("comment events = %d", len(rec.ofKind(KindComment)))
	}
}

// TestPumpCancelsMidStream is the acceptance criterion for cancellation.
func TestPumpCancelsMidStream(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)

	ctx, cancel := context.WithCancel(context.Background())
	block := make(chan struct{})

	src := &chunkReader{
		fragments: []string{"data: a\n\n", "data: b\n\n", "data: c\n\n"},
		block:     block,
	}

	done := make(chan error, 1)
	go func() { done <- Pump(ctx, p, src) }()

	// Let two fragments through, then cancel.
	block <- struct{}{}
	block <- struct{}{}
	cancel()
	close(block)

	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("pump err = %v, want context.Canceled", err)
	}

	// The events already parsed are delivered; the rest are not.
	if got := len(rec.messages()); got >= 3 {
		t.Errorf("messages = %d, the whole stream was consumed despite cancellation", got)
	}
}

// TestPumpDropsFragmentProducedAfterCancel pins the boundary that B-018 exposed:
// a source that does not observe the context can unblock after the cancel and
// hand back one more fragment. That fragment must not reach the parser.
func TestPumpDropsFragmentProducedAfterCancel(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)

	ctx, cancel := context.WithCancel(context.Background())
	src := &chunkReader{fragments: []string{"data: late\n\n"}}
	cancel()

	if err := Pump(ctx, p, src); !errors.Is(err, context.Canceled) {
		t.Fatalf("pump err = %v, want context.Canceled", err)
	}
	if got := rec.messages(); len(got) != 0 {
		t.Errorf("messages = %+v, want none: the fragment arrived after cancel", got)
	}
	if src.index != 0 {
		t.Errorf("source read %d fragments, want 0: nothing should be read after cancel", src.index)
	}
}

// TestPumpReportsCancelNotEOFWhenReadEndsAfterCancel pins that cancellation wins
// over the terminal error of a body that finished while the context was going
// down, so callers do not mistake a stop for a complete stream.
func TestPumpReportsCancelNotEOFWhenReadEndsAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	src := &cancelOnRead{
		fragments: []string{"data: a\n\n", "data: b\n\n"},
		cancel:    cancel,
	}

	err := Pump(ctx, NewParser(&recorder{}, 0), src)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pump err = %v, want context.Canceled rather than EOF", err)
	}
}

// TestPumpPropagatesReadError covers a transport failure during the body.
func TestPumpPropagatesReadError(t *testing.T) {
	rec := &recorder{}
	p := NewParser(rec, 0)

	src := &chunkReader{fragments: nil, err: errors.New("connection reset")}
	if err := Pump(context.Background(), p, src); err == nil {
		t.Fatal("expected the read error")
	}
}

// TestPumpAlreadyCancelled does nothing when the context is already done.
func TestPumpAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	src := &chunkReader{fragments: []string{"data: a\n\n"}}
	if err := Pump(ctx, NewParser(nil, 0), src); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

// TestProgressSnapshot covers the fraction and rate arithmetic.
func TestProgressSnapshot(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start

	p := NewProgress(200)
	p.SetClock(func() time.Time { return now })
	// Reset the start so elapsed is measured from the pinned clock.
	p.started = start

	now = start.Add(2 * time.Second)
	if got := p.Add(100); got != 100 {
		t.Errorf("total = %d", got)
	}

	snap := p.Snapshot()
	if snap.Received != 100 || snap.Expected != 200 {
		t.Errorf("snapshot = %+v", snap)
	}
	if snap.Fraction != 0.5 {
		t.Errorf("fraction = %v", snap.Fraction)
	}
	if snap.Elapsed != 2*time.Second {
		t.Errorf("elapsed = %v", snap.Elapsed)
	}
	if snap.BytesPerSecond != 50 {
		t.Errorf("rate = %v", snap.BytesPerSecond)
	}
	if snap.Done {
		t.Error("snapshot reported done before Finish")
	}

	final := p.Finish()
	if !final.Done {
		t.Error("Finish did not mark done")
	}
}

// TestProgressWithoutContentLength reports an unknown fraction as -1 rather than
// inventing a denominator.
func TestProgressWithoutContentLength(t *testing.T) {
	p := NewProgress(0)
	p.Add(50)

	snap := p.Snapshot()
	if snap.Expected != 0 {
		t.Errorf("expected = %d", snap.Expected)
	}
	if snap.Fraction != -1 {
		t.Errorf("fraction = %v, want -1 for unknown length", snap.Fraction)
	}
}

// TestProgressCanExceedExpected keeps a bar from claiming completion early.
func TestProgressCanExceedExpected(t *testing.T) {
	p := NewProgress(100)
	p.Add(150)

	snap := p.Snapshot()
	if snap.Fraction <= 1 {
		t.Errorf("fraction = %v; an understated Content-Length must not read as complete", snap.Fraction)
	}
}

// TestProgressIsConcurrencySafe covers the UI reading while the transport writes.
func TestProgressIsConcurrencySafe(t *testing.T) {
	p := NewProgress(1000)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			p.Add(1)
		}
	}()

	for i := 0; i < 100; i++ {
		_ = p.Snapshot()
	}
	wg.Wait()

	if got := p.Snapshot().Received; got != 100 {
		t.Errorf("received = %d", got)
	}
}
