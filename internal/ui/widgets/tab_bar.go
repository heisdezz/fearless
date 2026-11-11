package widgets

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
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
func RequestTabBar(c *ui.Context, tabs []*presenters.TabItem, activeUID string, props TabBarProps) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Row(c).FillWidth().Height(36).AlignItems(ui.Center).Background(tk.BgLayout).Children(func() {
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

					// Title
					title := tab.Title
					if title == "" {
						title = "Untitled"
					}
					ui.Text(c, title).
						FontSize(12).
						TextColor(tk.Text)

					// Dirty dot
					if tab.IsDirty {
						ui.Box(c).
							Size(6, 6).
							Radius(3).
							Background(tk.Warning)
					}

					// Close button
					action.Button(c, "✕", action.ButtonProps{
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
				action.Button(c, "+", action.ButtonProps{
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
