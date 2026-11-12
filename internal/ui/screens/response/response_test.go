package response

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

// completedPresenter builds a presenter holding a finished response.
func completedPresenter(t *testing.T) *presenters.ResponsePresenter {
	t.Helper()
	p := presenters.NewResponsePresenter()
	p.HandleEvent(events.RequestQueued{UID: "req-1"})
	p.HandleEvent(events.ResponseReceived{
		UID: "req-1",
		Received: model.Result{
			Status:     200,
			StatusText: "OK",
			Duration:   45 * time.Millisecond,
			Headers: model.ResponseHeaders{
				{Name: strp("Content-Type"), Value: strp("application/json"), Enabled: true},
				{Name: strp("X-Request-Id"), Value: strp("abc"), Enabled: true},
			},
			Body: model.BodyRef{Size: 13, Inline: []byte(`{"ok": true}`)},
			Timeline: model.Timeline{
				{Phase: model.PhaseConnect, Duration: 2 * time.Millisecond},
				{Phase: model.PhaseRequestSent, Duration: 10 * time.Millisecond},
				{Phase: model.PhaseDownload, Duration: 1 * time.Millisecond},
			},
		},
	})
	return p
}

func strp(s string) *string { return &s }

// TestScreenRendersStatusRow covers the headline the user reads first.
func TestScreenRendersStatusRow(t *testing.T) {
	p := completedPresenter(t)

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	if !tt.HasText("200 OK") {
		t.Error("expected the status line to show 200 OK")
	}
	if !tt.HasText("45 ms") {
		t.Error("expected the duration")
	}
	if !tt.HasText("13 B") {
		t.Error("expected the size")
	}
}

// TestScreenRendersIdleWithoutPanicking covers the pre-request state, which the
// shell shows before anything has been sent.
func TestScreenRendersIdleWithoutPanicking(t *testing.T) {
	p := presenters.NewResponsePresenter()

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	// The tabs are still present so the pane does not appear broken.
	if !tt.HasText("BODY") || !tt.HasText("HEADERS") {
		t.Error("expected the sub-tabs to render when idle")
	}
	if !tt.HasText("Send a request to see its response.") {
		t.Error("expected an explanation for the empty pane")
	}
}

// TestScreenRendersError covers a failed exchange.
func TestScreenRendersError(t *testing.T) {
	p := presenters.NewResponsePresenter()
	p.HandleEvent(events.RequestQueued{UID: "req-1"})
	p.HandleEvent(events.ErrorOccurred{UID: "req-1", Msg: "dial tcp: connection refused"})

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	if !tt.HasText("Error") {
		t.Error("expected an Error label")
	}
	if !tt.HasText("dial tcp: connection refused") {
		t.Error("expected the failure text")
	}
}

// TestStatusSeamIsOneDataRow covers the seam's shape: the figures are one row,
// separated by "·" separator nodes, inside a 28 DIP seam rather than a boxed
// strip.
func TestStatusSeamIsOneDataRow(t *testing.T) {
	p := completedPresenter(t)

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	if !tt.HasText("·") {
		t.Errorf("expected dim separators between the figures; texts: %v", tt.Texts())
	}

	status, ok := tt.Find("200 OK")
	if !ok {
		t.Fatal("status text not rendered")
	}
	if bottom := status.Y + status.H; bottom > 28 {
		t.Errorf("status text bottom = %.0f, want it inside the 28 DIP seam", bottom)
	}
}

// TestActiveTabCarriesTheAccentUnderline covers the seam's one accent: the
// active tab is marked by an underline, not by a chip.
func TestActiveTabCarriesTheAccentUnderline(t *testing.T) {
	p := completedPresenter(t)

	var primary ui.Color
	tt := ui.NewTester(func(c *ui.Context) {
		primary = lobetheme.TokensOf(c).Primary
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	label, ok := tt.Find("BODY")
	if !ok {
		t.Fatal("Body tab not rendered")
	}

	img := tt.Image()
	bounds := img.Bounds()

	// accentUnder reports whether the accent colour appears under a label.
	accentUnder := func(label ui.Rect) bool {
		for y := int(label.Y + label.H); y < int(label.Y+label.H+16) && y < bounds.Max.Y; y++ {
			for x := int(label.X) - 8; x < int(label.X+label.W)+8 && x < bounds.Max.X; x++ {
				if x < bounds.Min.X || y < bounds.Min.Y {
					continue
				}
				r, g, b, _ := img.At(x, y).RGBA()
				if uint8(r>>8) == primary.R && uint8(g>>8) == primary.G && uint8(b>>8) == primary.B {
					return true
				}
			}
		}
		return false
	}

	if !accentUnder(label) {
		t.Error("expected the active tab to carry the accent underline")
	}

	// An inactive tab carries no accent: one underline marks the selection.
	if other, ok := tt.Find("HEADERS"); ok && accentUnder(other) {
		t.Error("expected the inactive tab to carry no accent")
	}
}

// TestNoHardCodedColors is the D-017 guard for this package: the screen draws
// with theme tokens only, so a colour can never be pinned to one screen.
func TestNoHardCodedColors(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, literal := range []string{"ui.RGB(", "ui.RGBA(", "ui.Hex("} {
			if bytes.Contains(src, []byte(literal)) {
				t.Errorf("%s hard-codes a colour (%s): the screen must draw with theme tokens", f, literal)
			}
		}
	}
}

// TestBodyTabShowsBody covers the default pane.
func TestBodyTabShowsBody(t *testing.T) {
	p := completedPresenter(t)

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	if !tt.HasText(`{"ok": true}`) {
		t.Error("expected the response body")
	}
}

// TestTabClickSwitchesPane covers the interaction and its callback.
func TestTabClickSwitchesPane(t *testing.T) {
	p := completedPresenter(t)

	var selected presenters.ResponseSubTab
	view := func(c *ui.Context) {
		Screen(c, Props{
			Presenter: p,
			OnSelectTab: func(tab presenters.ResponseSubTab) {
				selected = tab
				p.SelectSubTab(tab)
			},
		})
	}

	tt := ui.NewTester(view, 800, 400)
	tt.Frame()

	if err := tt.Click("Headers"); err != nil {
		t.Fatalf("click Headers: %v", err)
	}
	if selected != presenters.RespSubTabHeaders {
		t.Errorf("selected = %q", selected)
	}

	tt.Frame()
	// The header pane is now showing.
	if !tt.HasText("Content-Type") || !tt.HasText("application/json") {
		t.Error("expected the header rows after switching")
	}
}

// TestHeadersAreSorted covers the ordering rule: a list that reshuffles every
// frame is unreadable.
func TestHeadersAreSorted(t *testing.T) {
	p := presenters.NewResponsePresenter()
	p.HandleEvent(events.RequestQueued{UID: "r"})
	p.HandleEvent(events.ResponseReceived{
		UID: "r",
		Received: model.Result{
			Status: 200,
			Headers: model.ResponseHeaders{
				{Name: strp("Zebra"), Value: strp("z")},
				{Name: strp("alpha"), Value: strp("a")},
				{Name: strp("Beta"), Value: strp("b")},
			},
		},
	})
	p.SelectSubTab(presenters.RespSubTabHeaders)

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	// The first header drawn must be the alphabetically first, case-insensitively.
	text := strings.Join(tt.Texts(), " ")
	alpha := strings.Index(text, "alpha")
	beta := strings.Index(text, "Beta")
	zebra := strings.Index(text, "Zebra")
	if alpha < 0 || beta < 0 || zebra < 0 {
		t.Fatalf("headers missing from render: %q", text)
	}
	if !(alpha < beta && beta < zebra) {
		t.Errorf("headers not sorted: alpha=%d beta=%d zebra=%d", alpha, beta, zebra)
	}
}

// TestTimelineTabShowsPhases covers the timing table and its total.
func TestTimelineTabShowsPhases(t *testing.T) {
	p := completedPresenter(t)
	p.SelectSubTab(presenters.RespSubTabTimeline)

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	for _, want := range []string{"connect", "request-sent", "download", "total"} {
		if !tt.HasText(want) {
			t.Errorf("expected %q in the timeline pane", want)
		}
	}
}

// TestTestsTabExplainsItself avoids an empty list that reads as "all passed".
func TestTestsTabExplainsItself(t *testing.T) {
	p := completedPresenter(t)
	p.SelectSubTab(presenters.RespSubTabTests)

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	if !tt.HasText("No tests recorded for this request.") {
		t.Error("the tests tab must say it has nothing rather than render blank")
	}
}

// TestSpilledBodyIsReported covers a body too large to inline.
func TestSpilledBodyIsReported(t *testing.T) {
	p := presenters.NewResponsePresenter()
	p.HandleEvent(events.RequestQueued{UID: "r"})
	p.HandleEvent(events.ResponseReceived{
		UID: "r",
		Received: model.Result{
			Status: 200,
			Body:   model.BodyRef{Size: 1 << 20, Path: "/tmp/spool/abc"},
		},
	})

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	if !tt.HasText("Body spilled to disk") {
		t.Error("a spilled body must say so, not render blank")
	}
}

// TestEmptyBodyExplained covers a 204-style response.
func TestEmptyBodyExplained(t *testing.T) {
	p := presenters.NewResponsePresenter()
	p.HandleEvent(events.RequestQueued{UID: "r"})
	p.HandleEvent(events.ResponseReceived{
		UID:      "r",
		Received: model.Result{Status: 204, StatusText: "No Content"},
	})

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p})
	}, 800, 400)
	tt.Frame()

	if !tt.HasText("The response body is empty.") {
		t.Error("an empty body must be explained")
	}
}

// TestNilPresenterDoesNotPanic covers a wiring mistake.
func TestNilPresenterDoesNotPanic(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{})
	}, 800, 400)
	tt.Frame()
}

// TestViewIsPure is the acceptance criterion: rendering twice with no change must
// not mutate the presenter.
func TestViewIsPure(t *testing.T) {
	p := completedPresenter(t)

	before := p.ActiveSubTab()
	stateBefore := p.State()

	tt := ui.NewTester(func(c *ui.Context) {
		Screen(c, Props{Presenter: p, OnSelectTab: func(tab presenters.ResponseSubTab) {
			t.Errorf("a render called OnSelectTab for %q", tab)
		}})
	}, 800, 400)
	tt.Frame()
	tt.Frame()
	tt.Frame()

	if got := p.ActiveSubTab(); got != before {
		t.Errorf("active tab changed by rendering: %q -> %q", before, got)
	}
	if got := p.State(); got != stateBefore {
		t.Errorf("state changed by rendering: %q -> %q", stateBefore, got)
	}
}

// TestStatusColourCategories covers the mapping that drives the status colour.
func TestStatusColourCategories(t *testing.T) {
	cases := map[int]string{
		200: "success",
		301: "redirect",
		404: "client_error",
		500: "server_error",
		0:   "",
	}
	for code, want := range cases {
		p := presenters.NewResponsePresenter()
		p.HandleEvent(events.RequestQueued{UID: "r"})
		p.HandleEvent(events.ResponseReceived{UID: "r", Received: model.Result{Status: code}})
		if got := p.StatusCodeCategory(); got != want {
			t.Errorf("status %d: category = %q, want %q", code, got, want)
		}
	}
}

// TestFormatDuration covers the phase-duration rendering.
func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		500 * time.Microsecond:  "500 µs",
		2 * time.Millisecond:    "2.0 ms",
		1500 * time.Millisecond: "1.50 s",
	}
	for d, want := range cases {
		if got := formatDuration(d); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}
