package response

import (
	"context"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/engine"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

func TestIsEventStream(t *testing.T) {
	strp := func(s string) *string { return &s }

	p := presenters.NewResponsePresenter()
	if IsEventStream(p) {
		t.Error("expected empty presenter to not be event stream")
	}

	// Content-Type: text/event-stream
	p.HandleEvent(events.RequestQueued{UID: "r1"})
	p.HandleEvent(events.ResponseReceived{
		UID: "r1",
		Received: model.Result{
			Headers: []model.KeyValue{
				{Name: strp("Content-Type"), Value: strp("text/event-stream; charset=utf-8")},
			},
		},
	})
	if !IsEventStream(p) {
		t.Error("expected text/event-stream header to be recognized as event stream")
	}

	// Body text starting with SSE data
	p2 := presenters.NewResponsePresenter()
	p2.HandleEvent(events.RequestQueued{UID: "r2"})
	p2.HandleEvent(events.ResponseChunk{UID: "r2", Data: []byte("data: hello\n\n")})
	if !IsEventStream(p2) {
		t.Error("expected body starting with data: to be recognized as event stream")
	}
}

func TestParseSSE(t *testing.T) {
	raw := "event: update\nid: msg-1\nretry: 5000\ndata: first line\ndata: second line\n\n: comment\n\ndata: second message\n\n"
	msgs := ParseSSE(raw)

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	m1 := msgs[0]
	if m1.Event != "update" {
		t.Errorf("expected event 'update', got %q", m1.Event)
	}
	if m1.ID != "msg-1" {
		t.Errorf("expected id 'msg-1', got %q", m1.ID)
	}
	if m1.Retry != 5*time.Second {
		t.Errorf("expected retry 5s, got %v", m1.Retry)
	}
	if m1.Data != "first line\nsecond line" {
		t.Errorf("expected multiline data, got %q", m1.Data)
	}

	m2 := msgs[1]
	if m2.Event != "message" {
		t.Errorf("expected default event 'message', got %q", m2.Event)
	}
	if m2.Data != "second message" {
		t.Errorf("expected data 'second message', got %q", m2.Data)
	}
}

func TestStreamPane_Render(t *testing.T) {
	msgs := []engine.Message{
		{Seq: 1, Event: "greeting", ID: "1", Data: `{"hello": "world"}`},
		{Seq: 2, Event: "status", Retry: 2500 * time.Millisecond, Data: "active"},
	}
	snap := engine.Snapshot{
		Received:       2048,
		Expected:       4096,
		Fraction:       0.5,
		BytesPerSecond: 10240,
		Done:           false,
	}

	cancelled := false
	props := StreamPaneProps{
		Messages:    msgs,
		Progress:    snap,
		IsStreaming: true,
		OnCancel: func() {
			cancelled = true
		},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		StreamPane(c, props)
	}, 800, 600)
	tt.Frame()

	// Header elements
	if !tt.HasText("STREAMING") {
		t.Error("expected STREAMING tag")
	}
	if !tt.HasText("2 events") {
		t.Error("expected '2 events'")
	}
	if !tt.HasText("2.0 KB") {
		t.Error("expected '2.0 KB' received bytes")
	}
	if !tt.HasText("10.0 KB/s") {
		t.Error("expected transfer rate '10.0 KB/s'")
	}

	// Message cards
	if !tt.HasText("#1") || !tt.HasText("greeting") || !tt.HasText(`{"hello": "world"}`) {
		t.Error("expected message 1 content")
	}
	if !tt.HasText("#2") || !tt.HasText("status") || !tt.HasText("active") {
		t.Error("expected message 2 content")
	}

	// Click Cancel Stream
	if err := tt.Click("Cancel Stream"); err != nil {
		t.Fatalf("click Cancel Stream: %v", err)
	}
	if !cancelled {
		t.Error("expected OnCancel callback to be called")
	}
}

func TestStreamPane_EmptyState(t *testing.T) {
	props := StreamPaneProps{
		Messages:    nil,
		Progress:    engine.Snapshot{},
		IsStreaming: true,
	}

	tt := ui.NewTester(func(c *ui.Context) {
		StreamPane(c, props)
	}, 800, 600)
	tt.Frame()

	if !tt.HasText("Waiting for stream events…") {
		t.Error("expected waiting message for empty stream")
	}
}

func TestSSEStreamPane_Integration(t *testing.T) {
	strp := func(s string) *string { return &s }

	p := presenters.NewResponsePresenter()
	p.HandleEvent(events.RequestQueued{UID: "r-sse"})
	p.HandleEvent(events.ResponseReceived{
		UID: "r-sse",
		Received: model.Result{
			Status:     200,
			StatusText: "OK",
			Headers: []model.KeyValue{
				{Name: strp("Content-Type"), Value: strp("text/event-stream")},
			},
			Body: model.BodyRef{
				Inline: []byte("event: heartbeat\ndata: ping\n\n"),
			},
		},
	})

	tt := ui.NewTester(func(c *ui.Context) {
		BodyPane(c, p)
	}, 800, 600)
	tt.Frame()

	if !tt.HasText("STREAM ENDED") {
		t.Error("expected completed stream status")
	}
	if !tt.HasText("1 events") {
		t.Error("expected 1 events count")
	}
	if !tt.HasText("heartbeat") || !tt.HasText("ping") {
		t.Error("expected SSE event and data rendered in BodyPane")
	}
}

func TestStreamCancellation_MidStream(t *testing.T) {
	// Acceptance criterion: stream renders without buffering the whole body; cancellation mid-stream tested.
	var received []engine.Message
	h := engine.HandlerFunc(func(e engine.Event) {
		if e.Kind == engine.KindMessage {
			received = append(received, e.Message)
		}
	})
	parser := engine.NewParser(h, 0)

	ctx, cancel := context.WithCancel(context.Background())
	src := &chunkStreamReader{
		chunks: [][]byte{
			[]byte("data: chunk-1\n\n"),
			[]byte("data: chunk-2\n\n"),
			[]byte("data: chunk-3\n\n"),
		},
		onRead: func(readIndex int) {
			if readIndex == 2 {
				cancel()
			}
		},
	}

	err := engine.Pump(ctx, parser, src)
	if err == nil {
		t.Fatal("expected Pump to return cancellation error")
	}
	if err != context.Canceled {
		t.Errorf("got %v, want context.Canceled", err)
	}

	// Third chunk should not be parsed because stream was cancelled mid-stream
	if len(received) > 2 {
		t.Errorf("expected at most 2 messages before cancel, got %d", len(received))
	}
}

type chunkStreamReader struct {
	chunks [][]byte
	idx    int
	onRead func(int)
}

func (r *chunkStreamReader) Read(p []byte) (int, error) {
	if r.idx >= len(r.chunks) {
		return 0, nil
	}
	if r.onRead != nil {
		r.onRead(r.idx + 1)
	}
	data := r.chunks[r.idx]
	r.idx++
	n := copy(p, data)
	return n, nil
}
