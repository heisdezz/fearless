package widgets

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/presenters"
)

// TabBarProps configures the RequestTabBar widget.
type TabBarProps struct {
	OnSelect func(uid string)
	OnClose  func(uid string)
	OnNewTab func()
}

// RequestTabBar renders the horizontal reorderable request tab strip.
// It uses exact 34 px height, no background hover tint (R-2), and italic preview tabs (R-3).
func RequestTabBar(c *ui.Context, tabs []*presenters.TabItem, activeUID string, props TabBarProps) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Row(c).FillWidth().Height(34).AlignItems(ui.Center).Background(tk.BgLayout).Children(func() {
		// Tabs list
		for _, t := range tabs {
			tab := t
			isActive := tab.UID == activeUID

			bg := tk.BgLayout
			borderCol := tk.Border
			if isActive {
				bg = tk.BgContainer
				borderCol = tk.Primary
			}

			ui.Row(c).
				AlignItems(ui.Center).
				Gap(6).
				Padding(4, 10).
				Background(bg).
				Border(1, borderCol).
				Radius(4, 4, 0, 0).
				OnClick(func() {
					if props.OnSelect != nil {
						props.OnSelect(tab.UID)
					}
				}).
				Children(func() {
					// Method badge
					MethodBadge(c, tab.Method, MethodBadgeProps{Size: 9, PaddingX: 4, PaddingY: 1})

					// Title (Decision R-3: preview tabs are italic only)
					title := tab.Title
					if title == "" {
						title = "Untitled"
					}
					txt := ui.Text(c, title).
						FontSize(12).
						TextColor(tk.Text)
					if tab.IsTransient {
						txt.Italic()
					}

					// Dirty dot
					if tab.IsDirty {
						ui.Box(c).
							Size(6, 6).
							Radius(3).
							Background(tk.Warning)
					}

					// Close button
					IconButton(c, IconButtonProps{
						Icon:    lucide.X(),
						Label:   "Close tab",
						Tooltip: "Close tab",
						Size:    lobetheme.Small,
						Variant: action.Text,
					}).OnClick(func() {
						if props.OnClose != nil {
							props.OnClose(tab.UID)
						}
					})
				})
		}

		// New Tab button "+"
		if props.OnNewTab != nil {
			ui.Box(c).Padding(0, 4).Children(func() {
				IconButton(c, IconButtonProps{
					Icon:    lucide.Plus(),
					Label:   "New tab",
					Tooltip: "New tab",
					Size:    lobetheme.Small,
					Variant: action.Text,
				}).OnClick(func() {
					props.OnNewTab()
				})
			})
		}

		ui.Box(c).Grow(1) // Spacer for remaining bar width
	})
}
