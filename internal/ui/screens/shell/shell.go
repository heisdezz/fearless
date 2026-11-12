// Package shell renders the window chrome screen — title bar, status row and
// console dock — composed over layout.RootView with the regions the other
// screens own injected from outside.
//
// The view is pure (packages.md, UI-SCREEN-SHELL): everything it draws comes
// from a Snapshot, plain data copied from the presenters before the frame, and
// everything it does goes through Actions, commands the app wires and owns.
// A View closure mutates neither the snapshot nor any domain state. The only
// writes that happen while a frame builds are the toolkit's own interaction
// handlers (dragging a splitter writes the layout ratio) and layout's
// idempotent Normalize of presentation state — presentation, not domain.
package shell

import (
	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/overlay"

	"fearless/internal/ui/layout"
)

// Snapshot is the chrome's input: plain data, copied from the presenters
// before the frame so a concurrent presenter write cannot tear the frame.
// Nothing here is a function but the dock bodies, which live on Screen.
type Snapshot struct {
	AppName         string
	CollectionName  string
	WorkspaceName   string
	ShowMenu        bool
	MenuOpen        *bool
	MenuItems       []overlay.MenuItem
	RequestCount    int
	SelectedCount   int
	EnvironmentName string
	// Environments are the names the title strip's selector offers, in
	// order; EnvironmentName is the one selected. They live on the strip
	// because that is where the choice is made now.
	Environments []string
	// InFlight swaps the title strip's Run for Stop.
	InFlight bool
	// Maximized reports the window's state, so the title strip's window
	// control shows the restore glyph rather than the maximize one.
	Maximized bool
	// Status are the right-hand chips of the status row.
	Status []StatusEntry
	// ActiveConsoleTab selects the dock's body. The bodies themselves are
	// injected on Screen.ConsoleTabs because they are views, not data.
	ActiveConsoleTab int
}

// StatusEntry is one right-hand status chip. Index names the action to run
// when it is clicked; a negative Index renders it as a plain label with no
// click target, which is how Bruno shows an informational status.
type StatusEntry struct {
	Label   string
	Tooltip string
	Index   int
}

// Actions are the chrome's commands. Each is called by the app, which owns
// the resulting state change: the view reports intent and never applies it.
type Actions struct {
	OnMenu              func()
	OnMenuItem          func(key string)
	OnWorkspace         func()
	OnSettings          func()
	OnSelectEnvironment func(name string)
	OnManageEnvironment func()
	OnRun               func()
	OnStop              func()
	// OnMinimize, OnToggleMaximize and OnClose are the window's controls.
	// The window has no native title bar, so the app draws them.
	OnMinimize       func()
	OnToggleMaximize func()
	OnClose          func()
	OnConsoleTab     func(int)
	OnConsoleClose   func()
	// OnStatus runs a status chip's action, addressed by
	// Snapshot.Status[].Index.
	OnStatus func(index int)
}

// Screen is the chrome screen. State and the region callbacks are supplied
// by UI-APP when it assembles the window; the screen itself owns no state.
type Screen struct {
	// State is UI-LAYOUT's presentation state. A nil State renders one
	// frame of the defaults: nothing panics, and drags do not persist.
	State *layout.LayoutState
	// Budget is the region budget; the zero value means
	// layout.DefaultRegionBudget (RootView applies the default itself).
	Budget   layout.RegionBudget
	Snapshot Snapshot
	Actions  Actions
	// ConsoleTabs are the dock's bodies (Logs, Network, …), injected
	// because they belong to other screens rather than to the chrome.
	ConsoleTabs []layout.ConsoleTab
	// The regions the other screens own; nil renders an empty box.
	Sidebar func(*ui.Context, lobe.SideNavState)
	// CollectionHeaderRegion is the optional strip between the title bar and
	// the request pane. The main window mounts none: the title strip carries
	// the collection name, the environment selector and the run control, so a
	// second strip would answer the same question twice.
	CollectionHeaderRegion func(*ui.Context)
	RequestPane            func(*ui.Context)
	ResponsePane           func(*ui.Context)
	RightRail              func(*ui.Context)
}

// CollectionHeader renders the collection-header region when one is mounted.
//
// It is a method rather than the field itself so that a caller can ask for
// the region without knowing whether one exists: an unmounted region renders
// nothing instead of panicking.
func (s *Screen) CollectionHeader(c *ui.Context) {
	if s.CollectionHeaderRegion != nil {
		s.CollectionHeaderRegion(c)
	}
}

// View builds the frame. It reads the screen's fields into locals and hands
// them to a fresh layout.RootView, so rendering mutates nothing on the
// receiver: two renders of the same Screen produce the same frame.
func (s *Screen) View(c *ui.Context) {
	st := s.State
	if st == nil {
		def := layout.DefaultLayoutState()
		st = &def
	}
	// Copy the snapshot: the frame renders a value, not the live struct a
	// presenter might be writing to.
	snap := s.Snapshot
	actions := s.Actions
	tabs := s.ConsoleTabs

	root := layout.RootView{
		State:  st,
		Budget: s.Budget,
		TitleBar: func(c *ui.Context) {
			renderTitleBar(c, snap, actions)
		},
		Sidebar:          s.Sidebar,
		CollectionHeader: s.CollectionHeaderRegion,
		RequestPane:      s.RequestPane,
		ResponsePane:     s.ResponsePane,
		RightRail:        s.RightRail,
		StatusBar: func(c *ui.Context) {
			renderStatusBar(c, snap, actions)
		},
		ConsoleDock: func(c *ui.Context) {
			renderConsoleDock(c, tabs, snap, actions)
		},
	}
	root.View(c)
}

func renderTitleBar(c *ui.Context, snap Snapshot, a Actions) {
	bar := layout.TitleBarView{
		AppName:             snap.AppName,
		CollectionName:      snap.CollectionName,
		ShowMenu:            snap.ShowMenu,
		MenuOpen:            snap.MenuOpen,
		MenuItems:           snap.MenuItems,
		OnMenuItem:          a.OnMenuItem,
		OnMenu:              a.OnMenu,
		Environments:        snap.Environments,
		ActiveEnvironment:   snap.EnvironmentName,
		OnSelectEnvironment: a.OnSelectEnvironment,
		OnManageEnvironment: a.OnManageEnvironment,
		InFlight:            snap.InFlight,
		OnRun:               a.OnRun,
		OnStop:              a.OnStop,
		Maximized:           snap.Maximized,
		OnMinimize:          a.OnMinimize,
		OnToggleMaximize:    a.OnToggleMaximize,
		OnClose:             a.OnClose,
	}
	bar.View(c)
}

func renderStatusBar(c *ui.Context, snap Snapshot, a Actions) {
	items := make([]layout.StatusItem, 0, len(snap.Status))
	for _, entry := range snap.Status {
		e := entry // loop variable, captured by the click handler below
		item := layout.StatusItem{Label: e.Label, Tooltip: e.Tooltip}
		if e.Index >= 0 && a.OnStatus != nil {
			item.OnClick = func() { a.OnStatus(e.Index) }
		}
		items = append(items, item)
	}
	bar := layout.StatusBarView{
		RequestCount:  snap.RequestCount,
		SelectedCount: snap.SelectedCount,
		WorkspaceName: snap.WorkspaceName,
		OnWorkspace:   a.OnWorkspace,
		OnSettings:    a.OnSettings,
		RightStatus:   items,
	}
	bar.View(c)
}

func renderConsoleDock(c *ui.Context, tabs []layout.ConsoleTab, snap Snapshot, a Actions) {
	dock := layout.ConsoleDockView{
		Tabs:        tabs,
		ActiveTab:   snap.ActiveConsoleTab,
		OnTabChange: a.OnConsoleTab,
		OnClose:     a.OnConsoleClose,
	}
	dock.View(c)
}
