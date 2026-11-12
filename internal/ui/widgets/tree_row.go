package widgets

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/presenters"
)

// TreeRowProps configures the CollectionTreeRow widget.
type TreeRowProps struct {
	OnToggle func(uid string)
	OnSelect func(uid string)
}

// CollectionTreeRow renders a single row in the collection tree sidebar.
// It uses exact dimensions (26 px / 1.6 rem height) and RequestMethod (Decision R-1).
// Selected state uses the Graphite Console electric blue accent (D-017, B-024).
func CollectionTreeRow(c *ui.Context, node *presenters.TreeNode, props TreeRowProps) ui.Element {
	if node == nil {
		return ui.Box(c)
	}

	tk := lobetheme.TokensOf(c)
	indent := float32(node.Level * 16)

	row := ui.Row(c).
		FillWidth().
		Height(26).
		AlignItems(ui.Center).
		Padding(2, 6).
		Radius(4).
		Cursor(ui.CursorPointer).
		OnClick(func() {
			if props.OnSelect != nil {
				props.OnSelect(node.UID)
			}
		})

	if node.IsSelected {
		// The selection is marked by the accent tint and the accent ink.
		// The bar is one pixel: a thicker coloured edge on a list row is
		// the floor's ban, and the tint already says which row is open.
		row.Background(tk.PrimaryBg).
			BorderWidth(0, 0, 0, 1).
			BorderColor(tk.Primary)
	} else {
		row.Background(tk.BgLayout)
	}

	return row.Children(func() {
		// Indentation spacer
		if indent > 0 {
			ui.Box(c).Width(indent)
		}

		// Expand/collapse chevron for folders
		if node.IsFolder {
			icon := lucide.ChevronRight()
			chevronLabel := "Expand " + node.Name
			if node.IsExpanded {
				icon = lucide.ChevronDown()
				chevronLabel = "Collapse " + node.Name
			}
			ui.Icon(c, icon).
				Size(12, 12).
				TextColor(tk.TextSecondary).
				Label(chevronLabel).
				OnClick(func() {
					if props.OnToggle != nil {
						props.OnToggle(node.UID)
					}
				})
			ui.Box(c).Width(6)
		} else {
			// Request node: compact RequestMethod label (Decision R-1)
			RequestMethod(c, node.Method)
			ui.Box(c).Width(8)
		}

		// Node Name
		txt := ui.Text(c, node.Name).
			FontSize(12)
		if node.IsSelected {
			txt.TextColor(tk.PrimaryText)
		} else {
			txt.TextColor(tk.Text)
		}
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
