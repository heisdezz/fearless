package theme

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"
)

// TabItem describes one label in a TabStrip. Count is optional; when > 0
// it is drawn as the world's count annotation.
type TabItem struct {
	Label string
	Count int
}

// TabStripProps configures TabStrip. Active is the 0-based index of the
// selected tab; OnSelect is called when the user activates an item.
// Trailing is an optional slot rendered at the right edge of the strip.
type TabStripProps struct {
	Items    []TabItem
	Active   int
	OnSelect func(index int)
	Trailing func()
}

// TabStripHeight is the standard height of the seam: 36 DIP, matching
// Lobe's nav.Tabs so the window's three tab bars speak with one voice.
const TabStripHeight float32 = 36

// TabStrip draws a flat hairline tab strip. One tab is active and carries
// the electric blue underline; the other labels are quiet. The strip has
// no container background and no border of its own, so it can be laid on
// the window's layout background or inside a card without styling seams.
//
// The tab labels are small-caps (11 DIP, letter-spaced, uppercase) to mark
// the strip as structure rather than content.
func TabStrip(c *ui.Context, p TabStripProps) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Row(c).
		FillWidth().
		Height(TabStripHeight).
		AlignItems(ui.Center).
		Gap(4).
		Padding(0, 4).
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
					Label(item.Label).
					Children(func() {
						ui.Row(c).Grow(1).AlignItems(ui.Center).Gap(4).Padding(6, 5).Children(func() {
							ui.Text(c, strings.ToUpper(item.Label)).
								FontSize(HeaderSize).
								LetterSpacing(0.2).
								TextColor(color)
							if item.Count > 0 {
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
				ui.Row(c).AlignItems(ui.Center).Shrink(0).Children(func() {
					p.Trailing()
				})
			}
		})
}

// countLabel renders a tab's figure in the world's counting form: a
// parenthesised number, so it reads as an annotation of the label rather
// than as part of it.
func countLabel(n int) string {
	return "(" + strconv.Itoa(n) + ")"
}
