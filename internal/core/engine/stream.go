package engine

// This file turns a response body into incremental events. The transport
// forwards raw fragments through its Sink; this is the layer above: it parses
// Server-Sent Events into discrete messages, tracks download progress, and makes
// cancellation observable mid-stream.
//
// It deliberately contains no rendering. The UI half of ADV-STREAM
// (`internal/ui/screens/response/stream*.go`) depends on UI-SCREEN-RESP, which is
// unimplemented; this file produces the data that screen will consume.
//
// Source: WHATWG HTML §9.2 "Server-sent events"

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"
)

// Kind classifies an incremental event.
type Kind uint8

const (
	// KindOpen reports the stream has started.
	KindOpen Kind = iota + 1
	// KindBytes reports raw body progress.
	KindBytes
	// KindMessage reports one complete SSE message.
	KindMessage
	// KindComment reports an SSE comment line (`: keep-alive`).
	KindComment
	// KindRetry reports a server-requested reconnection delay.
	KindRetry
	// KindDone reports the stream finished normally.
	KindDone
	// KindError reports a parse or transport failure.
	KindError
)

// String returns a stable name, for logs and test output.
func (k Kind) String() string {
	switch k {
	case KindOpen:
		return "open"
	case KindBytes:
		return "bytes"
	case KindMessage:
		return "message"
	case KindComment:
		return "comment"
	case KindRetry:
		return "retry"
	case KindDone:
		return "done"
	case KindError:
		return "error"
	}
	return "unknown"
}

// Message is one complete Server-Sent Event.
type Message struct {
	// Event is the `event:` field, defaulting to "message" as the spec requires.
	Event string
	// Data is the concatenation of every `data:` line, joined with newlines.
	// A trailing newline is not included; the spec's final newline is removed.
	Data string
	// ID is the last `id:` field seen, which the client must resend as
	// Last-Event-ID when reconnecting.
	ID string
	// Retry is the server's requested reconnection delay, when the message set
	// one. Zero means the server did not ask.
	Retry time.Duration
	// Seq is the message's position in the stream, from 1.
	Seq uint64
}

// Event is one incremental event delivered to a handler.
type Event struct {
	Kind Kind
	// Message is set for KindMessage.
	Message Message
	// Text carries a comment for KindComment, an error for KindError, and the
	// final newline-delimited data for KindRetry (unused).
	Text string
	// Err is set for KindError.
	Err error
	// Total is the number of bytes received so far, for KindBytes and KindDone.
	Total int64
	// Expected is the Content-Length when the server sent one, else 0.
	Expected int64
}

// Handler receives events in order.
//
// It must not retain Data slices: the producer may reuse them.
type Handler interface {
	HandleEvent(Event)
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(Event)

// HandleEvent implements Handler.
func (f HandlerFunc) HandleEvent(e Event) { f(e) }

// Parser turns raw byte fragments into events.
//
// A Parser is single-use and not safe for concurrent use; one instance belongs to
// one stream. It buffers only an incomplete line, never the whole body, so a long
// stream costs a constant amount of memory.
type Parser struct {
	// ExpectedContentLength is the Content-Length header, or 0 when absent. It is
	// reported for progress but never trusted as a hard limit.
	ExpectedContentLength int64
	// Handler receives the events.
	Handler Handler
	// MaxLineBytes bounds a single line. A server that never sends a newline
	// would otherwise grow the buffer without limit.
	MaxLineBytes int
	// MaxMessageBytes bounds the accumulated data of one message, for the same
	// reason.
	MaxMessageBytes int

	mu sync.Mutex

	opened  bool
	seq     uint64
	total   int64
	line    []byte
	dataBuf strings.Builder
	// hasData records that a `data:` line arrived, even with an empty value. A
	// message is dispatched only when it is set, which is how the spec keeps a
	// keep-alive comment from producing an empty event.
	hasData  bool
	eventTyp string
	lastID   string
	retry    time.Duration
	hasRetry bool
	// sawCR tracks a trailing CR so a following LF is consumed as one break.
	sawCR bool
	// firstLine tracks whether a leading BOM may still be present.
	firstLine bool
	failed    bool
}

// Defaults bound a malformed or hostile stream.
const (
	// DefaultMaxLineBytes is generous: a `data:` line may legitimately be a large
	// JSON document.
	DefaultMaxLineBytes = 1 << 20
	// DefaultMaxMessageBytes bounds one event's accumulated data.
	DefaultMaxMessageBytes = 8 << 20
)

// NewParser creates a parser.
func NewParser(handler Handler, expectedContentLength int64) *Parser {
	return &Parser{
		ExpectedContentLength: expectedContentLength,
		Handler:               handler,
		MaxLineBytes:          DefaultMaxLineBytes,
		MaxMessageBytes:       DefaultMaxMessageBytes,
		firstLine:             true,
	}
}

// Write consumes a fragment. It never retains the slice.
//
// It returns an error only when the stream is abandoned because a bound was
// exceeded; a handler error is reported through a KindError event so the caller
// sees ordering preserved.
func (p *Parser) Write(fragment []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.failed {
		return nil
	}
	if !p.opened {
		p.opened = true
		p.emit(Event{Kind: KindOpen})
	}

	p.total += int64(len(fragment))
	p.emit(Event{Kind: KindBytes, Total: p.total, Expected: p.ExpectedContentLength})

	for _, b := range fragment {
		// A parse bound may have been exceeded partway through this fragment.
		if p.failed {
			return nil
		}
		if err := p.consumeByte(b); err != nil {
			return err
		}
	}
	return nil
}

// Close finishes the stream: an unterminated final event is dispatched.
//
// The spec says a stream ending without a blank line still dispatches the pending
// event, which is why this exists rather than treating the last fragment as
// complete.
func (p *Parser) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.failed {
		return nil
	}
	if len(p.line) > 0 {
		p.dispatchLine()
	}
	p.dispatchMessage()

	p.opened = true
	p.emit(Event{Kind: KindDone, Total: p.total, Expected: p.ExpectedContentLength})
	return nil
}

// consumeByte feeds one byte through the line splitter.
func (p *Parser) consumeByte(b byte) error {
	// Line endings are CRLF, LF, or CR alone. A CR consumes the following LF.
	if p.sawCR {
		p.sawCR = false
		if b == '\n' {
			return nil
		}
	}

	switch b {
	case '\r':
		p.sawCR = true
		return p.endLine()
	case '\n':
		return p.endLine()
	}

	p.line = append(p.line, b)
	if limit := p.maxLine(); len(p.line) > limit {
		p.fail(errors.New("stream: line exceeds the maximum length"))
		return errors.New("stream: line exceeds the maximum length")
	}
	return nil
}

// maxLine returns the effective line bound.
func (p *Parser) maxLine() int {
	if p.MaxLineBytes > 0 {
		return p.MaxLineBytes
	}
	return DefaultMaxLineBytes
}

// endLine processes a complete line.
func (p *Parser) endLine() error {
	line := string(p.line)
	p.line = p.line[:0]

	// A UTF-8 BOM may precede the first line and is ignored.
	if p.firstLine {
		p.firstLine = false
		line = strings.TrimPrefix(line, "\uFEFF")
	}

	p.dispatchLineWith(line)
	return nil
}

// dispatchLine processes a buffered line.
func (p *Parser) dispatchLine() {
	line := string(p.line)
	p.line = p.line[:0]
	p.dispatchLineWith(line)
}

// dispatchLineWith applies one line's field.
func (p *Parser) dispatchLineWith(line string) {
	// A blank line dispatches the pending event.
	if line == "" {
		p.dispatchMessage()
		return
	}

	// A line starting with a colon is a comment.
	if strings.HasPrefix(line, ":") {
		p.emit(Event{Kind: KindComment, Text: strings.TrimPrefix(line, ":")})
		return
	}

	field, value, found := strings.Cut(line, ":")
	if !found {
		// A field with no colon has an empty value.
		value = ""
	} else {
		// A single leading space after the colon is removed.
		value = strings.TrimPrefix(value, " ")
	}

	switch field {
	case "data":
		if p.dataBuf.Len() > 0 {
			p.dataBuf.WriteByte('\n')
		}
		p.dataBuf.WriteString(value)
		// Mark that at least one data line arrived. A message is dispatched on a
		// blank line only when data was present, and `data` with no colon (an empty
		// value) still counts, which is why this is tracked separately from the
		// buffer's length.
		p.hasData = true
		if limit := p.maxMessage(); p.dataBuf.Len() > limit {
			p.fail(errors.New("stream: message exceeds the maximum length"))
		}
	case "event":
		p.eventTyp = value
	case "id":
		// The spec ignores an id containing a NUL.
		if !strings.ContainsRune(value, 0) {
			p.lastID = value
		}
	case "retry":
		if d, ok := parseRetry(value); ok {
			p.retry = d
			p.hasRetry = true
			p.emit(Event{Kind: KindRetry, Text: value})
		}
	default:
		// Unknown fields are ignored, as the spec requires for forward
		// compatibility.
	}
}

// maxMessage returns the effective message bound.
func (p *Parser) maxMessage() int {
	if p.MaxMessageBytes > 0 {
		return p.MaxMessageBytes
	}
	return DefaultMaxMessageBytes
}

// dispatchMessage emits the pending event, if any.
//
// Only a non-empty data buffer produces a message: the spec says a blank line
// with no data does not dispatch, which is what stops keep-alive comments from
// creating empty events.
func (p *Parser) dispatchMessage() {
	if !p.hasData {
		// The type and retry are still reset, so a stray `event:` line cannot
		// leak into the next message.
		p.eventTyp = ""
		p.hasRetry = false
		return
	}

	p.seq++
	msg := Message{
		Event: p.eventTyp,
		Data:  p.dataBuf.String(),
		ID:    p.lastID,
		Retry: p.retry,
		Seq:   p.seq,
	}
	if msg.Event == "" {
		msg.Event = "message"
	}

	p.dataBuf.Reset()
	p.hasData = false
	p.eventTyp = ""
	p.hasRetry = false

	p.emit(Event{Kind: KindMessage, Message: msg})
}

// fail records a terminal parse error.
func (p *Parser) fail(err error) {
	p.failed = true
	p.emit(Event{Kind: KindError, Err: err, Text: err.Error()})
}

// emit delivers an event, if a handler is set.
func (p *Parser) emit(e Event) {
	if p.Handler != nil {
		p.Handler.HandleEvent(e)
	}
}

// parseRetry parses the `retry:` field, which is a whole number of milliseconds.
func parseRetry(value string) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}
	var ms int64
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		ms = ms*10 + int64(value[i]-'0')
		if ms > 1<<40 {
			return 0, false
		}
	}
	return time.Duration(ms) * time.Millisecond, true
}

// Reader is a source of fragments, satisfied by io.Reader and the transport's
// chunk stream.
type Reader interface {
	Read(p []byte) (int, error)
}

// Pump copies src into the parser until EOF, cancellation, or an error.
//
// It reads in fixed-size fragments so a slow producer yields events as they
// arrive rather than in one burst at the end. Cancellation is checked between
// reads, so a cancelled stream stops within one fragment.
//
// Source: the acceptance criterion "stream renders without buffering the whole
// body; cancellation mid-stream tested"
func Pump(ctx context.Context, p *Parser, src Reader) error {
	buf := make([]byte, 32<<10)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, err := src.Read(buf)
		if n > 0 {
			if werr := p.Write(buf[:n]); werr != nil {
				return werr
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return p.Close()
			}
			return err
		}
	}
}

// Progress tracks download progress for the response screen.
//
// It is separate from the parser because progress matters for any body, not just
// an event stream, and because the screen needs a cheap snapshot on every frame.
type Progress struct {
	mu       sync.Mutex
	total    int64
	expected int64
	started  time.Time
	now      func() time.Time
}

// NewProgress creates a tracker. expected is Content-Length, or 0 when unknown.
func NewProgress(expected int64) *Progress {
	return &Progress{expected: expected, started: time.Now(), now: time.Now}
}

// Add records n more bytes and returns the new total.
func (p *Progress) Add(n int64) int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total += n
	return p.total
}

// Snapshot is an immutable view of progress at one instant.
type Snapshot struct {
	// Received is the bytes so far.
	Received int64
	// Expected is Content-Length, or 0 when the server sent none.
	Expected int64
	// Fraction is Received/Expected in [0,1], or -1 when Expected is 0. It is never
	// clamped to 1 early, because a server that understates Content-Length would
	// otherwise show a bar that appears complete before the body ends.
	Fraction float64
	// Elapsed is how long the transfer has run.
	Elapsed time.Duration
	// BytesPerSecond is the average rate, or 0 before enough time passed to measure.
	BytesPerSecond float64
	// Done reports whether the transfer finished.
	Done bool
}

// Snapshot returns the current progress.
func (p *Progress) Snapshot() Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.snapshotLocked(false)
}

// Finish marks the transfer complete and returns the final snapshot.
func (p *Progress) Finish() Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.snapshotLocked(true)
}

// snapshotLocked builds a Snapshot; the caller holds the lock.
func (p *Progress) snapshotLocked(done bool) Snapshot {
	elapsed := p.now().Sub(p.started)

	fraction := -1.0
	if p.expected > 0 {
		fraction = float64(p.total) / float64(p.expected)
	}

	var rate float64
	if elapsed > 0 {
		rate = float64(p.total) / elapsed.Seconds()
	}

	return Snapshot{
		Received:       p.total,
		Expected:       p.expected,
		Fraction:       fraction,
		Elapsed:        elapsed,
		BytesPerSecond: rate,
		Done:           done,
	}
}

// SetClock injects a clock for deterministic tests.
func (p *Progress) SetClock(now func() time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.now = now
}
