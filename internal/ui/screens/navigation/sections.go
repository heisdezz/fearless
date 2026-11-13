package navigation

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// renderExpanded renders the sidebar: the section header strip and the
// collections body.
//
// The width comes from DraggableSideNav, which is the element that owns it.
// Sizing the body with FillWidth takes whatever width the seam currently gives
// the container, so dragging the seam resizes the content with it; pinning a
// fixed Width here would leave the inner rows at their old size.
func (s *Screen) renderExpanded(c *ui.Context, snap Snapshot, actions Actions) ui.Element {
	tk := lobetheme.TokensOf(c)
	return ui.Column(c).
		FillWidth().
		// Stretch rather than FillHeight: the body is measured inside an
		// unbounded scroll column, so a percentage height resolves to zero.
		// Stretch takes the row's height, filling the nav to the bottom.
		AlignSelf(ui.Stretch).
		Background(tk.BgContainer).
		Children(func() {
			s.renderCollectionsSection(c, snap, actions)
		})
}

func (s *Screen) renderCollectionsSection(c *ui.Context, snap Snapshot, actions Actions) {
	tk := lobetheme.TokensOf(c)

	// Header: "Collections" title + actions (Search, +, Three dots)
	ui.Row(c).
		FillWidth().
		Height(34).
		Padding(0, 10).
		AlignItems(ui.Center).
		Children(func() {
			ui.Icon(c, lucide.Box()).Size(16, 16).TextColor(tk.Text).Shrink(0)
			ui.Box(c).Width(8).Shrink(0)
			ui.Text(c, "Collections").
				FontSize(theme.TitleSize).
				FontWeight(600).
				TextColor(tk.Text).
				SingleLine().
				Shrink(1)

			ui.Box(c).Grow(1) // Spacer pushing actions to the right

			// Search / filter toggle
			widgets.IconButton(c, widgets.IconButtonProps{
				Icon:    lucide.Search(),
				Tooltip: "Search requests",
				Size:    lobetheme.Small,
				Variant: action.Text,
			}).OnClick(func() {
				s.SearchVisible = !s.SearchVisible
			})

			// Add collection (+)
			widgets.IconButton(c, widgets.IconButtonProps{
				Icon:    lucide.Plus(),
				Tooltip: "Create collection",
				Size:    lobetheme.Small,
				Variant: action.Text,
			}).OnClick(func() {
				if actions.OnCreateCollection != nil {
					actions.OnCreateCollection()
				}
			})

			// Three dots menu with Lobe DropdownMenu
			headerKey := overlay.DropdownMenu(c, &s.HeaderMenuOpen, overlay.DropdownMenuProps{
				Items: []overlay.MenuItem{
					{Key: "create", Label: "Create collection", Icon: lucide.Plus()},
					{Key: "open", Label: "Open collection", Icon: lucide.Folder()},
					{Key: "import", Label: "Import collection", Icon: lucide.Download()},
				},
			}, func(trig ui.Element) {
				widgets.StyleIconButton(c, trig, widgets.IconButtonProps{
					Icon:    lucide.MoreVertical(),
					Label:   "Collection menu",
					Tooltip: "Collection menu",
					Size:    lobetheme.Small,
					Variant: action.Text,
				})
			})
			switch headerKey {
			case "create":
				if actions.OnCreateCollection != nil {
					actions.OnCreateCollection()
				}
			case "open":
				if actions.OnOpenCollection != nil {
					actions.OnOpenCollection()
				}
			case "import":
				if actions.OnImportCollection != nil {
					actions.OnImportCollection()
				}
			}
		})

	// Filter & Action Toolbar
	ui.Column(c).FillWidth().Padding(2, 8, 4, 8).Gap(4).Children(func() {
		// Filter field: rendered when search is toggled or there is a query
		if s.SearchVisible || snap.FilterQuery != "" {
			q := snap.FilterQuery
			widgets.FieldInput(c, &q, widgets.FieldInputProps{
				Placeholder: "Filter requests…",
				Size:        lobetheme.Small,
			})
			if q != snap.FilterQuery && actions.OnFilterChanged != nil {
				actions.OnFilterChanged(q)
			}
		}

		// Action buttons row: + Request, + Folder, Open.
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			widgets.IconButton(c, widgets.IconButtonProps{
				Icon:    lucide.FilePlus(),
				Label:   "+ Request",
				Tooltip: "New request",
				Size:    lobetheme.Small,
				Variant: action.Text,
			}).OnClick(func() {
				if actions.OnCreateRequest != nil {
					actions.OnCreateRequest(snap.SelectedUID)
				}
			})

			widgets.IconButton(c, widgets.IconButtonProps{
				Icon:    lucide.FolderPlus(),
				Label:   "+ Folder",
				Tooltip: "New folder",
				Size:    lobetheme.Small,
				Variant: action.Text,
			}).OnClick(func() {
				if actions.OnCreateFolder != nil {
					actions.OnCreateFolder(snap.SelectedUID)
				}
			})

			widgets.IconButton(c, widgets.IconButtonProps{
				Icon:    lucide.SlidersHorizontal(),
				Label:   "Sort",
				Tooltip: "Sort items",
				Size:    lobetheme.Small,
				Variant: action.Text,
			})

			ui.Box(c).Grow(1)

			widgets.Button(c, "Open", widgets.ButtonProps{
				Size:    lobetheme.Small,
				Variant: action.Text,
			}).OnClick(func() {
				if actions.OnOpenCollection != nil {
					actions.OnOpenCollection()
				}
			})
		})
	})

	// Empty state when there are no nodes in collection. It sizes to its
	// content: the sidebar body sits in an unbounded scroll column, so
	// growing here would open empty space under the buttons.
	if len(snap.VisibleNodes) == 0 {
		ui.Column(c).
			FillWidth().
			AlignItems(ui.Center).
			Padding(16).
			Gap(8).
			Children(func() {
				theme.Header(c, "No requests found", tk.TextSecondary).TextAlign(ui.Center)
				ui.Text(c, "Open an existing collection or create a new one").
					FontSize(theme.DataSize).
					TextAlign(ui.Center).
					TextColor(tk.TextSecondary)
				ui.Column(c).AlignItems(ui.Center).Gap(8).Children(func() {
					widgets.Button(c, "Create Collection", widgets.ButtonProps{
						Size:    lobetheme.Small,
						Variant: action.Primary,
					}).OnClick(func() {
						if actions.OnCreateCollection != nil {
							actions.OnCreateCollection()
						}
					})
					widgets.Button(c, "Open Collection", widgets.ButtonProps{
						Size:    lobetheme.Small,
						Variant: action.Default,
					}).OnClick(func() {
						if actions.OnOpenCollection != nil {
							actions.OnOpenCollection()
						}
					})
				})
			})
		return
	}

	// Collection Tree List. Inside the DraggableSideNav scroll body the
	// available height is unbounded, so the list sizes to its rows rather
	// than growing to fill: the outer scroll already handles overflow.
	ui.Column(c).
		FillWidth().
		Children(func() {
			for _, n := range snap.VisibleNodes {
				if n == nil {
					continue
				}

				node := n
				menuOpen := s.OpenMenuUID == node.UID
				menuItems := s.buildNodeMenuItems(node)
				widgets.CollectionTreeRow(c, node, widgets.TreeRowProps{
					OnSelect: func(uid string) {
						if actions.OnSelectNode != nil {
							actions.OnSelectNode(uid)
						}
					},
					OnToggle: func(uid string) {
						if actions.OnToggleNode != nil {
							actions.OnToggleNode(uid)
						}
					},
					MenuOpen:  &menuOpen,
					MenuItems: menuItems,
					OnMenuItem: func(key string) {
						s.handleMenuItem(key, node.UID, actions)
					},
				})
				if menuOpen != (s.OpenMenuUID == node.UID) {
					if menuOpen {
						s.OpenMenuUID = node.UID
						s.HeaderMenuOpen = false
					} else {
						s.OpenMenuUID = ""
					}
				}
			}
		})
}
