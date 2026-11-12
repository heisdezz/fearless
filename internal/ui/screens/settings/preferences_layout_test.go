package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

// TestPreferencesLayout_TopAnchored pins the body of the Preferences
// screen to the top of the window: the tab rail and the settings pane
// must start right below the header, not float centred mid-window.
func TestPreferencesLayout_TopAnchored(t *testing.T) {
	p := presenters.NewPreferencesPresenter(model.DefaultSettings(), &mockPrefSaver{}, nil, &mockOAuthCleaner{})

	tt := ui.NewTester(func(c *ui.Context) {
		PreferencesScreen(c, p)
	}, 900, 700)
	tt.Frame()

	header, ok := tt.Find("Preferences")
	if !ok {
		t.Fatal("expected 'Preferences' header")
	}
	first, ok := tt.Find("General")
	if !ok {
		t.Fatal("expected 'General' nav item")
	}
	last, ok := tt.Find("OAuth")
	if !ok {
		t.Fatal("expected 'OAuth' nav item")
	}
	pane, ok := tt.Find("General Settings")
	if !ok {
		t.Fatal("expected 'General Settings' pane heading")
	}

	t.Logf("header           y=%.1f h=%.1f", header.Y, header.H)
	t.Logf("nav 'General'    y=%.1f h=%.1f", first.Y, first.H)
	t.Logf("nav 'OAuth'      y=%.1f h=%.1f", last.Y, last.H)
	t.Logf("pane heading     y=%.1f h=%.1f", pane.Y, pane.H)

	// The nav rail must begin directly under the header (within one gap).
	if first.Y < header.Y+header.H || first.Y > header.Y+header.H+60 {
		t.Errorf("nav rail floats: first item y=%.1f, header bottom=%.1f",
			first.Y, header.Y+header.H)
	}

	// The pane heading must sit at the same top edge as the rail, not
	// somewhere in the middle of the window.
	if pane.Y > first.Y+80 {
		t.Errorf("pane heading floats below the rail top: pane y=%.1f, rail y=%.1f",
			pane.Y, first.Y)
	}
	if pane.Y > 400 {
		t.Errorf("pane heading is centred mid-window: y=%.1f (window h=700)", pane.Y)
	}
}
