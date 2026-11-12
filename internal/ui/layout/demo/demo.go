// Package demo renders UI-LAYOUT's region structure with placeholder
// content, so the whole §8.2 tree can be seen and dragged before the
// screens exist: draggable sidebar, header strip, stacked/side-by-side
// split, right rail, status bar and console dock.
//
// It is scaffolding owned by UI-LAYOUT (internal/ui/layout/**). When
// UI-APP lands it replaces the call from main.go with the real root view;
// nothing here is imported by any other package.
package demo

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/layout"
	uitheme "fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// state is the demo's presentation state. A real app receives one from the
// presenter and persists it through layout.EncodeState; the demo keeps it
// in a package var so the toggles below have something to flip.
var state = layout.DefaultLayoutState()

var title = &layout.TitleBarView{
	AppName:           "fearless",
	CollectionName:    "demo",
	ShowMenu:          true,
	OnMenu:            func() {},
	Environments:      []string{"dev", "prod"},
	ActiveEnvironment: "dev",
}

var status = &layout.StatusBarView{
	RequestCount:  12,
	SelectedCount: 2,
	WorkspaceName: "local",
	OnWorkspace:   func() {},
	OnSettings:    func() {},
	RightStatus: []layout.StatusItem{
		{Label: "idle", Tooltip: "engine (demo)", OnClick: func() {}},
	},
}

var dock = &layout.ConsoleDockView{
	Tabs: []layout.ConsoleTab{
		{Label: "Logs", Content: func(c *ui.Context) {
			tk := uitheme.TokensOf(c)
			ui.Column(c).Fill().Padding(8, 12).Gap(4).Children(func() {
				ui.Text(c, "12:00:01  collection loaded (demo)").TextColor(tk.TextSecondary)
				ui.Text(c, "12:00:02  GET /health → 200 (demo)").TextColor(tk.TextSecondary)
			})
		}},
		{Label: "Network", Content: func(c *ui.Context) {
			tk := uitheme.TokensOf(c)
			ui.Column(c).Fill().Padding(8, 12).Gap(4).Children(func() {
				ui.Text(c, "GET /health   200   12 ms   1.2 kB (demo)").TextColor(tk.TextSecondary)
			})
		}},
	},
	ActiveTab: 0,
}

// root is the §8.2 tree with this demo's content in each region.
var root = &layout.RootView{
	State:  &state,
	Budget: layout.DefaultRegionBudget(),

	TitleBar: func(c *ui.Context) { title.View(c) },

	Sidebar: func(c *ui.Context, nav lobe.SideNavState) {
		tk := uitheme.TokensOf(c)
		ui.Column(c).Fill().Padding(8, 8).Gap(6).Children(func() {
			// Labels while expanded, single letters in the 64-DIP rail.
			for _, item := range []string{"Collections", "Environments", "History"} {
				label := item
				if !nav.Expanded {
					label = label[:1]
				}
				ui.Text(c, label).TextColor(tk.Text)
			}
		})
	},

	CollectionHeader: func(c *ui.Context) {
		tk := uitheme.TokensOf(c)
		ui.Row(c).Fill().AlignItems(ui.Center).Gap(8).Padding(0, 12).Background(tk.BgContainer).Children(func() {
			ui.Text(c, "demo collection").TextColor(tk.Text)
			widgets.Button(c, "env: dev", widgets.ButtonProps{Variant: action.Text, Size: lobetheme.Small}).
				OnClick(func() {}).Tooltip("Environment (demo)")
			ui.Box(c).Grow(1)
			widgets.Button(c, "Flip", widgets.ButtonProps{Variant: action.Text, Size: lobetheme.Small}).
				OnClick(flipOrientation).Tooltip("Stacked ⇄ side-by-side (⌘J in the real app)")
			widgets.Button(c, "Console", widgets.ButtonProps{Variant: action.Text, Size: lobetheme.Small}).
				OnClick(func() { state.ConsoleOpen = !state.ConsoleOpen }).Tooltip("Toggle the console dock")
			widgets.Button(c, "Rail", widgets.ButtonProps{Variant: action.Text, Size: lobetheme.Small}).
				OnClick(func() { state.RailExpanded = !state.RailExpanded }).Tooltip("Toggle the right rail")
		})
	},

	RequestPane: func(c *ui.Context) {
		tk := uitheme.TokensOf(c)
		ui.Column(c).Fill().Background(tk.BgContainer).Padding(12, 12).Children(func() {
			ui.Text(c, "request pane — drag the divider, the sidebar seam or the rail seam").TextColor(tk.Text)
		})
	},

	ResponsePane: func(c *ui.Context) {
		tk := uitheme.TokensOf(c)
		ui.Column(c).Fill().Background(tk.BgElevated).Padding(12, 12).Children(func() {
			ui.Text(c, "response pane").TextColor(tk.Text)
		})
	},

	RightRail: func(c *ui.Context) {
		tk := uitheme.TokensOf(c)
		ui.Column(c).Fill().Background(tk.BgLayout).Padding(12, 8).Children(func() {
			ui.Text(c, "right rail").TextColor(tk.Text).Bold()
			ui.Text(c, "DraggablePanel — drag its seam, collapse with its button").TextColor(tk.TextSecondary)
		})
	},

	StatusBar:   func(c *ui.Context) { status.View(c) },
	ConsoleDock: func(c *ui.Context) { dock.View(c) },
}

func init() {
	dock.OnTabChange = func(i int) { dock.ActiveTab = i }
	dock.OnClose = func() { state.ConsoleOpen = false }
}

func flipOrientation() {
	if state.RequestResponseOrientation == layout.OrientationStacked {
		state.RequestResponseOrientation = layout.OrientationSideBySide
	} else {
		state.RequestResponseOrientation = layout.OrientationStacked
	}
}

// View is the demo's root view: the full §8.2 tree, one frame of it.
// main.go wires this into the window until UI-APP supplies the real one.
func View(c *ui.Context) {
	root.View(c)
}
