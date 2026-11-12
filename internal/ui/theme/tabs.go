package theme

import (
	"strconv"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/theme"
)

// The world's section-tab strip.
//
// Three screens speak in tabs — the sidebar's section switcher, the request
// column's section strip and the response pane's sub-tabs — and they have to
// speak with one voice. A strip drawn twice is a strip that drifts, and a
// strip that drifts is a second accent: the first version of the request
// column's strip came from a component library and painted its own blue
// underline, so the window carried two blues for one selection.
//
// The strip is a hairline seam with small-caps labels on it. The selected tab
// is marked by a 2 DIP accent underline that sits on the seam; the labels stay
// secondary until selected. The underline is drawn for every tab — transparent
// when unselected — so the row cannot shift when the selection moves.

// TabItem is one tab of the world's strip.
type TabItem struct {
	// Label is the tab's name, written in prose: the strip sets it in caps.
	Label string
	// Count is an optional figure, set in the data face so the numbers in a
	// strip line up on their digits.
	Count int
}

// TabStripProps configures the world's strip.
type TabStripProps struct {
	// Items are the tabs, in order.
	Items []TabItem
	// Active is the index of the selected tab.
	Active int
	// OnSelect reports a click on a tab by index. A strip with no callback
	// still renders and still marks its selection — it just does not move.
	OnSelect func(index int)
	// Trailing, when given, is drawn at the strip's right edge: the
	// sidebar's collapse control rides there.
	Trailing func()
}

// TabStripHeight is the strip's height in DIPs.
const TabStripHeight float32 = 32

// TabStrip renders the world's section-tab strip.
func TabStrip(c *ui.Context, p TabStripProps) ui.Element {
	tk := lobe.TokensOf(c)

	return ui.Row(c).
		FillWidth().
		Height(TabStripHeight).
		AlignItems(ui.Center).
		Gap(4).
		Padding(0, 8).
		BorderWidth(0, 0, 1, 0).
		BorderColor(tk.Border).
		Children(func() {
			for i, item := range p.Items {
				i, item := i, item
				isActive := i == p.Active

				color := tk.TextSecondary
				underline := ui.Transparent
				if isActive {
					color = tk.Text
					underline = tk.Primary
				}

				ui.Column(c).
					// The tab is named by its own label, without the
					// figure: a count is an annotation of the tab, not
					// part of its name.
					Label(item.Label).
					Children(func() {
						ui.Row(c).Grow(1).AlignItems(ui.Center).Gap(4).Padding(6, 8).Children(func() {
							Header(c, item.Label, color)
							if item.Count > 0 {
								// The figure is set in the data face at
								// the header's size, in the secondary
								// ink: a count is a number the reader has
								// to be able to read, not a hint.
								DataText(c, countLabel(item.Count)).
									FontSize(HeaderSize).
									TextColor(tk.TextSecondary)
							}
						})
						ui.Box(c).FillWidth().Height(2).Background(underline)
					}).
					OnClick(func() {
						if p.OnSelect != nil {
							p.OnSelect(i)
						}
					})
			}

			ui.Box(c).Grow(1)

			if p.Trailing != nil {
				p.Trailing()
			}
		})
}

// countLabel renders a tab's figure in the world's counting form: a
// parenthesised number, so it reads as an annotation of the label rather
// than as part of it.
func countLabel(n int) string {
	return "(" + strconv.Itoa(n) + ")"
}
