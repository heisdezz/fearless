package presenters

import (
	"errors"
	"testing"
	"time"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

func TestResponsePresenter_Defaults(t *testing.T) {
	p := NewResponsePresenter()

	if p.State() != ResponseStateIdle {
		t.Errorf("expected Idle state, got %v", p.State())
	}
	if p.ActiveSubTab() != RespSubTabBody {
		t.Errorf("expected RespSubTabBody, got %v", p.ActiveSubTab())
	}
	if p.Result() != nil {
		t.Errorf("expected nil result")
	}
	if p.ErrorMessage() != "" {
		t.Errorf("expected empty error message")
	}
	if p.FormattedStatus() != "" {
		t.Errorf("expected empty formatted status")
	}
	if p.StatusCodeCategory() != "" {
		t.Errorf("expected empty status code category")
	}
	if p.FormattedDuration() != "" {
		t.Errorf("expected empty formatted duration")
	}
	if p.FormattedSize() != "" {
		t.Errorf("expected empty formatted size")
	}
	if p.BodyText() != "" {
		t.Errorf("expected empty body text")
	}
}

func TestResponsePresenter_SubTabAndChanged(t *testing.T) {
	p := NewResponsePresenter()
	changed := 0
	p.OnChanged(func() {
		changed++
	})

	for _, tab := range AllResponseSubTabs {
		p.SelectSubTab(tab)
		if p.ActiveSubTab() != tab {
			t.Errorf("expected subtab %v, got %v", tab, p.ActiveSubTab())
		}
	}
	if changed != len(AllResponseSubTabs) {
		t.Errorf("expected %d change notifications, got %d", len(AllResponseSubTabs), changed)
	}
}

func TestResponsePresenter_StatusCodeCategoryAndStatus(t *testing.T) {
	p := NewResponsePresenter()

	tests := []struct {
		code         int
		text         string
		expectedCat  string
		expectedStat string
	}{
		{200, "OK", "success", "200 OK"},
		{201, "", "success", "201"},
		{301, "Moved Permanently", "redirect", "301 Moved Permanently"},
		{404, "Not Found", "client_error", "404 Not Found"},
		{500, "Internal Server Error", "server_error", "500 Internal Server Error"},
		{101, "Switching Protocols", "", "101 Switching Protocols"},
	}

	for _, tc := range tests {
		p.Reset()
		p.HandleEvent(events.RequestQueued{UID: "req-1"})
		p.HandleEvent(events.ResponseReceived{
			UID: "req-1",
			Received: model.Result{
				Status:     tc.code,
				StatusText: tc.text,
			},
		})

		if cat := p.StatusCodeCategory(); cat != tc.expectedCat {
			t.Errorf("code %d: expected category %s, got %s", tc.code, tc.expectedCat, cat)
		}
		if stat := p.FormattedStatus(); stat != tc.expectedStat {
			t.Errorf("code %d: expected status %s, got %s", tc.code, tc.expectedStat, stat)
		}
	}
}

func TestResponsePresenter_Formatting(t *testing.T) {
	p := NewResponsePresenter()

	// FormattedDuration
	p.HandleEvent(events.RequestQueued{UID: "req-1"})
	p.HandleEvent(events.ResponseReceived{
		UID: "req-1",
		Received: model.Result{
			Status:   200,
			Duration: 500 * time.Microsecond,
			Body: model.BodyRef{
				Size:   5,
				Inline: []byte("Hello"),
			},
		},
	})
	if dur := p.FormattedDuration(); dur != "< 1 ms" {
		t.Errorf("expected '< 1 ms', got %q", dur)
	}

	p.Result().Duration = 45 * time.Millisecond
	if dur := p.FormattedDuration(); dur != "45 ms" {
		t.Errorf("expected '45 ms', got %q", dur)
	}

	p.Result().Duration = 1500 * time.Millisecond
	if dur := p.FormattedDuration(); dur != "1.50 s" {
		t.Errorf("expected '1.50 s', got %q", dur)
	}

	// FormattedSize and BodyText
	if sz := p.FormattedSize(); sz != "5 B" {
		t.Errorf("expected '5 B', got %q", sz)
	}
	if b := p.BodyText(); b != "Hello" {
		t.Errorf("expected 'Hello', got %q", b)
	}

	// Larger size
	p.Result().Body = model.BodyRef{
		Size:   2048,
		Inline: make([]byte, 2048),
	}
	if sz := p.FormattedSize(); sz != "2.0 KB" {
		t.Errorf("expected '2.0 KB', got %q", sz)
	}

	// Spilled body
	p.Result().Body = model.BodyRef{
		Size: 5000000,
		Path: "/tmp/spooled-response.bin",
	}
	if b := p.BodyText(); b != "[Body spilled to disk: /tmp/spooled-response.bin]" {
		t.Errorf("expected spilled text, got %q", b)
	}
}

func TestResponsePresenter_StreamingAndError(t *testing.T) {
	p := NewResponsePresenter()

	p.HandleEvent(events.RequestQueued{UID: "req-stream"})
	if p.State() != ResponseStateLoading {
		t.Errorf("expected loading state, got %v", p.State())
	}

	// Chunk for wrong request UID should be ignored
	p.HandleEvent(events.ResponseChunk{UID: "wrong-req", Data: []byte("ignore")})
	if p.BodyText() != "" {
		t.Errorf("expected empty body text, got %q", p.BodyText())
	}

	// Valid chunk
	p.HandleEvent(events.ResponseChunk{UID: "req-stream", Data: []byte("chunk1 ")})
	p.HandleEvent(events.ResponseChunk{UID: "req-stream", Data: []byte("chunk2")})
	if p.BodyText() != "chunk1 chunk2" {
		t.Errorf("expected 'chunk1 chunk2', got %q", p.BodyText())
	}
	if p.FormattedSize() != "13 B" {
		t.Errorf("expected '13 B', got %q", p.FormattedSize())
	}

	// Error event
	p.HandleEvent(events.ErrorOccurred{UID: "req-stream", Msg: "connection failed", Err: errors.New("connection failed")})
	if p.State() != ResponseStateError {
		t.Errorf("expected error state, got %v", p.State())
	}
	if p.ErrorMessage() != "connection failed" {
		t.Errorf("expected 'connection failed', got %q", p.ErrorMessage())
	}
	if p.FormattedStatus() != "Error" {
		t.Errorf("expected 'Error' status, got %q", p.FormattedStatus())
	}

	// Reset
	p.Reset()
	if p.State() != ResponseStateIdle {
		t.Errorf("expected idle after reset, got %v", p.State())
	}
	if p.ErrorMessage() != "" {
		t.Errorf("expected cleared error message")
	}
	if p.BodyText() != "" {
		t.Errorf("expected cleared body text")
	}
}
