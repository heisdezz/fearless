package widgets

import (
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"
)

// TestIconButtonCarriesItsName covers the accessibility contract of an icon
// button: the icon draws no word, so the control names itself — for assistive
// technology and for a test that has to find it.
func TestIconButtonCarriesItsName(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		IconButton(c, IconButtonProps{
			Icon:    lucide.Menu(),
			Label:   "Menu",
			Tooltip: "Open the menu",
			Size:    lobetheme.Small,
			Variant: action.Text,
		})
	}, 200, 80)
	tt.Frame()

	if !tt.HasText("Menu") {
		t.Errorf("expected the icon button to be named Menu; texts: %v", tt.Texts())
	}
}

// TestIconButtonDrawsNoGlyph keeps the app's icons drawn: a Unicode character
// standing in for an icon is the floor's ban, and this is the test that says
// so in the package that owns the buttons.
func TestIconButtonDrawsNoGlyph(t *testing.T) {
	src, err := os.ReadFile("controls.go")
	if err != nil {
		t.Fatalf("read controls.go: %v", err)
	}
	for _, glyph := range []string{"☰", "⚙", "✕", "«", "»"} {
		if strings.Contains(string(src), "\""+glyph+"\"") {
			t.Errorf("controls.go still uses the glyph %q as a button label", glyph)
		}
	}
}
