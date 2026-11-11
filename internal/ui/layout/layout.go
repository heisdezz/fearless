package layout

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

// RegionBudget holds the pixel budgets for each UI region
type RegionBudget struct {
	TitleBar       float32 // ~28-32 px
	SidebarMin     float32 // 220 px
	SidebarMax     float32 // 600 px
	SidebarCollapsed float32 // 64 px (icon rail)
	CollectionHeader float32 // ~40 px
	StatusBar      float32 // ~24 px
	ConsoleDock    float32 // ~300 px when open
	MinWindowWidth  float32 // 900 px
	MinWindowHeight float32 // 600 px
}

func DefaultRegionBudget() RegionBudget {
	return RegionBudget{
		TitleBar:         32,
		SidebarMin:       220,
		SidebarMax:       600,
		SidebarCollapsed: 64,
		CollectionHeader: 40,
		StatusBar:        24,
		ConsoleDock:      300,
		MinWindowWidth:   900,
		MinWindowHeight:  600,
	}
}

// Orientation represents the request/response pane orientation
type Orientation int

const (
	OrientationStacked Orientation = iota
	OrientationSideBySide
)

// LayoutState holds presentation state for the main window
type LayoutState struct {
	SidebarWidth         float32
	SidebarExpanded      bool
	RequestResponseOrientation Orientation
	ConsoleOpen          bool
	SplitRatio           float32 // 0.0-1.0 for request/response split
	WindowWidth          float32
	WindowHeight         float32
}

func DefaultLayoutState() LayoutState {
	return LayoutState{
		SidebarWidth:              280,
		SidebarExpanded:           true,
		RequestResponseOrientation: OrientationStacked,
		ConsoleOpen:               false,
		SplitRatio:                0.55,
		WindowWidth:               1200,
		WindowHeight:              800,
	}
}

// RootView is the main application layout root
type RootView struct {
	State        *LayoutState
	Budget       RegionBudget
	TitleBar     func(*ui.Context)
	Sidebar      func(*ui.Context, layout.SideNavState)
	RequestPane  func(*ui.Context)
	ResponsePane func(*ui.Context)
	StatusBar    func(*ui.Context)
	ConsoleDock  func(*ui.Context)
}

func (r *RootView) View(c *ui.Context) {
	theme.Apply(c)
	tk := theme.TokensOf(c)

	ui.Column(c).Children(func() {
		// Title bar
		if r.TitleBar != nil {
			r.TitleBar(c)
		}

		// Main body area
		ui.Row(c).Children(func() {
			// Sidebar with draggable resize
			if r.Sidebar != nil {
				props := layout.DraggableSideNavProps{
					MinWidth:   r.Budget.SidebarMin,
					MaxWidth:   r.Budget.SidebarMax,
					Background: tk.BgContainer,
				}
				layout.DraggableSideNav(c, &r.State.SidebarWidth, &r.State.SidebarExpanded, props, func(state layout.SideNavState) {
					r.Sidebar(c, state)
				}).FillHeight()
			}

			// Main content area
			ui.Column(c).Children(func() {
				// Collection header
				ui.Box(c).Height(r.Budget.CollectionHeader).FillWidth().Children(func() {
					// Environment selector, runner buttons go here
					// Implemented by presenter
				})

				// Request/Response split
				if r.State.RequestResponseOrientation == OrientationStacked {
					ui.SplitVertical(c, &r.State.SplitRatio, func() {
						if r.RequestPane != nil {
							r.RequestPane(c)
						}
					}, func() {
						if r.ResponsePane != nil {
							r.ResponsePane(c)
						}
					}).Fill()
				} else {
					ui.Split(c, &r.State.SplitRatio, func() {
						if r.RequestPane != nil {
							r.RequestPane(c)
						}
					}, func() {
						if r.ResponsePane != nil {
							r.ResponsePane(c)
						}
					}).Fill()
				}
			}).Fill().Grow(1)
		}).FillHeight().Grow(1)

		// Bottom chrome (status bar + console dock)
		ui.Column(c).Children(func() {
			if r.StatusBar != nil {
				ui.Box(c).Height(r.Budget.StatusBar).FillWidth().Children(func() {
					r.StatusBar(c)
				})
			}

			if r.State.ConsoleOpen && r.ConsoleDock != nil {
				ui.Box(c).Height(r.Budget.ConsoleDock).FillWidth().Children(func() {
					r.ConsoleDock(c)
				})
			}
		})

		// Toast host (mounted once)
		feedback.ToastHost(c, feedback.ToastHostProps{})
	})
}

// TitleBarView creates the custom title bar row
type TitleBarView struct {
	AppName        string
	CollectionName string
	WorkspaceName  string
	ShowMenu       bool
	OnMenu         func()
	OnWorkspace    func()
	OnSettings     func()
}

func (t *TitleBarView) View(c *ui.Context) {
	tk := theme.TokensOf(c)

	// Title bar content goes in the window's title bar area
	ui.Row(c).Gap(8).Padding(8, 12).Background(tk.BgContainer).Children(func() {
		// App menu (Windows/Linux)
		if t.ShowMenu && t.OnMenu != nil {
			action.Button(c, "☰", action.ButtonProps{
				Variant: action.Text,
				Size:    theme.Small,
			}).OnClick(t.OnMenu).Tooltip("Menu")
		}

		// App name / collection
		ui.Text(c, t.AppName).TextColor(tk.Text).Bold()
		if t.CollectionName != "" {
			ui.Text(c, " — ").TextColor(tk.TextSecondary)
			ui.Text(c, t.CollectionName).TextColor(tk.Text)
		}

		ui.Box(c).Grow(1) // spacer

		// Workspace selector
		if t.WorkspaceName != "" && t.OnWorkspace != nil {
			action.Button(c, t.WorkspaceName, action.ButtonProps{
				Variant: action.Text,
				Size:    theme.Small,
			}).OnClick(t.OnWorkspace).Tooltip("Switch workspace")
		}

		// Settings
		if t.OnSettings != nil {
			action.Button(c, "⚙", action.ButtonProps{
				Variant: action.Text,
				Size:    theme.Small,
			}).OnClick(t.OnSettings).Tooltip("Settings")
		}
	})
}

// StatusBarView creates the bottom status bar
type StatusBarView struct {
	Budget          RegionBudget
	RequestCount    int
	SelectedCount   int
	EnvironmentName string
	OnEnvironment   func()
	RightStatus     []StatusItem
}

type StatusItem struct {
	Label   string
	Color   ui.Color
	Tooltip string
	OnClick func()
}

func (s *StatusBarView) View(c *ui.Context) {
	tk := theme.TokensOf(c)

	ui.Row(c).Gap(12).Padding(0, 12).Height(s.Budget.StatusBar).Background(tk.BgContainer).Children(func() {
		// Left: request counts
		if s.RequestCount > 0 {
			ui.Text(c, formatCount(s.RequestCount, "request")).TextColor(tk.TextSecondary).FontSize(tk.FontSizeSM)
			if s.SelectedCount > 0 {
				ui.Text(c, " · ").TextColor(tk.TextSecondary)
				ui.Text(c, formatCount(s.SelectedCount, "selected")).TextColor(tk.TextSecondary)
			}
		}

		// Environment selector
		if s.EnvironmentName != "" && s.OnEnvironment != nil {
			action.Button(c, s.EnvironmentName, action.ButtonProps{
				Variant: action.Text,
				Size:    theme.Small,
			}).OnClick(s.OnEnvironment).Tooltip("Change environment")
		}

		ui.Box(c).Grow(1) // spacer

		// Right status items
		for _, item := range s.RightStatus {
			if item.OnClick != nil {
				action.Button(c, item.Label, action.ButtonProps{
					Variant: action.Text,
					Size:    theme.Small,
				}).OnClick(item.OnClick).Tooltip(item.Tooltip)
			} else {
				ui.Text(c, item.Label).TextColor(item.Color).FontSize(tk.FontSizeSM)
			}
		}
	})
}

func formatCount(n int, singular string) string {
	if n == 1 {
		return "1 " + singular
	}
	return "0 " + singular + "s"
}

// ConsoleDockView creates the bottom console dock
type ConsoleDockView struct {
	Tabs        []ConsoleTab
	ActiveTab   int
	OnTabChange func(int)
	OnClose     func()
}

type ConsoleTab struct {
	Label   string
	Content func(*ui.Context)
	Icon    string
	Badge   string
}

func (c *ConsoleDockView) View(ctx *ui.Context) {
	tk := theme.TokensOf(ctx)

	ui.Box(ctx).Height(300).FillWidth().Background(tk.BgBase).Children(func() {
		// Tab bar
		ui.Row(ctx).Gap(4).Padding(8, 8).Children(func() {
			for i, tab := range c.Tabs {
				active := i == c.ActiveTab
				btn := action.Button(ctx, tab.Label, action.ButtonProps{
					Variant: action.Text,
					Size:    theme.Small,
				}).OnClick(func() {
					if c.OnTabChange != nil {
						c.OnTabChange(i)
					}
				})
				if active {
					btn.Background(tk.Primary).TextColor(tk.PrimaryForeground)
				}
			}
			ui.Box(ctx).Grow(1)
			if c.OnClose != nil {
				action.Button(ctx, "✕", action.ButtonProps{
					Variant: action.Text,
					Size:    theme.Small,
				}).OnClick(c.OnClose).Tooltip("Close console")
			}
		})

		// Active tab content
		if c.ActiveTab >= 0 && c.ActiveTab < len(c.Tabs) {
			c.Tabs[c.ActiveTab].Content(ctx)
		}
	})
}