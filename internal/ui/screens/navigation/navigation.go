package navigation

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/presenters"
	"fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// Snapshot is an immutable snapshot of presenters data for rendering navigation views.
type Snapshot struct {
	ActiveSection     presenters.SidebarSection
	IsCollapsed       bool
	SidebarWidth      float32
	FilterQuery       string
	VisibleNodes      []*presenters.TreeNode
	SelectedUID       string
	CollectionName    string
	CollectionUID     string
	Environments      []string
	ActiveEnvironment string
}

// Actions captures all intent-based callbacks dispatched from navigation view interactions.
type Actions struct {
	OnSelectSection      func(presenters.SidebarSection)
	OnToggleCollapse     func()
	OnFilterChanged      func(string)
	OnSelectNode         func(uid string)
	OnToggleNode         func(uid string)
	OnOpenCollection     func()
	OnCreateCollection   func()
	OnImportCollection   func()
	OnCreateRequest      func(parentUID string)
	OnCreateFolder       func(parentUID string)
	OnDeleteItem         func(uid string)
	OnSelectEnvironment  func(envName string)
	OnManageEnvironments func()
	OnRunCollection      func(colUID string)
	OnCollectionSettings func(colUID string)
	OnCloneCollection    func(colUID string)
	OnRenameItem         func(uid string)
	OnRevealItem         func(uid string)
	OnOpenTerminal       func(uid string)
	OnCreateMockServer   func()
	OnCollapseAll        func(colUID string)
}

// TakeSnapshot creates an immutable plain data Snapshot from the given sidebar presenter.
func TakeSnapshot(sidebar *presenters.SidebarPresenter) Snapshot {
	if sidebar == nil {
		return Snapshot{ActiveSection: presenters.SidebarSectionCollections}
	}

	colPres := sidebar.Collections()
	envPres := sidebar.Environments()

	snap := Snapshot{
		ActiveSection: sidebar.ActiveSection(),
		IsCollapsed:   sidebar.IsCollapsed(),
		SidebarWidth:  sidebar.Width(),
	}

	if colPres != nil {
		snap.FilterQuery = colPres.Filter()
		nodes := colPres.VisibleNodes()
		if len(nodes) > 0 {
			snap.VisibleNodes = make([]*presenters.TreeNode, len(nodes))
			for i, n := range nodes {
				if n != nil {
					cp := *n
					snap.VisibleNodes[i] = &cp
				}
			}
		}
		snap.SelectedUID = colPres.SelectedUID()
		cols := colPres.Collections()
		if len(cols) > 0 && cols[0] != nil {
			snap.CollectionName = cols[0].Name
			snap.CollectionUID = cols[0].UID
		}
	}

	if envPres != nil {
		envs := envPres.Environments()
		for _, env := range envs {
			if env != nil {
				snap.Environments = append(snap.Environments, env.Name)
			}
		}
		snap.ActiveEnvironment = envPres.ActiveEnvironmentName()
	}

	return snap
}

// Screen encapsulates the state and action callbacks for the navigation views.
type Screen struct {
	Snapshot        Snapshot
	Actions         Actions
	HeaderMenuOpen  bool
	SearchVisible   bool
	OpenMenuUID     string
	MockServersOpen bool
}

// NewScreen creates an initialized Screen from a sidebar presenter and command actions.
func NewScreen(sidebar *presenters.SidebarPresenter, actions Actions) *Screen {
	return &Screen{
		Snapshot:        TakeSnapshot(sidebar),
		Actions:         actions,
		MockServersOpen: false,
	}
}

// Sidebar renders the navigation sidebar according to the layout state.
func (s *Screen) Sidebar(c *ui.Context, nav lobe.SideNavState) ui.Element {
	snap := s.Snapshot
	actions := s.Actions

	// Detect if collapsed (either explicitly in Snapshot or during drag/threshold in nav)
	isRail := snap.IsCollapsed || nav.Collapse > 0.5

	if isRail {
		return s.renderRail(c, snap, actions, nav)
	}

	return s.renderExpanded(c, snap, actions)
}

func (s *Screen) renderRail(c *ui.Context, snap Snapshot, actions Actions, nav lobe.SideNavState) ui.Element {
	tk := lobetheme.TokensOf(c)
	railWidth := nav.RailWidth
	if railWidth <= 0 {
		railWidth = 64
	}

	return ui.Column(c).
		FillHeight().
		Width(railWidth).
		Background(tk.BgContainer).
		AlignItems(ui.Center).
		Padding(12, 4).
		Children(func() {
			// Rail Section Buttons
			s.renderRailSectionButton(c, "Collections", lucide.FolderOpen(), snap.ActiveSection == presenters.SidebarSectionCollections, func() {
				if actions.OnSelectSection != nil {
					actions.OnSelectSection(presenters.SidebarSectionCollections)
				}
			})
			ui.Box(c).Height(8)
			s.renderRailSectionButton(c, "History", lucide.History(), snap.ActiveSection == presenters.SidebarSectionHistory, func() {
				if actions.OnSelectSection != nil {
					actions.OnSelectSection(presenters.SidebarSectionHistory)
				}
			})
			ui.Box(c).Height(8)
			s.renderRailSectionButton(c, "Runner", lucide.Play(), snap.ActiveSection == presenters.SidebarSectionRunner, func() {
				if actions.OnSelectSection != nil {
					actions.OnSelectSection(presenters.SidebarSectionRunner)
				}
			})

			ui.Box(c).Grow(1)

			// Expand button at bottom of rail
			widgets.IconButton(c, widgets.IconButtonProps{
				Icon:    lucide.PanelLeftOpen(),
				Label:   "Expand sidebar",
				Tooltip: "Expand sidebar",
				Variant: action.Text,
				Size:    lobetheme.Small,
			}).OnClick(func() {
				if actions.OnToggleCollapse != nil {
					actions.OnToggleCollapse()
				}
			})
		})
}

func (s *Screen) renderRailSectionButton(c *ui.Context, tooltip string, icon *ui.SVG, active bool, onClick func()) {
	var variant action.ButtonVariant
	if active {
		variant = action.Primary
	} else {
		variant = action.Text
	}

	widgets.IconButton(c, widgets.IconButtonProps{
		Icon:    icon,
		Label:   tooltip,
		Tooltip: tooltip,
		Variant: variant,
		Size:    lobetheme.Small,
	}).OnClick(onClick)
}

func (s *Screen) renderExpanded(c *ui.Context, snap Snapshot, actions Actions) ui.Element {
	tk := lobetheme.TokensOf(c)
	width := snap.SidebarWidth
	if width <= 0 {
		width = presenters.DefaultSidebarWidth
	}

	return ui.Column(c).
		FillHeight().
		Width(width).
		Background(tk.BgContainer).
		BorderWidth(0, 1, 0, 0).
		BorderColor(tk.BorderSecondary).
		Children(func() {
			// 1. Shared Section Tab Strip
			theme.TabStrip(c, theme.TabStripProps{
				Items: []theme.TabItem{
					{Label: "Collections"},
					{Label: "History"},
					{Label: "Runner"},
				},
				Active: sectionIndex(snap.ActiveSection),
				OnSelect: func(i int) {
					if actions.OnSelectSection != nil && i >= 0 && i < len(sections) {
						actions.OnSelectSection(sections[i])
					}
				},
				Trailing: func() {
					widgets.IconButton(c, widgets.IconButtonProps{
						Icon:    lucide.PanelLeftClose(),
						Label:   "Collapse sidebar",
						Tooltip: "Collapse sidebar",
						Variant: action.Text,
						Size:    lobetheme.Small,
					}).OnClick(func() {
						if actions.OnToggleCollapse != nil {
							actions.OnToggleCollapse()
						}
					})
				},
			})

			// 2. Active Section Body
			switch snap.ActiveSection {
			case presenters.SidebarSectionHistory:
				s.renderHistorySection(c, snap, actions)
			case presenters.SidebarSectionRunner:
				s.renderRunnerSection(c, snap, actions)
			default:
				s.renderCollectionsSection(c, snap, actions)
			}

			// 3. Mock Servers dock at bottom (replicated from collection_with_menu_opened.png)
			// Collapsed by default — click to toggle accordion open/closed.
			ui.Column(c).
				FillWidth().
				Background(tk.BgContainer).
				BorderWidth(1, 0, 0, 0).
				BorderColor(tk.BorderSecondary).
				Children(func() {
					ui.Row(c).
						FillWidth().
						Height(32).
						Padding(0, 10).
						AlignItems(ui.Center).
						Children(func() {
							// Left clickable toggle area (Icon + Title + Chevron)
							ui.ButtonBase(c).
								Row().
								Grow(1).
								AlignItems(ui.Center).
								Cursor(ui.CursorPointer).
								OnClick(func() {
									s.MockServersOpen = !s.MockServersOpen
								}).
								Children(func() {
									ui.Icon(c, lucide.Server()).Size(14, 14).TextColor(tk.TextSecondary).Shrink(0)
									ui.Box(c).Width(6).Shrink(0)
									ui.Text(c, "Mock Servers").FontSize(11).Bold().TextColor(tk.TextSecondary).SingleLine().Shrink(1)
									ui.Box(c).Width(4)
									if s.MockServersOpen {
										ui.Icon(c, lucide.ChevronDown()).Size(12, 12).TextColor(tk.TextSecondary).Shrink(0)
									} else {
										ui.Icon(c, lucide.ChevronRight()).Size(12, 12).TextColor(tk.TextSecondary).Shrink(0)
									}
								})

							// Action button: + (Create mock server)
							widgets.IconButton(c, widgets.IconButtonProps{
								Icon:    lucide.Plus(),
								Tooltip: "Create mock server",
								Size:    lobetheme.Small,
								Variant: action.Text,
							}).OnClick(func() {
								if actions.OnCreateMockServer != nil {
									actions.OnCreateMockServer()
								}
							})
						})

					// Section body (only when expanded)
					if s.MockServersOpen {
						ui.Column(c).
							FillWidth().
							Padding(8, 12, 12, 12).
							Gap(6).
							Children(func() {
								ui.Text(c, "No mock servers configured").
									FontSize(11).
									TextColor(tk.TextTertiary)
							})
					}
				})
		})
}

// sections are the sidebar's sections, in the order the switcher shows them.
var sections = []presenters.SidebarSection{
	presenters.SidebarSectionCollections,
	presenters.SidebarSectionHistory,
	presenters.SidebarSectionRunner,
}

// sectionIndex maps the active section onto its place in the strip.
func sectionIndex(section presenters.SidebarSection) int {
	for i, s := range sections {
		if s == section {
			return i
		}
	}
	return 0
}

func (s *Screen) renderMenuItem(c *ui.Context, icon *ui.SVG, label, badge string, onClick func()) {
	tk := lobetheme.TokensOf(c)
	ui.Row(c).
		FillWidth().
		Height(26).
		Padding(0, 10).
		AlignItems(ui.Center).
		Cursor(ui.CursorPointer).
		Radius(4).
		OnClick(onClick).
		Children(func() {
			if icon != nil {
				ui.Icon(c, icon).Size(13, 13).TextColor(tk.TextSecondary).Shrink(0)
				ui.Box(c).Width(8).Shrink(0)
			}
			ui.Text(c, label).FontSize(12).TextColor(tk.Text).SingleLine().Shrink(1)
			if badge != "" {
				ui.Box(c).Grow(1)
				ui.Box(c).
					Padding(1, 4).
					Radius(3).
					Background(tk.BgContainerSecondary).
					Children(func() {
						ui.Text(c, badge).FontSize(9).TextColor(tk.TextTertiary)
					})
			}
		})
}

func (s *Screen) renderCollectionContextMenu(c *ui.Context, nodeUID string, actions Actions) {
	tk := lobetheme.TokensOf(c)
	ui.Box(c).Padding(2, 6).Children(func() {
		ui.Column(c).
			FillWidth().
			Background(tk.BgElevated).
			Radius(6).
			BorderWidth(1).
			BorderColor(tk.BorderSecondary).
			Padding(4, 0).
			Children(func() {
				s.renderMenuItem(c, lucide.FilePlus(), "New Request", "", func() {
					s.OpenMenuUID = ""
					if actions.OnCreateRequest != nil {
						actions.OnCreateRequest(nodeUID)
					}
				})
				s.renderMenuItem(c, lucide.FolderPlus(), "New Folder", "", func() {
					s.OpenMenuUID = ""
					if actions.OnCreateFolder != nil {
						actions.OnCreateFolder(nodeUID)
					}
				})
				s.renderMenuItem(c, lucide.Play(), "Run", "", func() {
					s.OpenMenuUID = ""
					if actions.OnRunCollection != nil {
						actions.OnRunCollection(nodeUID)
					}
				})
				s.renderMenuItem(c, lucide.Copy(), "Clone", "", func() {
					s.OpenMenuUID = ""
					if actions.OnCloneCollection != nil {
						actions.OnCloneCollection(nodeUID)
					}
				})
				s.renderMenuItem(c, lucide.Pencil(), "Rename", "", func() {
					s.OpenMenuUID = ""
					if actions.OnRenameItem != nil {
						actions.OnRenameItem(nodeUID)
					}
				})
				s.renderMenuItem(c, lucide.Share2(), "Share", "", func() {
					s.OpenMenuUID = ""
				})
				s.renderMenuItem(c, lucide.BookOpen(), "Generate Docs", "", func() {
					s.OpenMenuUID = ""
				})
				s.renderMenuItem(c, lucide.FoldVertical(), "Collapse", "", func() {
					s.OpenMenuUID = ""
					if actions.OnCollapseAll != nil {
						actions.OnCollapseAll(nodeUID)
					}
				})
				s.renderMenuItem(c, lucide.Folder(), "Reveal in File Manager", "", func() {
					s.OpenMenuUID = ""
					if actions.OnRevealItem != nil {
						actions.OnRevealItem(nodeUID)
					}
				})
				s.renderMenuItem(c, lucide.Server(), "Create Mock server", "Beta", func() {
					s.OpenMenuUID = ""
					if actions.OnCreateMockServer != nil {
						actions.OnCreateMockServer()
					}
				})

				// Divider line
				ui.Box(c).FillWidth().Height(1).Background(tk.BorderSecondary).Margin(2, 0)

				s.renderMenuItem(c, lucide.Settings(), "Settings", "", func() {
					s.OpenMenuUID = ""
					if actions.OnCollectionSettings != nil {
						actions.OnCollectionSettings(nodeUID)
					}
				})
				s.renderMenuItem(c, lucide.Terminal(), "Open in Terminal", "", func() {
					s.OpenMenuUID = ""
					if actions.OnOpenTerminal != nil {
						actions.OnOpenTerminal(nodeUID)
					}
				})
				s.renderMenuItem(c, lucide.X(), "Remove", "", func() {
					s.OpenMenuUID = ""
					if actions.OnDeleteItem != nil {
						actions.OnDeleteItem(nodeUID)
					}
				})
			})
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

			// Three dots menu
			widgets.IconButton(c, widgets.IconButtonProps{
				Icon:    lucide.MoreVertical(),
				Tooltip: "Collection menu",
				Size:    lobetheme.Small,
				Variant: action.Text,
			}).OnClick(func() {
				s.HeaderMenuOpen = !s.HeaderMenuOpen
			})
		})

	// Header Dropdown Menu (replicated from refs/new_refs/sidebar.png)
	if s.HeaderMenuOpen {
		ui.Box(c).Padding(2, 6).Children(func() {
			ui.Column(c).
				FillWidth().
				Background(tk.BgElevated).
				Radius(6).
				BorderWidth(1).
				BorderColor(tk.BorderSecondary).
				Padding(4, 0).
				Children(func() {
					s.renderMenuItem(c, lucide.Plus(), "Create collection", "", func() {
						s.HeaderMenuOpen = false
						if actions.OnCreateCollection != nil {
							actions.OnCreateCollection()
						}
					})
					s.renderMenuItem(c, lucide.Folder(), "Open collection", "", func() {
						s.HeaderMenuOpen = false
						if actions.OnOpenCollection != nil {
							actions.OnOpenCollection()
						}
					})
					s.renderMenuItem(c, lucide.Download(), "Import collection", "", func() {
						s.HeaderMenuOpen = false
						if actions.OnImportCollection != nil {
							actions.OnImportCollection()
						}
					})
				})
		})
	}

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

	// Empty state when there are no nodes in collection
	if len(snap.VisibleNodes) == 0 {
		ui.Column(c).
			FillWidth().
			Grow(1).
			Center().
			Padding(16).
			Gap(8).
			Children(func() {
				theme.PanelHeader(c, "No requests found")
				ui.Text(c, "Open an existing collection or create a new one").
					FontSize(11).
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

	// Collection Tree List
	ui.Scroll(c).
		FillWidth().
		Grow(1).
		Children(func() {
			for _, n := range snap.VisibleNodes {
				if n == nil {
					continue
				}

				widgets.CollectionTreeRow(c, n, widgets.TreeRowProps{
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
					OnMenu: func(uid string) {
						if s.OpenMenuUID == uid {
							s.OpenMenuUID = ""
						} else {
							s.OpenMenuUID = uid
							s.HeaderMenuOpen = false
						}
					},
				})

				// If menu opened for this node, render context menu (replicated from refs/new_refs/collection_with_menu_opened.png)
				if s.OpenMenuUID == n.UID {
					s.renderCollectionContextMenu(c, n.UID, actions)
				}
			}
		})
}

func (s *Screen) renderHistorySection(c *ui.Context, snap Snapshot, actions Actions) {
	tk := lobetheme.TokensOf(c)
	ui.Column(c).
		FillWidth().
		Grow(1).
		Center().
		Padding(16).
		Gap(6).
		Children(func() {
			theme.PanelHeader(c, "Request History")
			ui.Text(c, "Past request executions will appear here").
				FontSize(11).
				TextColor(tk.TextSecondary)
		})
}

func (s *Screen) renderRunnerSection(c *ui.Context, snap Snapshot, actions Actions) {
	tk := lobetheme.TokensOf(c)
	ui.Column(c).
		FillWidth().
		Grow(1).
		Center().
		Padding(16).
		Gap(12).
		Children(func() {
			theme.PanelHeader(c, "Collection Runner")

			colName := snap.CollectionName
			if colName == "" {
				colName = "No active collection"
			}
			ui.Text(c, colName).
				FontSize(12).
				TextColor(tk.TextSecondary)

			if actions.OnRunCollection != nil {
				widgets.Button(c, "Run Collection", widgets.ButtonProps{
					Size:    lobetheme.Small,
					Variant: action.Primary,
				}).OnClick(func() {
					if actions.OnRunCollection != nil {
						actions.OnRunCollection(snap.CollectionUID)
					}
				})
			} else {
				ui.Text(c, "No runner is connected yet.").
					FontSize(11).
					TextColor(tk.TextSecondary)
			}
		})
}

// CollectionHeader renders the environment selector and collection actions strip.
func (s *Screen) CollectionHeader(c *ui.Context) ui.Element {
	tk := lobetheme.TokensOf(c)
	snap := s.Snapshot
	actions := s.Actions

	colName := snap.CollectionName
	if colName == "" {
		colName = "fearless"
	}

	return ui.Row(c).
		FillWidth().
		Height(40).
		Background(tk.BgLayout).
		BorderWidth(0, 0, 1, 0).
		BorderColor(tk.Border).
		Padding(0, 12).
		AlignItems(ui.Center).
		Children(func() {
			// Collection Name
			ui.Text(c, colName).
				FontSize(theme.TitleSize).
				FontWeight(500).
				TextColor(tk.Text)

			ui.Box(c).Grow(1) // Spacer between collection name and env selector

			// Environment Selector
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				theme.PanelHeader(c, "Environment")

				// Environment select dropdown
				var envOptions []field.SelectOption
				for _, env := range snap.Environments {
					envOptions = append(envOptions, field.SelectOption{
						Label: env,
						Value: env,
					})
				}
				if len(envOptions) == 0 {
					envOptions = append(envOptions, field.SelectOption{
						Label: "No Environment",
						Value: "",
					})
				}

				activeEnv := snap.ActiveEnvironment
				ui.Box(c).Width(140).Children(func() {
					widgets.FieldSelect(c, &activeEnv, widgets.FieldSelectProps{
						Options: envOptions,
						Size:    lobetheme.Small,
					})
				})
				if activeEnv != snap.ActiveEnvironment && actions.OnSelectEnvironment != nil {
					actions.OnSelectEnvironment(activeEnv)
				}

				// Environment settings / manage button
				widgets.IconButton(c, widgets.IconButtonProps{
					Icon:    lucide.Settings(),
					Label:   "Environment settings",
					Tooltip: "Environment settings",
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(func() {
					if actions.OnManageEnvironments != nil {
						actions.OnManageEnvironments()
					}
				})

				// Run button
				widgets.Button(c, "Run", widgets.ButtonProps{
					Size:    lobetheme.Small,
					Variant: action.Text,
				}).OnClick(func() {
					if actions.OnRunCollection != nil {
						actions.OnRunCollection(snap.CollectionUID)
					}
				})
			})
		})
}

// View renders the navigation screen: the sidebar with its section switcher.
func (s *Screen) View(c *ui.Context) {
	ui.Row(c).FillHeight().Grow(1).Children(func() {
		s.Sidebar(c, lobe.SideNavState{})
	})
}

// Sidebar renders the navigation sidebar from an instantiated screen.
func Sidebar(c *ui.Context, s *Screen, nav lobe.SideNavState) ui.Element {
	if s == nil {
		s = &Screen{}
	}
	return s.Sidebar(c, nav)
}

// CollectionHeader renders the collection header strip from an instantiated screen.
func CollectionHeader(c *ui.Context, s *Screen) ui.Element {
	if s == nil {
		s = &Screen{}
	}
	return s.CollectionHeader(c)
}
