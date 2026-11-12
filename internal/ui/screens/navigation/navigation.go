// Package navigation renders the navigation sidebar and collection explorer screen
// as described in docs/ui/application-layout.md and docs/ui/components.md.
//
// The screen is a pure function of Snapshot (packages.md, UI-SCREEN-NAV):
// everything it draws is plain data, and all interactions trigger Actions
// callbacks owned and applied by the application. Rendering never mutates
// presenters or domain models.
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

// Snapshot is the plain-data view-model representing navigation and collection explorer state.
// No fields are pointers to live mutable presenter state.
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

// Actions specifies the user command handlers for navigation and collection actions.
// View functions invoke these callbacks to report user intent without performing state mutations.
type Actions struct {
	OnSelectSection      func(section presenters.SidebarSection)
	OnToggleCollapse     func()
	OnFilterChanged      func(query string)
	OnSelectNode         func(uid string)
	OnToggleNode         func(uid string)
	OnOpenCollection     func()
	OnCreateCollection   func()
	OnCreateRequest      func(parentUID string)
	OnCreateFolder       func(parentUID string)
	OnDeleteItem         func(uid string)
	OnSelectEnvironment  func(envName string)
	OnManageEnvironments func()
	OnRunCollection      func(colUID string)
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
	Snapshot Snapshot
	Actions  Actions
}

// NewScreen creates an initialized Screen from a sidebar presenter and command actions.
func NewScreen(sidebar *presenters.SidebarPresenter, actions Actions) *Screen {
	return &Screen{
		Snapshot: TakeSnapshot(sidebar),
		Actions:  actions,
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

			ui.Box(c).Grow(1) // Spacer pushing expand button to bottom

			// Expand toggle button
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

// renderRailSectionButton draws one rail tile.
//
// The rail is icon-only: at 64 DIP wide a small-caps word does not fit, so
// the tile carries a drawn icon and its tooltip, and the selected section is
// marked by the accent ink on that icon — the one accent in the window, spent
// on the selected state.
func (s *Screen) renderRailSectionButton(c *ui.Context, label string, icon *ui.SVG, active bool, onClick func()) {
	tk := lobetheme.TokensOf(c)
	bg := tk.BgContainer
	fg := tk.TextSecondary
	if active {
		bg = tk.BgContainerSecondary
		fg = tk.Primary
	}

	ui.Column(c).
		Size(44, 44).
		Center().
		Radius(6).
		Background(bg).
		Tooltip(label).
		Label(label).
		OnClick(onClick).
		Children(func() {
			ui.Icon(c, icon).Size(18, 18).TextColor(fg)
		})
}

func (s *Screen) renderExpanded(c *ui.Context, snap Snapshot, actions Actions) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Column(c).
		FillHeight().
		FillWidth().
		Background(tk.BgContainer).
		Children(func() {
			// 1. Top Section Switcher: the world's strip, shared with the
			// request column and the response pane so the three speak with
			// one voice. The collapse control rides at its right edge.
			theme.TabStrip(c, theme.TabStripProps{
				Items: []theme.TabItem{
					{Label: "Collections"},
					{Label: "History"},
					{Label: "Runner"},
				},
				Active: sectionIndex(snap.ActiveSection),
				OnSelect: func(index int) {
					if actions.OnSelectSection != nil {
						actions.OnSelectSection(sections[index])
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

func (s *Screen) renderCollectionsSection(c *ui.Context, snap Snapshot, actions Actions) {
	tk := lobetheme.TokensOf(c)

	// Filter & Action Toolbar
	ui.Column(c).FillWidth().Padding(8, 8, 4, 8).Gap(6).Children(func() {
		// Filter field: the app's own field surface, not Lobe's SearchBar,
		// which paints MyGo's near-white input surface in the dark
		// appearance (the white-input defect). The filter still applies as
		// the query changes: the field writes into the local copy, and the
		// copy that differs from the snapshot is the filter to apply.
		q := snap.FilterQuery
		widgets.FieldInput(c, &q, widgets.FieldInputProps{
			Placeholder: "Filter requests…",
			Size:        lobetheme.Small,
		})
		if q != snap.FilterQuery && actions.OnFilterChanged != nil {
			actions.OnFilterChanged(q)
		}

		// Action buttons row: + Request, + Folder, Open.
		//
		// They are quiet on purpose. The one accent in the window belongs to
		// the primary action — Send — and the selected state; a toolbar of
		// three identical filled buttons spends the hierarchy on nothing.
		// The toolbar's actions are drawn, not spelled: three words do not
		// fit across a 248 DIP sidebar, and a clipped label ("Oper") is
		// worse than a word too many. Each control keeps its name in its
		// tooltip, so the action is still named for anyone who asks.
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
				Icon:    lucide.Library(),
				Label:   "+ Col",
				Tooltip: "New collection",
				Size:    lobetheme.Small,
				Variant: action.Text,
			}).OnClick(func() {
				if actions.OnCreateCollection != nil {
					actions.OnCreateCollection()
				}
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

	// Tree list area
	if len(snap.VisibleNodes) == 0 {
		ui.Column(c).
			Fill().
			Grow(1).
			Center().
			Padding(16).
			Gap(8).
			Children(func() {
				theme.PanelHeader(c, "No requests found")
				ui.Text(c, "Open an existing collection or create a new one").
					FontSize(11).
					TextColor(tk.TextSecondary)
				ui.Row(c).Gap(6).Children(func() {
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

	// Render the list of visible nodes
	ui.Scroll(c).
		FillWidth().
		Grow(1).
		Padding(4, 6).
		Children(func() {
			for _, node := range snap.VisibleNodes {
				if node == nil {
					continue
				}
				n := node
				widgets.CollectionTreeRow(c, n, widgets.TreeRowProps{
					OnToggle: func(uid string) {
						if actions.OnToggleNode != nil {
							actions.OnToggleNode(uid)
						}
					},
					OnSelect: func(uid string) {
						if actions.OnSelectNode != nil {
							actions.OnSelectNode(uid)
						}
					},
				})
			}
		})
}

func (s *Screen) renderHistorySection(c *ui.Context, snap Snapshot, actions Actions) {
	tk := lobetheme.TokensOf(c)
	ui.Column(c).
		Fill().
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
		Fill().
		Grow(1).
		Padding(16).
		Gap(12).
		Children(func() {
			theme.PanelHeader(c, "Collection Runner")

			colName := snap.CollectionName
			if colName == "" {
				colName = "No Collection Selected"
			}
			ui.Text(c, "Target: "+colName).
				FontSize(12).
				TextColor(tk.TextSecondary)

			// The run button appears only when a runner is wired: a button
			// with no backend behind it is a dead end, and the section
			// says what is missing instead.
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
//
// The main window does not mount it: the title strip carries the collection
// name, the environment selector and the run control now (the direction
// contract's first viewport), so a second environment selector under the title
// strip would answer the same question twice. It stays available for a window
// that embeds the sidebar on its own.
func (s *Screen) CollectionHeader(c *ui.Context) ui.Element {
	tk := lobetheme.TokensOf(c)
	snap := s.Snapshot
	actions := s.Actions

	colName := snap.CollectionName
	if colName == "" {
		colName = "Fearless Collection"
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
				envOptions = append(envOptions, field.SelectOption{
					Value: "",
					Label: "No Environment",
				})
				for _, env := range snap.Environments {
					envOptions = append(envOptions, field.SelectOption{
						Value: env,
						Label: env,
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
//
// The collection-header strip is not part of it — the title strip owns the
// collection name, the environment and the run control in the main window.
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
