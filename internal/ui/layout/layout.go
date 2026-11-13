// Package layout composes the application's regions — title bar, sidebar,
// request/response split, status bar and console dock — as described in
// docs/ui/application-layout.md §8.
//
// It owns presentation state only (§8.3): split ratios, sidebar width, pane
// orientation and console visibility are plain data, serialised by
// EncodeState and repaired on load by DecodeState. Nothing in the package
// reads domain state — TestImportsAreDomainFree enforces that over the
// package's own syntax. Region sizes come from RegionBudget (§8.1); the
// contents of every region arrive as callbacks from the screens, so no
// screen is composed or restyled here.
//
// All sizes are DIPs. MyGo converts DIPs to device pixels at the window's
// scale, so the budget is identical at 100 % and 200 % DPI — asserted by
// TestRegionBudgetAtDPIScales.
package layout

import (
	"encoding/json"
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	uitheme "fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// RegionBudget is the §8.1 region budget in DIPs: the fixed chrome heights
// and the bounds every resizable region must respect. Zero fields fall back
// to DefaultRegionBudget through withDefaults, so a partially specified
// budget still produces the full structure.
type RegionBudget struct {
	TitleBar         float32 // top chrome; max(window controls, this) when the frame is hidden
	SidebarMin       float32 // least width a drag leaves while expanded
	SidebarMax       float32 // most width a drag gives
	SidebarCollapsed float32 // rail width while collapsed (DraggableSideNav MinWidth)
	CollectionHeader float32 // environment selector strip
	StatusBar        float32 // bottom chrome
	ConsoleDock      float32 // fixed height while open; opening it shrinks the content
	RailMin          float32 // right rail (DraggablePanel) smallest size
	RailMax          float32 // right rail largest size
	MinWindowWidth   float32 // window floor the app enforces (UI-APP/PLATFORM)
	MinWindowHeight  float32
}

// DefaultRegionBudget returns the §8.1 budget at 100 % DPI. MyGo scales
// DIPs itself, so these numbers hold at every DPI; they are never
// multiplied by a scale factor here.
//
// The chrome heights are the direction contract's: a 30 DIP title strip
// carrying the collection name and the environment selector, and a 24 DIP
// status strip closing the window (the collection-header strip of §8.1 is
// still budgeted for a screen that asks for it, but the main window no
// longer mounts one — see RootView.View).
func DefaultRegionBudget() RegionBudget {
	return RegionBudget{
		TitleBar:         30,
		SidebarMin:       220,
		SidebarMax:       600,
		SidebarCollapsed: 64,
		CollectionHeader: 40,
		StatusBar:        24,
		ConsoleDock:      300,
		RailMin:          240,
		RailMax:          560,
		MinWindowWidth:   900,
		MinWindowHeight:  600,
	}
}

// withDefaults fills zero or negative fields from DefaultRegionBudget.
func (b RegionBudget) withDefaults() RegionBudget {
	d := DefaultRegionBudget()
	if b.TitleBar <= 0 {
		b.TitleBar = d.TitleBar
	}
	if b.SidebarMin <= 0 {
		b.SidebarMin = d.SidebarMin
	}
	if b.SidebarMax <= 0 {
		b.SidebarMax = d.SidebarMax
	}
	if b.SidebarCollapsed <= 0 {
		b.SidebarCollapsed = d.SidebarCollapsed
	}
	if b.CollectionHeader <= 0 {
		b.CollectionHeader = d.CollectionHeader
	}
	if b.StatusBar <= 0 {
		b.StatusBar = d.StatusBar
	}
	if b.ConsoleDock <= 0 {
		b.ConsoleDock = d.ConsoleDock
	}
	if b.RailMin <= 0 {
		b.RailMin = d.RailMin
	}
	if b.RailMax <= 0 {
		b.RailMax = d.RailMax
	}
	if b.MinWindowWidth <= 0 {
		b.MinWindowWidth = d.MinWindowWidth
	}
	if b.MinWindowHeight <= 0 {
		b.MinWindowHeight = d.MinWindowHeight
	}
	return b
}

// Orientation is how the request and response panes share the middle
// region — Bruno's requestPaneOrientation, flipped by the changeLayout
// hotkey (⌘/Ctrl+J).
type Orientation int

const (
	// OrientationStacked puts the request pane above the response pane.
	OrientationStacked Orientation = iota
	// OrientationSideBySide puts the response pane to the right.
	OrientationSideBySide
)

// Split-percent bounds: state persisted by an older build or edited by hand
// must not be able to erase a pane.
const (
	minSplitPercent = 5
	maxSplitPercent = 95
)

// LayoutState is the persisted presentation state of the main window
// (§8.3). It is deliberately plain data: screens mutate it through the
// presenter, which saves it to the settings file, and the core never reads
// it. Zero fields are repaired and out-of-range values clamped by
// Normalize, so loading is always DecodeState → usable state.
//
// Window bounds are not here: they belong to PLATFORM's windowstate.
type LayoutState struct {
	SidebarWidth    float32 `json:"sidebar_width"` // DIPs, the expanded width
	SidebarExpanded bool    `json:"sidebar_expanded"`
	// RequestResponseOrientation selects OrientationStacked or
	// OrientationSideBySide.
	RequestResponseOrientation Orientation `json:"orientation"`
	// SplitPercent is the request pane's share of the request/response
	// region, 0–100. A percent rather than DIPs so the persisted ratio
	// survives window resizes (SplitPercent/SplitVerticalPercent grow and
	// shrink both panes in proportion).
	SplitPercent float32 `json:"split_percent"`
	ConsoleOpen  bool    `json:"console_open"`
	// RailWidth and RailExpanded drive the optional right rail
	// (DraggablePanel), Bruno's pop-out AI sidebar.
	RailWidth    float32 `json:"rail_width"`
	RailExpanded bool    `json:"rail_expanded"`
}

// DefaultLayoutState returns the default §8 structure: expanded 248 DIP
// sidebar (the direction contract's first viewport), stacked panes at a
// 55 % request share, console closed, rail hidden.
func DefaultLayoutState() LayoutState {
	return LayoutState{
		SidebarWidth:               248,
		SidebarExpanded:            true,
		RequestResponseOrientation: OrientationStacked,
		SplitPercent:               55,
		ConsoleOpen:                false,
		RailWidth:                  320,
		RailExpanded:               false,
	}
}

// Normalize repairs zero or invalid numeric fields with the defaults and
// clamps the rest into b, so a truncated or hand-edited settings file
// cannot produce an impossible layout. Bools are taken as written — false
// is a legitimate value for a collapsed sidebar — so callers that need
// default bools should start from DefaultLayoutState or use DecodeState,
// which overlays the file on the defaults.
func (s *LayoutState) Normalize(b RegionBudget) {
	b = b.withDefaults()
	d := DefaultLayoutState()

	if s.SidebarWidth <= 0 {
		s.SidebarWidth = d.SidebarWidth
	}
	s.SidebarWidth = clamp(s.SidebarWidth, b.SidebarMin, b.SidebarMax)

	// The sidebar is resizable but never collapsible: a state file written by
	// a build that supported the rail must not reopen as one.
	s.SidebarExpanded = true

	if s.SplitPercent <= 0 {
		s.SplitPercent = d.SplitPercent
	}
	s.SplitPercent = clamp(s.SplitPercent, minSplitPercent, maxSplitPercent)

	if s.RailWidth <= 0 {
		s.RailWidth = d.RailWidth
	}
	s.RailWidth = clamp(s.RailWidth, b.RailMin, b.RailMax)

	if s.RequestResponseOrientation != OrientationStacked &&
		s.RequestResponseOrientation != OrientationSideBySide {
		s.RequestResponseOrientation = OrientationStacked
	}
}

// Degrade applies the §8.4 degraded modes for a window of the given width:
// below MinWindowWidth the panes stack, so the editor keeps a usable width.
// The sidebar no longer collapses to a rail — it just narrows to its minimum
// width instead — so only the orientation is downgraded here. It is
// idempotent; call it whenever the window resizes.
func (s *LayoutState) Degrade(width float32, b RegionBudget) {
	b = b.withDefaults()
	if width > 0 && width < b.MinWindowWidth {
		s.SidebarWidth = b.SidebarMin
		s.RequestResponseOrientation = OrientationStacked
	}
}

// EncodeState returns the JSON the settings file holds for s. The caller
// persists the bytes — PERSIST owns storage; this package never touches
// disk.
func EncodeState(s LayoutState) ([]byte, error) {
	return json.Marshal(s)
}

// DecodeState parses what EncodeState wrote, overlays it on the defaults —
// so keys an older build never wrote keep their default value — and
// normalises against b, so values from a hand edit cannot break the layout.
// Corrupt input returns an error; it never panics.
func DecodeState(data []byte, b RegionBudget) (LayoutState, error) {
	s := DefaultLayoutState()
	if err := json.Unmarshal(data, &s); err != nil {
		return LayoutState{}, fmt.Errorf("layout: decode state: %w", err)
	}
	s.Normalize(b)
	return s, nil
}

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// RootView is the §8.2 window tree: chrome, body, bottom regions and the
// toast host, with every content region supplied by the caller. View is a
// pure function of State apart from the split, sidebar and rail handles
// MyGo writes through their pointers; it holds no domain data.
//
// State must not be nil: the app layer owns one LayoutState and hands it
// here so the drag handles, the presenter and the settings file all agree.
type RootView struct {
	State  *LayoutState
	Budget RegionBudget

	TitleBar func(*ui.Context)
	// Sidebar builds the side-nav content; the SideNavState tells it how
	// the nav currently shows (expanded, collapse progress, rail width).
	Sidebar          func(*ui.Context, lobe.SideNavState)
	CollectionHeader func(*ui.Context) // environment selector + runner strip
	RequestPane      func(*ui.Context)
	ResponsePane     func(*ui.Context)
	// RightRail is the optional pop-out rail (DraggablePanel); nil keeps
	// it out of the tree entirely.
	RightRail   func(*ui.Context)
	StatusBar   func(*ui.Context)
	ConsoleDock func(*ui.Context)
}

// View renders the window tree for one frame.
func (r *RootView) View(c *ui.Context) {
	// theme.Apply once at the top of the root view, before any Lobe
	// component renders (§8.2 layer note).
	uitheme.Apply(c)
	tk := uitheme.TokensOf(c)

	b := r.Budget.withDefaults()

	// Repair the state on the way in. DecodeState normalises what it reads
	// from disk, but a state built in memory — an older build's file, a
	// hand edit, a test — can still carry values outside the §8.1 budget,
	// and rendering must never be driven out of it. Normalize is idempotent,
	// clamps numbers into the budget and takes bools as written (review
	// finding on UI-LAYOUT, 2026-11-11T02:18Z).
	r.State.Normalize(b)

	ui.Column(c).Fill().Children(func() {
		// Title bar: the §8.1 chrome row, padded clear of the window
		// controls of a hidden frame and draggable as a whole; children
		// take the pointer before the drag does.
		bar := c.TitleBar()
		ui.Box(c).
			Height(max(bar.Height, b.TitleBar)).
			FillWidth().
			Padding(0, bar.Right, 0, bar.Left).
			DragWindow().
			Children(func() {
				if r.TitleBar != nil {
					r.TitleBar(c)
				}
			})

		// Body: DraggableSideNav | main column, with the optional right
		// rail as the last flow child so expanding it pushes the panes
		// rather than covering them.
		ui.Row(c).FillHeight().Grow(1).Children(func() {
			if r.Sidebar != nil {
				// The sidebar only resizes: NoExpand drops the collapse tab,
				// and NoResize is left off so the seam stays draggable.
				// MinWidth is where a drag stops, so the nav can never collapse
				// into a rail.
				props := lobe.DraggableSideNavProps{
					MinWidth:   b.SidebarMin,
					MaxWidth:   b.SidebarMax,
					NoExpand:   true,
					Background: tk.BgContainer,
				}
				lobe.DraggableSideNav(c, &r.State.SidebarWidth, &r.State.SidebarExpanded, props, func(nav lobe.SideNavState) {
					r.Sidebar(c, nav)
				}).FillHeight()
			}

			ui.Column(c).Fill().Grow(1).Children(func() {
				// Collection header strip: fixed budget, content optional —
				// and no box at all when nothing is mounted. The main window
				// leaves this callback nil now (the environment selector and
				// run moved up into the title strip), and an empty 40 DIP
				// band above the request pane would be a hole in the first
				// viewport rather than chrome.
				if r.CollectionHeader != nil {
					ui.Box(c).Height(b.CollectionHeader).FillWidth().Children(func() {
						r.CollectionHeader(c)
					})
				}

				// Request/response split. Percent-based so the persisted
				// ratio survives resizes; the orientation flag is the
				// only thing changeLayout flips, as in Bruno.
				req := func() {
					if r.RequestPane != nil {
						r.RequestPane(c)
					}
				}
				resp := func() {
					if r.ResponsePane != nil {
						r.ResponsePane(c)
					}
				}
				if r.State.RequestResponseOrientation == OrientationSideBySide {
					ui.SplitPercent(c, &r.State.SplitPercent, req, resp).Fill()
				} else {
					ui.SplitVerticalPercent(c, &r.State.SplitPercent, req, resp).Fill()
				}
			})

			// Optional right rail: DraggablePanel resizes from its inner
			// seam, collapses to zero width and remembers RailWidth.
			if r.RightRail != nil {
				lobe.DraggablePanel(c, &r.State.RailWidth, &r.State.RailExpanded,
					lobe.DraggablePanelProps{
						Placement: lobe.Right,
						MinSize:   b.RailMin,
						MaxSize:   b.RailMax,
					}, func() {
						r.RightRail(c)
					})
			}
		})

		// Bottom chrome: status bar over the console dock. The dock is a
		// fixed-height box, so opening it shrinks the content — the
		// calc(100vh - 60px - 300px) reservation of §1, expressed
		// structurally rather than by overlaying the editor.
		ui.Column(c).Children(func() {
			ui.Box(c).Height(b.StatusBar).FillWidth().Children(func() {
				if r.StatusBar != nil {
					r.StatusBar(c)
				}
			})
			if r.State.ConsoleOpen && r.ConsoleDock != nil {
				ui.Box(c).Height(b.ConsoleDock).FillWidth().Children(func() {
					r.ConsoleDock(c)
				})
			}
		})

		// Toast host, mounted exactly once (§8.2).
		feedback.ToastHost(c, feedback.ToastHostProps{})
	})
}

// TitleBarView is the content of the title-strip region: the collection name
// at left, and at right the environment selector, its manage button and the
// run/stop control — the three things the first viewport has to answer
// (which collection, which environment, run it) before the eye reaches the
// request column. The environment strip that used to sit under this bar is
// unmounted; RootView supplies the height, the window-control insets and the
// drag region.
//
// The window's own controls are drawn here too: the window has no native
// title bar, so the app owns the buttons that minimize, maximize and close
// it. They sit at the strip's far right, where every other window puts them.
type TitleBarView struct {
	AppName        string
	CollectionName string
	ShowMenu       bool
	MenuOpen       *bool
	MenuItems      []overlay.MenuItem
	OnMenuItem     func(key string)
	OnMenu         func()

	// Environments are the names the selector offers, in order. The
	// selector is always drawn: a strip that hides the environment when
	// there is none cannot say "no environment" either, and that fact is
	// one of the three the viewport leads with.
	Environments []string
	// ActiveEnvironment is the selected name; "" means none.
	ActiveEnvironment string
	// OnSelectEnvironment reports a selection made in the strip;
	// OnManageEnvironment opens the environment settings.
	OnSelectEnvironment func(name string)
	OnManageEnvironment func()

	// InFlight swaps Run for Stop: the control keeps its slot on the right
	// edge and its meaning follows the wire. Run opens the collection
	// runner; Stop cancels what is in flight.
	InFlight bool
	OnRun    func()
	OnStop   func()

	// Maximized reports the window's state, so the window control shows the
	// restore glyph rather than the maximize one.
	Maximized bool
	// OnMinimize, OnToggleMaximize and OnClose are the window's controls.
	// They are drawn only when given: a strip that cannot close its window
	// would be a trap.
	OnMinimize       func()
	OnToggleMaximize func()
	OnClose          func()
}

// View renders the title-strip row; it fills whatever height RootView gives.
func (t *TitleBarView) View(c *ui.Context) {
	tk := uitheme.TokensOf(c)

	ui.Row(c).Fill().AlignItems(ui.Center).Gap(8).Children(func() {
		// App menu (Windows/Linux)
		if t.ShowMenu {
			if t.MenuOpen != nil && len(t.MenuItems) > 0 {
				key := overlay.DropdownMenu(c, t.MenuOpen, overlay.DropdownMenuProps{
					Items: t.MenuItems,
				}, func(trig ui.Element) {
					widgets.StyleIconButton(c, trig, widgets.IconButtonProps{
						Icon:    lucide.Menu(),
						Label:   "Menu",
						Tooltip: "Open the menu",
						Variant: action.Text,
						Size:    lobetheme.Small,
					})
				})
				if key != "" && t.OnMenuItem != nil {
					t.OnMenuItem(key)
				}
			} else if t.OnMenu != nil {
				widgets.IconButton(c, widgets.IconButtonProps{
					Icon:    lucide.Menu(),
					Label:   "Menu",
					Tooltip: "Open the menu",
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(t.OnMenu)
			}
		}

		// Collection name: the strip leads with what is open. The app name
		// stands in only while no collection is mounted, so the window is
		// never anonymous.
		name := t.CollectionName
		if name == "" {
			name = t.AppName
		}
		if name != "" {
			ui.Text(c, name).
				FontSize(uitheme.TitleSize).
				FontWeight(500).
				TextColor(tk.Text).
				SingleLine()
		}

		ui.Box(c).Grow(1) // spacer

		// Environment selector: the active name, chosen here rather than
		// in a preferences screen.
		env := t.ActiveEnvironment
		opts := make([]field.SelectOption, 0, len(t.Environments)+1)
		opts = append(opts, field.SelectOption{Value: "", Label: "No Environment"})
		for _, e := range t.Environments {
			opts = append(opts, field.SelectOption{Value: e, Label: e})
		}
		ui.Box(c).Width(150).Children(func() {
			widgets.FieldSelect(c, &env, widgets.FieldSelectProps{
				Options: opts,
				Size:    lobetheme.Small,
			})
		})
		if env != t.ActiveEnvironment && t.OnSelectEnvironment != nil {
			t.OnSelectEnvironment(env)
		}

		// Environment settings, next to the selector it configures. It
		// keeps its word: the application settings are a gear at the other
		// end of the window, and two identical gears with two meanings is
		// one meaning too many.
		if t.OnManageEnvironment != nil {
			widgets.IconButton(c, widgets.IconButtonProps{
				Icon:    lucide.Globe(),
				Text:    "Env",
				Label:   "Environment settings",
				Tooltip: "Environment settings",
				Variant: action.Text,
				Size:    lobetheme.Small,
			}).OnClick(t.OnManageEnvironment)
		}

		// Run / Stop.
		if t.InFlight {
			if t.OnStop != nil {
				widgets.Button(c, "Stop", widgets.ButtonProps{
					Variant: action.Default,
					Size:    lobetheme.Small,
				}).OnClick(t.OnStop).Tooltip("Cancel the in-flight request")
			}
		} else if t.OnRun != nil {
			widgets.Button(c, "Run", widgets.ButtonProps{
				Variant: action.Text,
				Size:    lobetheme.Small,
			}).OnClick(t.OnRun).Tooltip("Run this collection")
		}

		// The window's controls, drawn by the app: the window has no
		// native title bar, so the buttons that minimize, maximize and
		// close it live here, at the far right, where a window puts them.
		if t.OnMinimize != nil || t.OnToggleMaximize != nil || t.OnClose != nil {
			ui.Box(c).Width(6)

			if t.OnMinimize != nil {
				widgets.IconButton(c, widgets.IconButtonProps{
					Icon:    lucide.Minus(),
					Label:   "Minimize",
					Tooltip: "Minimize",
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(t.OnMinimize)
			}
			if t.OnToggleMaximize != nil {
				icon := lucide.Square()
				label := "Maximize"
				if t.Maximized {
					icon = lucide.Copy()
					label = "Restore"
				}
				widgets.IconButton(c, widgets.IconButtonProps{
					Icon:    icon,
					Label:   label,
					Tooltip: label,
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(t.OnToggleMaximize)
			}
			if t.OnClose != nil {
				widgets.IconButton(c, widgets.IconButtonProps{
					Icon:    lucide.X(),
					Label:   "Close",
					Tooltip: "Close",
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(t.OnClose)
			}
		}
	})
}

// StatusItem is one right-aligned entry of the status bar.
type StatusItem struct {
	Label   string
	Color   ui.Color
	Tooltip string
	OnClick func()
}

// StatusBarView is the content of the status-bar region: the counts and
// engine facts on the left, the workspace and settings controls closing the
// window on the right. RootView supplies the height; this fills it.
type StatusBarView struct {
	RequestCount  int
	SelectedCount int
	RightStatus   []StatusItem

	// Workspace and settings live here rather than in the title strip: the
	// title strip is spoken for by the three facts the request leads with,
	// and these two are app-level controls that close the window.
	WorkspaceName string
	OnWorkspace   func()
	OnSettings    func()
}

// View renders the status row.
func (s *StatusBarView) View(c *ui.Context) {
	tk := uitheme.TokensOf(c)

	// Top divider: without it the bar reads as dead background rather
	// than chrome, especially in dark mode where BgBase and the content
	// surface are near-neighbours.
	ui.Row(c).Fill().AlignItems(ui.Center).Gap(12).Padding(0, 12).
		BorderWidth(1, 0, 0, 0).BorderColor(tk.Border).
		Children(func() {
			// Left: request counts. A count is a figure, so it is set in the
			// data face and lines up on its digits with the rest of the
			// window's numbers.
			if s.RequestCount > 0 {
				uitheme.DataText(c, formatCount(s.RequestCount, "request", "requests")).
					TextColor(tk.TextSecondary).FontSize(tk.FontSizeSM)
				if s.SelectedCount > 0 {
					uitheme.DataText(c, " · ").TextColor(tk.TextSecondary).FontSize(tk.FontSizeSM)
					uitheme.DataText(c, formatCount(s.SelectedCount, "selected", "selected")).
						TextColor(tk.TextSecondary).FontSize(tk.FontSizeSM)
				}
			}

			ui.Box(c).Grow(1) // spacer

			// Right status items
			for _, item := range s.RightStatus {
				if item.OnClick != nil {
					widgets.Button(c, item.Label, widgets.ButtonProps{
						Variant: action.Text,
						Size:    lobetheme.Small,
					}).OnClick(item.OnClick).Tooltip(item.Tooltip)
				} else {
					ui.Text(c, item.Label).TextColor(item.Color).FontSize(tk.FontSizeSM)
				}
			}

			// Workspace selector and settings, last in the window.
			if s.WorkspaceName != "" && s.OnWorkspace != nil {
				widgets.Button(c, s.WorkspaceName, widgets.ButtonProps{
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(s.OnWorkspace).Tooltip("Switch workspace")
			}
			if s.OnSettings != nil {
				widgets.IconButton(c, widgets.IconButtonProps{
					Icon:    lucide.Settings(),
					Label:   "Settings",
					Tooltip: "Settings",
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(s.OnSettings)
			}
		})
}

// formatCount renders a counted label with an explicit plural, so
// adjectives ("3 selected") and nouns ("5 requests") both read right.
func formatCount(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %s", n, plural)
}

// ConsoleTab is one tab of the console dock.
type ConsoleTab struct {
	Label   string
	Content func(*ui.Context)
	Icon    string
	Badge   string
}

// ConsoleDockView is the content of the console-dock region: a tab strip
// over the active tab's body. RootView supplies the dock height (§8.1's
// ConsoleDock budget) and only mounts it while ConsoleOpen; this fills
// whatever it is given.
type ConsoleDockView struct {
	Tabs        []ConsoleTab
	ActiveTab   int
	OnTabChange func(int)
	OnClose     func()
}

// View renders the dock: tab strip above, active tab content below.
func (d *ConsoleDockView) View(c *ui.Context) {
	tk := uitheme.TokensOf(c)

	ui.Column(c).Fill().Background(tk.BgBase).Children(func() {
		// Tab bar
		ui.Row(c).Gap(4).Padding(8, 8).Children(func() {
			for i, tab := range d.Tabs {
				active := i == d.ActiveTab
				btn := widgets.Button(c, tab.Label, widgets.ButtonProps{
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(func() {
					if d.OnTabChange != nil {
						d.OnTabChange(i)
					}
				})
				if active {
					btn.Background(tk.Primary).TextColor(tk.PrimaryForeground)
				}
			}
			ui.Box(c).Grow(1)
			if d.OnClose != nil {
				widgets.IconButton(c, widgets.IconButtonProps{
					Icon:    lucide.X(),
					Label:   "Close console",
					Tooltip: "Close console",
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(d.OnClose)
			}
		})

		// Active tab content fills the rest of the dock.
		ui.Box(c).Grow(1).FillWidth().Children(func() {
			if d.ActiveTab >= 0 && d.ActiveTab < len(d.Tabs) && d.Tabs[d.ActiveTab].Content != nil {
				d.Tabs[d.ActiveTab].Content(c)
			}
		})
	})
}
