package widgets

import (
	"github.com/egoist/mygo/ui"
)

// VirtualizedListProps configures a windowed list renderer.
type VirtualizedListProps struct {
	ItemCount     int
	ItemHeight    float32
	VisibleHeight float32
	ScrollOffset  float32
	Buffer        int // Extra items before and after visible window
	RenderItem    func(c *ui.Context, index int)
}

// VirtualizedList renders only the visible slice of items based on ScrollOffset and VisibleHeight,
// using top and bottom spacer boxes to preserve total scroll extent.
func VirtualizedList(c *ui.Context, props VirtualizedListProps) ui.Element {
	if props.ItemCount <= 0 || props.ItemHeight <= 0 {
		return ui.Box(c)
	}

	buffer := props.Buffer
	if buffer <= 0 {
		buffer = 2
	}

	// Calculate visible slice
	startIndex := int(props.ScrollOffset/props.ItemHeight) - buffer
	if startIndex < 0 {
		startIndex = 0
	}

	visibleCount := int(props.VisibleHeight/props.ItemHeight) + 1 + (2 * buffer)
	endIndex := startIndex + visibleCount
	if endIndex > props.ItemCount {
		endIndex = props.ItemCount
	}

	topSpacerHeight := float32(startIndex) * props.ItemHeight
	bottomSpacerHeight := float32(props.ItemCount-endIndex) * props.ItemHeight
	if bottomSpacerHeight < 0 {
		bottomSpacerHeight = 0
	}

	return ui.Column(c).FillWidth().Children(func() {
		if topSpacerHeight > 0 {
			ui.Box(c).FillWidth().Height(topSpacerHeight)
		}

		for i := startIndex; i < endIndex; i++ {
			idx := i
			ui.Box(c).FillWidth().Height(props.ItemHeight).Children(func() {
				if props.RenderItem != nil {
					props.RenderItem(c, idx)
				}
			})
		}

		if bottomSpacerHeight > 0 {
			ui.Box(c).FillWidth().Height(bottomSpacerHeight)
		}
	})
}
