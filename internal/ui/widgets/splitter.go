package widgets

import (
	"github.com/egoist/mygo/ui"
)

// SplitOrientation defines the layout direction of split panes.
type SplitOrientation string

const (
	SplitHorizontal SplitOrientation = "horizontal" // side by side (left | right)
	SplitVertical   SplitOrientation = "vertical"   // stacked (top / bottom)
)

// Splitter renders two resizable panes according to ratio and orientation.
func Splitter(
	c *ui.Context,
	ratio *float32,
	orientation SplitOrientation,
	first func(*ui.Context),
	second func(*ui.Context),
) ui.Element {
	if ratio == nil {
		defRatio := float32(50)
		ratio = &defRatio
	} else if *ratio <= 1.0 && *ratio > 0.0 {
		// Convert fraction (e.g. 0.4) to percentage (40)
		*ratio = *ratio * 100
	}

	firstEl := func() {
		if first != nil {
			first(c)
		}
	}
	secondEl := func() {
		if second != nil {
			second(c)
		}
	}

	if orientation == SplitVertical {
		return ui.SplitVerticalPercent(c, ratio, firstEl, secondEl).Fill()
	}
	return ui.SplitPercent(c, ratio, firstEl, secondEl).Fill()
}
