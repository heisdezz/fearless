package widgets

import (
	"strings"

	"github.com/egoist/mygo/ui"

	uitheme "fearless/internal/ui/theme"
)

// MethodBadgeProps configures the MethodBadge widget.
type MethodBadgeProps struct {
	Size     float32 // Font size; defaults to 11 DIP
	PaddingX float32 // Horizontal padding; defaults to 6 DIP
	PaddingY float32 // Vertical padding; defaults to 2 DIP
	Radius   float32 // Corner radius; defaults to 3 DIP
}

// MethodBadge renders a colored badge chip for an HTTP method (GET, POST, etc.).
// Colors are dynamically selected from the active theme using Bruno's exact verb palette.
func MethodBadge(c *ui.Context, method string, props ...MethodBadgeProps) ui.Element {
	norm := strings.ToUpper(strings.TrimSpace(method))
	if norm == "" {
		norm = "GET"
	}

	p := MethodBadgeProps{
		Size:     11,
		PaddingX: 6,
		PaddingY: 2,
		Radius:   3,
	}
	if len(props) > 0 {
		if props[0].Size > 0 {
			p.Size = props[0].Size
		}
		if props[0].PaddingX > 0 {
			p.PaddingX = props[0].PaddingX
		}
		if props[0].PaddingY > 0 {
			p.PaddingY = props[0].PaddingY
		}
		if props[0].Radius > 0 {
			p.Radius = props[0].Radius
		}
	}

	methodColor := uitheme.MethodColorFor(c, norm)

	return ui.Box(c).
		Padding(p.PaddingY, p.PaddingX).
		Radius(p.Radius).
		Border(1, methodColor).
		Children(func() {
			ui.Text(c, norm).
				FontSize(p.Size).
				Bold().
				TextColor(methodColor)
		})
}
