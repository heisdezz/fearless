package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

// TestResponsePaneTabsAreClickable covers the wiring behind the response tabs:
// clicking Headers must move the presenter's active sub-tab. The app used to
// pass no OnSelectTab, so the tabs rendered and did nothing.
func TestResponsePaneTabsAreClickable(t *testing.T) {
	a := New()

	// A completed response, so the pane has something to show.
	a.ResponsePresenter.HandleEvent(events.RequestQueued{UID: "r1"})
	a.ResponsePresenter.HandleEvent(events.ResponseReceived{
		UID: "r1",
		Received: model.Result{
			Status:     200,
			StatusText: "OK",
			Headers: model.ResponseHeaders{
				{Name: strPtr("Content-Type"), Value: strPtr("application/json")},
			},
		},
	})
	if got := a.ResponsePresenter.ActiveSubTab(); got != presenters.RespSubTabBody {
		t.Fatalf("initial tab = %q, want body", got)
	}

	// Render just the pane, then click the Headers tab.
	tt := ui.NewTester(func(c *ui.Context) {
		a.ShellScreen.ResponsePane(c)
	}, 800, 400)
	tt.Frame()

	if err := tt.Click("Headers"); err != nil {
		t.Fatalf("click Headers: %v", err)
	}
	tt.Frame()

	if got := a.ResponsePresenter.ActiveSubTab(); got != presenters.RespSubTabHeaders {
		t.Errorf("tab after clicking Headers = %q, want headers", got)
	}
}
