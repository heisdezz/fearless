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
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/presenters"
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
			s.renderRailSectionButton(c, "Collections", "📁", snap.ActiveSection == presenters.SidebarSectionCollections, func() {
				if actions.OnSelectSection != nil {
					actions.OnSelectSection(presenters.SidebarSectionCollections)
				}
			})
			ui.Box(c).Height(8)

			s.renderRailSectionButton(c, "History", "⏱", snap.ActiveSection == presenters.SidebarSectionHistory, func() {
				if actions.OnSelectSection != nil {
					actions.OnSelectSection(presenters.SidebarSectionHistory)
				}
			})
			ui.Box(c).Height(8)

			s.renderRailSectionButton(c, "Runner", "▶", snap.ActiveSection == presenters.SidebarSectionRunner, func() {
				if actions.OnSelectSection != nil {
					actions.OnSelectSection(presenters.SidebarSectionRunner)
				}
			})

			ui.Box(c).Grow(1) // Spacer pushing expand button to bottom

			// Expand toggle button
			ui.Row(c).
				Size(36, 36).
				Center().
				Radius(4).
				Background(tk.BgContainerSecondary).
				Tooltip("Expand Sidebar").
				OnClick(func() {
					if actions.OnToggleCollapse != nil {
						actions.OnToggleCollapse()
					}
				}).
				Children(func() {
					ui.Text(c, "»").FontSize(14).TextColor(tk.TextSecondary)
				})
		})
}

func (s *Screen) renderRailSectionButton(c *ui.Context, label, icon string, active bool, onClick func()) {
	tk := lobetheme.TokensOf(c)
	bg := tk.BgContainer
	fg := tk.TextSecondary
	if active {
		bg = tk.BgElevated
		fg = tk.Text
	}

	ui.Column(c).
		Size(44, 44).
		Center().
		Radius(6).
		Background(bg).
		Tooltip(label).
		OnClick(onClick).
		Children(func() {
			ui.Text(c, icon).FontSize(16).TextColor(fg)
			ui.Text(c, label).FontSize(9).TextColor(fg)
		})
}

func (s *Screen) renderExpanded(c *ui.Context, snap Snapshot, actions Actions) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Column(c).
		FillHeight().
		FillWidth().
		Background(tk.BgContainer).
		Children(func() {
			// 1. Top Section Switcher
			ui.Row(c).
				FillWidth().
				Height(36).
				Background(tk.BgLayout).
				Padding(4, 8).
				Gap(6).
				AlignItems(ui.Center).
				Children(func() {
					s.renderSectionTab(c, "Collections", presenters.SidebarSectionCollections, snap.ActiveSection, actions)
					s.renderSectionTab(c, "History", presenters.SidebarSectionHistory, snap.ActiveSection, actions)
					s.renderSectionTab(c, "Runner", presenters.SidebarSectionRunner, snap.ActiveSection, actions)

					ui.Box(c).Grow(1)

					// Collapse toggle button
					ui.Row(c).
						Padding(4, 6).
						Radius(4).
						Tooltip("Collapse Sidebar").
						OnClick(func() {
							if actions.OnToggleCollapse != nil {
								actions.OnToggleCollapse()
							}
						}).
						Children(func() {
							ui.Text(c, "«").FontSize(12).TextColor(tk.TextSecondary)
						})
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

func (s *Screen) renderSectionTab(c *ui.Context, label string, section, activeSection presenters.SidebarSection, actions Actions) {
	tk := lobetheme.TokensOf(c)
	isActive := section == activeSection

	bg := tk.BgLayout
	fg := tk.TextSecondary
	if isActive {
		bg = tk.BgElevated
		fg = tk.Text
	}

	ui.Row(c).
		Padding(4, 8).
		Radius(4).
		Background(bg).
		OnClick(func() {
			if actions.OnSelectSection != nil {
				actions.OnSelectSection(section)
			}
		}).
		Children(func() {
			t := ui.Text(c, label).FontSize(12).TextColor(fg)
			if isActive {
				t.Bold()
			}
		})
}

func (s *Screen) renderCollectionsSection(c *ui.Context, snap Snapshot, actions Actions) {
	tk := lobetheme.TokensOf(c)

	// Filter & Action Toolbar
	ui.Column(c).FillWidth().Padding(8, 8, 4, 8).Gap(6).Children(func() {
		// Filter SearchBar
		q := snap.FilterQuery
		field.SearchBar(c, &q, field.SearchBarProps{
			Placeholder: "Filter requests...",
			Size:        lobetheme.Small,
			OnSearch: func(query string) {
				if actions.OnFilterChanged != nil {
					actions.OnFilterChanged(query)
				}
			},
		})
		if q != snap.FilterQuery && actions.OnFilterChanged != nil {
			actions.OnFilterChanged(q)
		}

		// Action buttons row: + Request, + Folder, Open
		ui.Row(c).FillWidth().Gap(4).AlignItems(ui.Center).Children(func() {
			widgets.Button(c, "+ Request", widgets.ButtonProps{
				Size:    lobetheme.Small,
				Variant: action.Default,
			}).OnClick(func() {
				if actions.OnCreateRequest != nil {
					actions.OnCreateRequest(snap.SelectedUID)
				}
			})

			widgets.Button(c, "+ Folder", widgets.ButtonProps{
				Size:    lobetheme.Small,
				Variant: action.Default,
			}).OnClick(func() {
				if actions.OnCreateFolder != nil {
					actions.OnCreateFolder(snap.SelectedUID)
				}
			})

			ui.Box(c).Grow(1)

			widgets.Button(c, "Open", widgets.ButtonProps{
				Size:    lobetheme.Small,
				Variant: action.Default,
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
			Gap(6).
			Children(func() {
				ui.Text(c, "No requests found").
					FontSize(13).
					Bold().
					TextColor(tk.TextSecondary)
				ui.Text(c, "Open a collection or create a request").
					FontSize(11).
					TextColor(tk.TextTertiary)
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
			ui.Text(c, "Request History").
				FontSize(13).
				Bold().
				TextColor(tk.TextSecondary)
			ui.Text(c, "Past request executions will appear here").
				FontSize(11).
				TextColor(tk.TextTertiary)
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
			ui.Text(c, "Collection Runner").
				FontSize(14).
				Bold().
				TextColor(tk.Text)

			colName := snap.CollectionName
			if colName == "" {
				colName = "No Collection Selected"
			}
			ui.Text(c, "Target: "+colName).
				FontSize(12).
				TextColor(tk.TextSecondary)

			widgets.Button(c, "Run Collection", widgets.ButtonProps{
				Size:    lobetheme.Small,
				Variant: action.Primary,
			}).OnClick(func() {
				if actions.OnRunCollection != nil {
					actions.OnRunCollection(snap.CollectionUID)
				}
			})
		})
}

// CollectionHeader renders the environment selector and collection actions strip.
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
		Border(1, tk.BorderSecondary).
		Padding(4, 12).
		AlignItems(ui.Center).
		Children(func() {
			// Collection Name
			ui.Text(c, colName).
				FontSize(13).
				Bold().
				TextColor(tk.Text)

			ui.Box(c).Grow(1) // Spacer between collection name and env selector

			// Environment Selector
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, "Env:").FontSize(11).TextColor(tk.TextSecondary)

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
				widgets.Button(c, "⚙ Env", widgets.ButtonProps{
					Size:    lobetheme.Small,
					Variant: action.Default,
				}).OnClick(func() {
					if actions.OnManageEnvironments != nil {
						actions.OnManageEnvironments()
					}
				})

				// Run button
				widgets.Button(c, "Run", widgets.ButtonProps{
					Size:    lobetheme.Small,
					Variant: action.Primary,
				}).OnClick(func() {
					if actions.OnRunCollection != nil {
						actions.OnRunCollection(snap.CollectionUID)
					}
				})
			})
		})
}

// View renders the complete composite navigation screen (CollectionHeader + Sidebar).
func (s *Screen) View(c *ui.Context) {
	ui.Column(c).Fill().Children(func() {
		s.CollectionHeader(c)
		ui.Row(c).FillHeight().Grow(1).Children(func() {
			s.Sidebar(c, lobe.SideNavState{})
		})
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
