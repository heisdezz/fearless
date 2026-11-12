package history

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"fearless/internal/history"
)

func TestHistoryScreen_EmptyAndPopulated(t *testing.T) {
	snap := Snapshot{
		Summary: history.Summary{
			Total:  0,
			Passed: 0,
			Failed: 0,
		},
	}

	var (
		selectedID    string
		deletedID     string
		cleared       bool
		replayedID    string
		filterQuery   string
		failedToggled bool
	)

	actions := Actions{
		OnSelectEntry: func(id string) {
			selectedID = id
		},
		OnDeleteEntry: func(id string) {
			deletedID = id
		},
		OnClearHistory: func() {
			cleared = true
		},
		OnReplayEntry: func(id string) {
			replayedID = id
		},
		OnFilterChanged: func(q string) {
			filterQuery = q
		},
		OnToggleFailed: func(failed bool) {
			failedToggled = failed
		},
	}

	screen := NewScreen(snap, actions)

	view := func(c *ui.Context) {
		screen.View(c)
	}

	tt := ui.NewTester(view, 800, 600)
	tt.Frame()

	// Empty state assertions
	if !tt.HasText("Request History") {
		t.Error("expected 'Request History' header text")
	}
	if !tt.HasText("No request history recorded") {
		t.Error("expected empty state message")
	}

	// Now populate with entries
	entries := []history.Entry{
		{
			ID:          "hist-1",
			ExecutedAt:  time.Now().Add(-5 * time.Minute),
			Method:      "GET",
			URL:         "https://api.example.com/users",
			ItemName:    "Get Users",
			Status:      200,
			StatusText:  "OK",
			DurationMs:  124,
			Size:        1024,
			Environment: "Development",
			RequestHeaders: map[string][]string{
				"Accept": {"application/json"},
			},
			ResponseHeaders: map[string][]string{
				"Content-Type": {"application/json; charset=utf-8"},
			},
		},
		{
			ID:         "hist-2",
			ExecutedAt: time.Now().Add(-2 * time.Minute),
			Method:     "POST",
			URL:        "https://api.example.com/login",
			ItemName:   "User Login",
			Status:     401,
			StatusText: "Unauthorized",
			DurationMs: 45,
			Size:       128,
			Error:      "invalid credentials",
		},
	}

	screen.Snapshot = Snapshot{
		Entries:    entries,
		SelectedID: "hist-1",
		Summary: history.Summary{
			Total:  2,
			Passed: 1,
			Failed: 1,
		},
	}

	tt.Frame()

	// Verify entries exist
	if !tt.HasText("Get Users") {
		t.Error("expected 'Get Users' entry visible")
	}
	if !tt.HasText("User Login") {
		t.Error("expected 'User Login' entry visible")
	}
	if !tt.HasText("Total: 2") {
		t.Error("expected 'Total: 2' summary text")
	}
	if !tt.HasText("Passed: 1") {
		t.Error("expected 'Passed: 1' summary text")
	}
	if !tt.HasText("Failed: 1") {
		t.Error("expected 'Failed: 1' summary text")
	}

	// Verify detail pane for selected entry "hist-1"
	if !tt.HasText("Response Headers") {
		t.Error("expected Response Headers section in detail view")
	}
	if !tt.HasText("Accept") {
		t.Error("expected Request Header 'Accept' in detail view")
	}

	// Click delete button on detail view
	if err := tt.Click("Delete"); err != nil {
		t.Errorf("failed to click Delete: %v", err)
	} else if deletedID != "hist-1" {
		t.Errorf("expected deletedID to be 'hist-1', got %q", deletedID)
	}

	// Click Clear History
	if err := tt.Click("Clear History"); err != nil {
		t.Errorf("failed to click Clear History: %v", err)
	} else if !cleared {
		t.Error("expected OnClearHistory to be triggered")
	}

	// Click Replay
	if err := tt.Click("Replay"); err != nil {
		t.Errorf("failed to click Replay: %v", err)
	} else if replayedID == "" {
		t.Error("expected OnReplayEntry to be triggered")
	}

	_ = filterQuery
	_ = failedToggled
	_ = selectedID
}
