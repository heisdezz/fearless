package widgets

import (
	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/presenters"
)

// TreeRowProps configures the CollectionTreeRow widget.
type TreeRowProps struct {
	OnToggle func(uid string)
	OnSelect func(uid string)
}

// CollectionTreeRow renders a single row in the collection tree sidebar.
func CollectionTreeRow(c *ui.Context, node *presenters.TreeNode, props TreeRowProps) ui.Element {
	if node == nil {
		return ui.Box(c)
	}

	tk := lobetheme.TokensOf(c)
	indent := float32(node.Level * 16)

	bg := tk.BgLayout
	if node.IsSelected {
		bg = tk.BgElevated
	}

	return ui.Row(c).
		FillWidth().
		Height(28).
		AlignItems(ui.Center).
		Padding(2, 6).
		Background(bg).
		Radius(4).
		OnClick(func() {
			if props.OnSelect != nil {
				props.OnSelect(node.UID)
			}
		}).
		Children(func() {
			// Indentation spacer
			if indent > 0 {
				ui.Box(c).Width(indent)
			}

			// Expand/collapse chevron for folders
			if node.IsFolder {
				icon := "▸"
				if node.IsExpanded {
					icon = "▾"
				}
				ui.Text(c, icon).
					FontSize(12).
					TextColor(tk.TextSecondary).
					OnClick(func() {
						if props.OnToggle != nil {
							props.OnToggle(node.UID)
						}
					})
				ui.Box(c).Width(6)
			} else {
				// Request node: method badge
				MethodBadge(c, node.Method, MethodBadgeProps{Size: 9, PaddingX: 4, PaddingY: 1})
				ui.Box(c).Width(6)
			}

			// Node Name
			txt := ui.Text(c, node.Name).
				FontSize(12).
				TextColor(tk.Text)
			if node.IsFolder {
				txt.Bold()
			}

			// Dirty indicator
			if node.IsDirty {
				ui.Box(c).Width(6)
				ui.Box(c).
					Size(5, 5).
					Radius(2.5).
					Background(tk.Warning)
			}

			ui.Box(c).Grow(1) // Spacer
		})
}
