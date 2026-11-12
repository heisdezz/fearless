package layout

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"
)

// The window the headless tests lay out in, in DIPs.
const (
	testW = 1200
	testH = 800
	// tol absorbs font-metric rounding between DPI scales.
	tol = 2.0
)

// sentinelRoot wires a RootView whose regions each show one findable word,
// so ui.Tester.Find can measure where every region landed.
func sentinelRoot(st LayoutState, rail bool) *RootView {
	r := &RootView{
		State:            &st,
		Budget:           DefaultRegionBudget(),
		TitleBar:         func(c *ui.Context) { ui.Text(c, "TITLE") },
		Sidebar:          func(c *ui.Context, _ lobe.SideNavState) { ui.Text(c, "SIDEBAR") },
		CollectionHeader: func(c *ui.Context) { ui.Text(c, "HEADER") },
		RequestPane:      func(c *ui.Context) { ui.Text(c, "REQUEST") },
		ResponsePane:     func(c *ui.Context) { ui.Text(c, "RESPONSE") },
		StatusBar:        func(c *ui.Context) { ui.Text(c, "STATUS") },
		ConsoleDock:      func(c *ui.Context) { ui.Text(c, "CONSOLE") },
	}
	if rail {
		r.RightRail = func(c *ui.Context) { ui.Text(c, "RAIL") }
	}
	return r
}

// newTester renders root in a testW×testH window and lets the entrance
// animations of the side nav and rail settle before measuring.
func newTester(root *RootView) *ui.Tester {
	tt := ui.NewTester(root.View, testW, testH)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	tt.Frame()
	return tt
}

// requireIn fails unless the text rect lies inside the horizontal band
// [loY, hiY) of the window.
func requireIn(t *testing.T, scale float32, name string, r ui.Rect, loY, hiY float32) {
	t.Helper()
	if r.Y < loY-tol || r.Y+r.H > hiY+tol {
		t.Errorf("scale=%.1f %s occupies Y [%.1f, %.1f], outside band [%.1f, %.1f]",
			scale, name, r.Y, r.Y+r.H, loY, hiY)
	}
}

// TestRegionBudgetAtDPIScales is the §8.1 acceptance test: every region
// keeps its budgeted band at 100 %, 150 % and 200 % DPI, and the layout in
// DIPs is identical across scales (MyGo does the device-pixel conversion).
func TestRegionBudgetAtDPIScales(t *testing.T) {
	st := DefaultLayoutState()
	st.ConsoleOpen = true
	root := sentinelRoot(st, false)
	b := DefaultRegionBudget()

	// Bands for console open, stacked, from §8.1:
	// title [0,30) header [30,70) panes [70,476) status [476,500) console [500,800)
	bands := map[string][2]float32{
		"TITLE":    {0, b.TitleBar},
		"HEADER":   {b.TitleBar, b.TitleBar + b.CollectionHeader},
		"REQUEST":  {b.TitleBar + b.CollectionHeader, testH - b.StatusBar - b.ConsoleDock},
		"RESPONSE": {b.TitleBar + b.CollectionHeader, testH - b.StatusBar - b.ConsoleDock},
		"STATUS":   {testH - b.StatusBar - b.ConsoleDock, testH - b.ConsoleDock},
		"CONSOLE":  {testH - b.ConsoleDock, testH},
		"SIDEBAR":  {b.TitleBar, testH - b.StatusBar - b.ConsoleDock},
	}

	tt := newTester(root)
	var at100 map[string]ui.Rect

	for _, scale := range []float32{1.0, 1.5, 2.0} {
		tt.SetScale(scale)
		tt.Advance(100 * time.Millisecond)
		tt.Frame()

		got := map[string]ui.Rect{}
		for name, band := range bands {
			r, ok := tt.Find(name)
			if !ok {
				t.Errorf("scale=%.1f region %s not rendered", scale, name)
				continue
			}
			requireIn(t, scale, name, r, band[0], band[1])
			got[name] = r
		}

		// Stacked: the request pane sits above the response pane.
		if r, ok := got["REQUEST"]; ok {
			if resp, ok2 := got["RESPONSE"]; ok2 && r.Y >= resp.Y {
				t.Errorf("scale=%.1f stacked layout: request Y=%.1f not above response Y=%.1f",
					scale, r.Y, resp.Y)
			}
		}

		// The sidebar stays inside its §8.1 width budget.
		if r, ok := got["SIDEBAR"]; ok && r.X+r.W > b.SidebarMax+tol {
			t.Errorf("scale=%.1f sidebar right edge %.1f exceeds SidebarMax %.1f",
				scale, r.X+r.W, b.SidebarMax)
		}

		// Rendering must not mutate the persisted ratio.
		if st.SplitPercent != 55 {
			t.Errorf("scale=%.1f View mutated SplitPercent to %v", scale, st.SplitPercent)
		}

		// Device pixels follow the scale; DIPs do not.
		img := tt.Image()
		if wantW, wantH := int(testW*scale), int(testH*scale); img.Bounds().Dx() != wantW || img.Bounds().Dy() != wantH {
			t.Errorf("scale=%.1f image %v, want %dx%d", scale, img.Bounds(), wantW, wantH)
		}

		// The DIP layout at this scale matches 100 %.
		if scale == 1.0 {
			at100 = got
			continue
		}
		for name, r := range got {
			base, ok := at100[name]
			if !ok {
				continue
			}
			if d := abs32(r.Y - base.Y); d > tol {
				t.Errorf("%s moved %.1f DIPs vertically between 100%% and %.0f%% DPI (%.1f → %.1f)",
					name, d, scale*100, base.Y, r.Y)
			}
			if d := abs32(r.X - base.X); d > tol {
				t.Errorf("%s moved %.1f DIPs horizontally between 100%% and %.0f%% DPI (%.1f → %.1f)",
					name, d, scale*100, base.X, r.X)
			}
		}
	}
}

// TestConsoleShrinksContent covers §1: closing the console gives its height
// back to the content; opening it shrinks rather than overlays.
func TestConsoleShrinksContent(t *testing.T) {
	b := DefaultRegionBudget()

	open := DefaultLayoutState()
	open.ConsoleOpen = true
	ttOpen := newTester(sentinelRoot(open, false))
	status, ok := ttOpen.Find("STATUS")
	if !ok {
		t.Fatal("STATUS not rendered with console open")
	}
	requireIn(t, 1, "STATUS(console open)", status, testH-b.StatusBar-b.ConsoleDock, testH-b.ConsoleDock)

	closed := DefaultLayoutState()
	ttClosed := newTester(sentinelRoot(closed, false))
	if _, ok := ttClosed.Find("CONSOLE"); ok {
		t.Error("console rendered while ConsoleOpen is false")
	}
	status, ok = ttClosed.Find("STATUS")
	if !ok {
		t.Fatal("STATUS not rendered with console closed")
	}
	requireIn(t, 1, "STATUS(console closed)", status, testH-b.StatusBar, testH)
}

// TestSideBySideOrientation: changeLayout flips only the constructor —
// the response pane moves to the right of the request pane.
func TestSideBySideOrientation(t *testing.T) {
	st := DefaultLayoutState()
	st.ConsoleOpen = true
	st.RequestResponseOrientation = OrientationSideBySide
	tt := newTester(sentinelRoot(st, false))

	req, ok := tt.Find("REQUEST")
	if !ok {
		t.Fatal("REQUEST not rendered")
	}
	resp, ok := tt.Find("RESPONSE")
	if !ok {
		t.Fatal("RESPONSE not rendered")
	}
	if resp.X <= req.X {
		t.Errorf("side-by-side: response X=%.1f not right of request X=%.1f", resp.X, req.X)
	}
	b := DefaultRegionBudget()
	requireIn(t, 1, "REQUEST(side-by-side)", req, b.TitleBar+b.CollectionHeader, testH-b.StatusBar-b.ConsoleDock)
	requireIn(t, 1, "RESPONSE(side-by-side)", resp, b.TitleBar+b.CollectionHeader, testH-b.StatusBar-b.ConsoleDock)
}

// TestRightRailPushesPanes: the DraggablePanel rail lives in the body's
// flow, so expanding it narrows the request/response region instead of
// covering it, and stays inside RailMin/RailMax.
func TestRightRailPushesPanes(t *testing.T) {
	st := DefaultLayoutState()
	st.ConsoleOpen = true
	st.RailExpanded = true
	b := DefaultRegionBudget()

	tt := newTester(sentinelRoot(st, true))
	rail, ok := tt.Find("RAIL")
	if !ok {
		t.Fatal("RAIL not rendered when RailExpanded")
	}
	req, ok := tt.Find("REQUEST")
	if !ok {
		t.Fatal("REQUEST not rendered")
	}
	if rail.X+tol < float32(testW)-b.RailMax || rail.X > float32(testW)-b.RailMin+tol {
		t.Errorf("rail left edge %.1f outside [%d-RailMax, %d-RailMin] = [%.1f, %.1f]",
			rail.X, testW, testW, float32(testW)-b.RailMax, float32(testW)-b.RailMin)
	}
	if req.X+req.W > rail.X+tol {
		t.Errorf("request pane [%.1f,%.1f] overlaps rail at %.1f", req.X, req.X+req.W, rail.X)
	}
}

// TestCollapsedSidebarStaysInRail: a collapsed nav never exceeds the
// SidebarCollapsed rail width by more than the label that spills into it.
func TestCollapsedSidebarStaysInRail(t *testing.T) {
	st := DefaultLayoutState()
	st.SidebarExpanded = false
	b := DefaultRegionBudget()

	tt := newTester(sentinelRoot(st, false))
	rail, ok := tt.Find("SIDEBAR")
	if !ok {
		t.Fatal("SIDEBAR not rendered")
	}
	if rail.X > b.SidebarCollapsed+tol {
		t.Errorf("collapsed sidebar starts at X=%.1f, beyond rail width %.1f", rail.X, b.SidebarCollapsed)
	}
	if rail.X+rail.W > b.SidebarMin+tol {
		t.Errorf("collapsed sidebar spans to %.1f, beyond SidebarMin %.1f", rail.X+rail.W, b.SidebarMin)
	}
}

// TestStatePersistsThroughJSON is the split-ratio persistence acceptance:
// what EncodeState writes, DecodeState reads back unchanged.
func TestStatePersistsThroughJSON(t *testing.T) {
	b := DefaultRegionBudget()
	st := DefaultLayoutState()
	st.SidebarWidth = 350
	st.SplitPercent = 72.5
	st.RequestResponseOrientation = OrientationSideBySide
	st.ConsoleOpen = true
	st.RailWidth = 400
	st.RailExpanded = true
	st.SidebarExpanded = false

	data, err := EncodeState(st)
	if err != nil {
		t.Fatalf("EncodeState: %v", err)
	}
	got, err := DecodeState(data, b)
	if err != nil {
		t.Fatalf("DecodeState: %v", err)
	}
	if got != st {
		t.Errorf("round trip changed state:\n got %+v\nwant %+v", got, st)
	}
}

// TestDecodeRepairsState: corrupt input errors out, hostile or stale values
// are normalised — the layout can never be driven out of its budget by the
// settings file.
func TestDecodeRepairsState(t *testing.T) {
	b := DefaultRegionBudget()

	if _, err := DecodeState([]byte("{not json"), b); err == nil {
		t.Error("corrupt state must return an error, not a zero state")
	}

	got, err := DecodeState([]byte(`{}`), b)
	if err != nil {
		t.Fatalf("empty state: %v", err)
	}
	if want := DefaultLayoutState(); got != want {
		t.Errorf("empty state decoded to %+v, want defaults %+v", got, want)
	}

	got, err = DecodeState([]byte(`{
		"sidebar_width": 99999,
		"split_percent": 500,
		"orientation": 7,
		"rail_width": 0.5
	}`), b)
	if err != nil {
		t.Fatalf("hostile state: %v", err)
	}
	if got.SidebarWidth != b.SidebarMax {
		t.Errorf("sidebar_width = %v, want clamped to %v", got.SidebarWidth, b.SidebarMax)
	}
	if got.SplitPercent != maxSplitPercent {
		t.Errorf("split_percent = %v, want clamped to %v", got.SplitPercent, maxSplitPercent)
	}
	if got.RequestResponseOrientation != OrientationStacked {
		t.Errorf("orientation = %v, want repaired to OrientationStacked", got.RequestResponseOrientation)
	}
	if got.RailWidth != b.RailMin {
		t.Errorf("rail_width = %v, want clamped to %v", got.RailWidth, b.RailMin)
	}
}

// TestNormalizeDefaults: a zeroed state gets its numbers back from the
// default structure and stays inside budget; bools are taken as written
// (DecodeState overlays on the defaults, so absent keys get default bools).
func TestNormalizeDefaults(t *testing.T) {
	b := DefaultRegionBudget()
	var s LayoutState
	s.Normalize(b)
	d := DefaultLayoutState()
	if s.SidebarWidth != d.SidebarWidth || s.SplitPercent != d.SplitPercent ||
		s.RailWidth != d.RailWidth || s.RequestResponseOrientation != d.RequestResponseOrientation {
		t.Errorf("Normalize(zero) numerics = %+v, want defaults %+v", s, d)
	}
	if s.SidebarExpanded {
		t.Error("Normalize must take bools as written, not force defaults")
	}

	s = LayoutState{SidebarWidth: 100, SplitPercent: 1, RailWidth: 9000}
	s.Normalize(b)
	if s.SidebarWidth != b.SidebarMin || s.SplitPercent != minSplitPercent || s.RailWidth != b.RailMax {
		t.Errorf("Normalize clamps: %+v", s)
	}
}

// TestDegrade: §8.4 below 900 DIP the sidebar collapses and the panes
// stack; growing back does not reopen what it shut.
func TestDegrade(t *testing.T) {
	b := DefaultRegionBudget()
	st := DefaultLayoutState()
	st.RequestResponseOrientation = OrientationSideBySide

	st.Degrade(b.MinWindowWidth-1, b)
	if st.SidebarExpanded {
		t.Error("degraded mode must collapse the sidebar")
	}
	if st.RequestResponseOrientation != OrientationStacked {
		t.Error("degraded mode must force stacked panes")
	}

	st.Degrade(1600, b)
	if st.SidebarExpanded {
		t.Error("growing back must not reopen the sidebar")
	}

	fresh := DefaultLayoutState()
	fresh.RequestResponseOrientation = OrientationSideBySide
	fresh.Degrade(0, b) // unknown width: leave the state alone
	if !fresh.SidebarExpanded || fresh.RequestResponseOrientation != OrientationSideBySide {
		t.Error("zero width must not degrade the state")
	}
}

// TestImportsAreDomainFree is the §8.3 acceptance: the layout package may
// import the theme, its own presentation widgets, the toolkit and the stdlib —
// never the core or app layers whose state it must not read.
func TestImportsAreDomainFree(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no source files found")
	}
	for _, f := range files {
		file, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("unquote import in %s: %v", f, err)
			}
			if path == "fearless/internal/ui/theme" {
				continue // the declared UI-THEME dependency
			}
			if path == "fearless/internal/ui/widgets" {
				continue // presentation primitives: control and surface shims, no state
			}
			if strings.HasPrefix(path, "fearless/") {
				t.Errorf("%s imports %q: the layout layer must not read domain state (§8.3)", f, path)
			}
		}
	}
}

// TestChromeViewsRender: the title bar, status bar and console dock fill
// the regions RootView budgets for them, and the dock switches tabs.
func TestChromeViewsRender(t *testing.T) {
	st := DefaultLayoutState()
	st.ConsoleOpen = true

	title := &TitleBarView{
		AppName:             "Fearless",
		CollectionName:      "demo",
		ShowMenu:            true,
		OnMenu:              func() {},
		Environments:        []string{"dev", "prod"},
		ActiveEnvironment:   "dev",
		OnSelectEnvironment: func(string) {},
		OnManageEnvironment: func() {},
		OnRun:               func() {},
	}
	status := &StatusBarView{
		RequestCount:  12,
		SelectedCount: 3,
		WorkspaceName: "local",
		OnWorkspace:   func() {},
		OnSettings:    func() {},
		RightStatus:   []StatusItem{{Label: "idle"}},
	}
	dock := &ConsoleDockView{
		Tabs: []ConsoleTab{
			{Label: "Logs", Content: func(c *ui.Context) { ui.Text(c, "logs-body") }},
			{Label: "Network", Content: func(c *ui.Context) { ui.Text(c, "network-body") }},
		},
		ActiveTab: 0,
	}
	dock.OnTabChange = func(i int) { dock.ActiveTab = i }

	root := &RootView{
		State:            &st,
		Budget:           DefaultRegionBudget(),
		TitleBar:         func(c *ui.Context) { title.View(c) },
		CollectionHeader: func(c *ui.Context) { ui.Text(c, "HEADER") },
		RequestPane:      func(c *ui.Context) { ui.Text(c, "REQUEST") },
		ResponsePane:     func(c *ui.Context) { ui.Text(c, "RESPONSE") },
		StatusBar:        func(c *ui.Context) { status.View(c) },
		ConsoleDock:      func(c *ui.Context) { dock.View(c) },
	}

	tt := newTester(root)
	for _, want := range []string{"demo", "local", "12 requests", "3 selected", "dev", "idle", "Logs", "logs-body"} {
		if !tt.HasText(want) {
			t.Errorf("chrome missing %q; texts: %v", want, tt.Texts())
		}
	}

	if err := tt.Click("Network"); err != nil {
		t.Fatalf("click Network tab: %v", err)
	}
	if !tt.HasText("network-body") {
		t.Errorf("tab switch did not render; texts: %v", tt.Texts())
	}
}

// TestNilCollectionHeaderUnmountsTheRegion: the main window leaves this
// callback nil now that the environment selector and run live in the title
// strip, and nil must cost zero height. An empty 40 DIP band above the
// request pane would be a hole in the first viewport, not chrome.
func TestNilCollectionHeaderUnmountsTheRegion(t *testing.T) {
	st := DefaultLayoutState()
	root := sentinelRoot(st, false)
	root.CollectionHeader = nil

	tt := newTester(root)
	if tt.HasText("HEADER") {
		t.Error("HEADER rendered with a nil CollectionHeader callback")
	}
	b := DefaultRegionBudget()
	r, ok := tt.Find("REQUEST")
	if !ok {
		t.Fatal("request pane not rendered")
	}
	requireIn(t, 1, "REQUEST(nil header)", r, b.TitleBar, testH-b.StatusBar)
	if r.Y > b.TitleBar+tol {
		t.Errorf("request pane starts at Y=%.1f, want at the title strip's edge (%.1f): the unmounted region still costs height",
			r.Y, b.TitleBar)
	}
}

// TestViewNormalizesHostileState closes the review finding that View read
// the state without normalising it: a state that never went through
// DecodeState (built in memory, or written by an older build) is repaired on
// the first frame and still lays out inside the §8.1 budget.
func TestViewNormalizesHostileState(t *testing.T) {
	b := DefaultRegionBudget()
	st := LayoutState{
		SidebarWidth:               99999,
		SplitPercent:               500,
		RequestResponseOrientation: 7,
		RailWidth:                  0.5,
	}
	root := sentinelRoot(st, true)

	tt := newTester(root)

	if root.State.SidebarWidth != b.SidebarMax {
		t.Errorf("sidebar width = %v, want clamped to %v", root.State.SidebarWidth, b.SidebarMax)
	}
	if root.State.SplitPercent != maxSplitPercent {
		t.Errorf("split percent = %v, want clamped to %v", root.State.SplitPercent, maxSplitPercent)
	}
	if root.State.RequestResponseOrientation != OrientationStacked {
		t.Errorf("orientation = %v, want repaired to stacked", root.State.RequestResponseOrientation)
	}
	if root.State.RailWidth != b.RailMin {
		t.Errorf("rail width = %v, want clamped to %v", root.State.RailWidth, b.RailMin)
	}

	// And the rendered frame is still inside the budget.
	if r, ok := tt.Find("SIDEBAR"); ok && r.X+r.W > b.SidebarMax+tol {
		t.Errorf("sidebar right edge %.1f exceeds SidebarMax %v after repair", r.X+r.W, b.SidebarMax)
	}
	if r, ok := tt.Find("STATUS"); ok {
		requireIn(t, 1, "STATUS(repaired)", r, testH-b.StatusBar, testH)
	}
	// Console closed: the panes must fill down to the status bar and no
	// further, whatever the hostile values asked for.
	if r, ok := tt.Find("REQUEST"); ok {
		requireIn(t, 1, "REQUEST(repaired)", r, b.TitleBar+b.CollectionHeader, testH-b.StatusBar)
	}
}

func TestDefaultRegionBudget(t *testing.T) {
	b := DefaultRegionBudget()
	want := RegionBudget{
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
	if b != want {
		t.Errorf("DefaultRegionBudget() = %+v, want %+v", b, want)
	}
	if got := (RegionBudget{}).withDefaults(); got != want {
		t.Errorf("zero budget withDefaults() = %+v, want %+v", got, want)
	}
	partial := RegionBudget{ConsoleDock: 250}.withDefaults()
	if partial.ConsoleDock != 250 || partial.TitleBar != 30 {
		t.Errorf("partial budget withDefaults() = %+v", partial)
	}
}

func TestDefaultLayoutState(t *testing.T) {
	s := DefaultLayoutState()
	want := LayoutState{
		SidebarWidth:               248,
		SidebarExpanded:            true,
		RequestResponseOrientation: OrientationStacked,
		SplitPercent:               55,
		ConsoleOpen:                false,
		RailWidth:                  320,
		RailExpanded:               false,
	}
	if s != want {
		t.Errorf("DefaultLayoutState() = %+v, want %+v", s, want)
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		n        int
		singular string
		plural   string
		want     string
	}{
		{1, "request", "requests", "1 request"},
		{0, "request", "requests", "0 requests"},
		{5, "item", "items", "5 items"},
		{2, "selected", "selected", "2 selected"},
	}
	for _, tc := range tests {
		if got := formatCount(tc.n, tc.singular, tc.plural); got != tc.want {
			t.Errorf("formatCount(%d, %q, %q) = %q, want %q", tc.n, tc.singular, tc.plural, got, tc.want)
		}
	}
}

func TestOrientationConstants(t *testing.T) {
	if OrientationStacked != 0 {
		t.Errorf("OrientationStacked = %v, want 0", OrientationStacked)
	}
	if OrientationSideBySide != 1 {
		t.Errorf("OrientationSideBySide = %v, want 1", OrientationSideBySide)
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
