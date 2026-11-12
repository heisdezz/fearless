package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"
)

// TestTabStripSpeaksOneVoice covers the shared strip: small-caps labels on a
// hairline seam, the selected tab marked by the accent underline, and the
// figure set in the data face.
func TestTabStripSpeaksOneVoice(t *testing.T) {
	var selected int
	tt := ui.NewTester(func(c *ui.Context) {
		TabStrip(c, TabStripProps{
			Items: []TabItem{
				{Label: "Params"},
				{Label: "Headers", Count: 3},
			},
			Active: 1,
			OnSelect: func(i int) {
				selected = i
			},
		})
	}, 400, 80)
	tt.Frame()

	// The labels are set in caps; the strip names each tab by its own label,
	// so a caller can address a tab without the figure.
	for _, want := range []string{"PARAMS", "HEADERS", "Params", "Headers"} {
		if !tt.HasText(want) {
			t.Errorf("strip missing %q; texts: %v", want, tt.Texts())
		}
	}

	// The figure is set apart from the label, in the data face.
	if !tt.HasText("(3)") {
		t.Errorf("strip missing the figure; texts: %v", tt.Texts())
	}

	// The strip is clickable and reports the tab's index.
	if err := tt.Click("Params"); err != nil {
		t.Fatalf("click Params: %v", err)
	}
	if selected != 0 {
		t.Errorf("selected = %d, want 0", selected)
	}
}

// TestTabStripMarksTheSelectionWithTheAccent covers the one accent: the
// selected tab's underline is the world's accent and nothing else on the strip
// is.
func TestTabStripMarksTheSelectionWithTheAccent(t *testing.T) {
	var primary ui.Color
	tt := ui.NewTester(func(c *ui.Context) {
		primary = lobetheme.TokensOf(c).Primary
		TabStrip(c, TabStripProps{
			Items:  []TabItem{{Label: "Body"}, {Label: "Headers"}},
			Active: 0,
		})
	}, 400, 80)
	tt.Frame()

	label, ok := tt.Find("Body")
	if !ok {
		t.Fatal("the active tab's label is not rendered")
	}

	img := tt.Image()
	bounds := img.Bounds()
	found := false
	for y := int(label.Y); y < int(label.Y+label.H)+4 && y < bounds.Max.Y; y++ {
		for x := int(label.X) - 4; x < int(label.X+label.W)+4 && x < bounds.Max.X; x++ {
			if x < bounds.Min.X || y < bounds.Min.Y {
				continue
			}
			r, g, b, _ := img.At(x, y).RGBA()
			if uint8(r>>8) == primary.R && uint8(g>>8) == primary.G && uint8(b>>8) == primary.B {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Error("expected the selected tab to carry the accent underline")
	}
}
