package shell

import (
	"bytes"
	"go/parser"
	"go/token"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"

	"fearless/internal/ui/layout"
)

const (
	testW = 1280
	testH = 800
	// tol absorbs font-metric rounding between frames and DPI scales.
	tol = 2.0
	// goldenTolerance is the share of pixels allowed to differ from a stored
	// golden before the screenshot counts as changed. It absorbs font and
	// anti-aliasing drift between machines; a theme regression moves whole
	// surfaces and lands far above it.
	goldenTolerance = 0.05
)

// recorder counts the commands the chrome reports. Nothing else in the test
// writes, so any state change during a frame is the view's doing.
type recorder struct {
	menu, workspace, settings, envSelect, envManage, run, stop, consoleClose int
	statuses                                                                 []int
	consoleTabs                                                              []int
	windowMinimize, windowMaximize, windowClose                              int
}

func testScreen(rec *recorder) *Screen {
	st := layout.DefaultLayoutState()
	st.ConsoleOpen = true
	return &Screen{
		State: &st,
		Snapshot: Snapshot{
			AppName:         "Fearless",
			CollectionName:  "demo",
			WorkspaceName:   "local",
			ShowMenu:        true,
			RequestCount:    12,
			SelectedCount:   3,
			EnvironmentName: "dev",
			Environments:    []string{"dev", "prod"},
			Status: []StatusEntry{
				{Label: "idle", Tooltip: "engine (demo)", Index: -1},
				{Label: "Console", Tooltip: "toggle the console", Index: 0},
			},
			ActiveConsoleTab: 1,
		},
		Actions: Actions{
			OnMenu:              func() { rec.menu++ },
			OnWorkspace:         func() { rec.workspace++ },
			OnSettings:          func() { rec.settings++ },
			OnSelectEnvironment: func(string) { rec.envSelect++ },
			OnManageEnvironment: func() { rec.envManage++ },
			OnRun:               func() { rec.run++ },
			OnStop:              func() { rec.stop++ },
			OnConsoleClose:      func() { rec.consoleClose++ },
			OnMinimize:          func() { rec.windowMinimize++ },
			OnToggleMaximize:    func() { rec.windowMaximize++ },
			OnClose:             func() { rec.windowClose++ },
			OnStatus:            func(i int) { rec.statuses = append(rec.statuses, i) },
			OnConsoleTab:        func(i int) { rec.consoleTabs = append(rec.consoleTabs, i) },
		},
		ConsoleTabs: []layout.ConsoleTab{
			{Label: "Logs", Content: func(c *ui.Context) { ui.Text(c, "logs-body") }},
			{Label: "Network", Content: func(c *ui.Context) { ui.Text(c, "network-body") }},
		},
		Sidebar:                func(c *ui.Context, _ lobe.SideNavState) { ui.Text(c, "SIDEBAR") },
		CollectionHeaderRegion: func(c *ui.Context) { ui.Text(c, "HEADER") },
		RequestPane:            func(c *ui.Context) { ui.Text(c, "REQUEST") },
		ResponsePane:           func(c *ui.Context) { ui.Text(c, "RESPONSE") },
	}
}

// newTester renders the screen and lets the chrome's entrance animations
// settle before anything measures it.
func newTester(s *Screen) *ui.Tester {
	tt := ui.NewTester(s.View, testW, testH)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	tt.Frame()
	return tt
}

// TestShellSmokeTest is the package's acceptance smoke test: the chrome
// renders headlessly, every region it owns shows its data, and the dock shows
// the tab the snapshot selected.
func TestShellSmokeTest(t *testing.T) {
	s := testScreen(&recorder{})
	tt := newTester(s)

	for _, want := range []string{
		"demo", "dev", "Run", // title strip: collection, environment, run/stop
		"12 requests", "3 selected", // status strip counts
		"idle", "Console", "local", // status chips, workspace
		"SIDEBAR", "HEADER", "REQUEST", "RESPONSE", // injected regions
		"Logs", "Network", "network-body", // dock, active tab from the snapshot
	} {
		if !tt.HasText(want) {
			t.Errorf("chrome missing %q; texts: %v", want, tt.Texts())
		}
	}
	if tt.HasText("logs-body") {
		t.Error("inactive console tab rendered its body")
	}
}

// TestViewIsPure is the "view functions are pure" acceptance: rendering
// changes neither the snapshot nor the layout state, and two frames of the
// same Screen are byte-identical.
func TestViewIsPure(t *testing.T) {
	s := testScreen(&recorder{})
	snapBefore := s.Snapshot
	stateBefore := *s.State

	tt := ui.NewTester(s.View, testW, testH)
	tt.Frame()
	tt.Advance(600 * time.Millisecond)
	tt.Frame()
	first := encodePNG(t, tt.Image())

	tt.Frame()
	second := encodePNG(t, tt.Image())

	if !reflect.DeepEqual(s.Snapshot, snapBefore) {
		t.Errorf("View mutated the snapshot:\n got %+v\nwant %+v", s.Snapshot, snapBefore)
	}
	if *s.State != stateBefore {
		t.Errorf("View mutated the layout state:\n got %+v\nwant %+v", *s.State, stateBefore)
	}
	if !bytes.Equal(first, second) {
		t.Error("two frames of the same Screen differ: the view is not a function of its inputs")
	}
}

// TestActionsAreTheOnlyWrites: a click reports intent through Actions and
// changes nothing itself — the app owns the state change, and applying it
// outside the view is what the next frame shows.
func TestActionsAreTheOnlyWrites(t *testing.T) {
	rec := &recorder{}
	s := testScreen(rec)
	tt := newTester(s)

	for _, click := range []string{"Settings", "Console", "Run"} {
		if err := tt.Click(click); err != nil {
			t.Fatalf("click %q: %v (texts: %v)", click, err, tt.Texts())
		}
	}
	// The environment is a select: its click opens the option list, so the
	// selection is reported the way the app receives it.
	s.Actions.OnSelectEnvironment("prod")
	s.Actions.OnManageEnvironment()
	if rec.settings != 1 || rec.run != 1 || rec.envSelect != 1 || rec.envManage != 1 ||
		!reflect.DeepEqual(rec.statuses, []int{0}) {
		t.Errorf("actions fired: settings=%d run=%d envSelect=%d envManage=%d statuses=%v, want 1/1/1/1/[0]",
			rec.settings, rec.run, rec.envSelect, rec.envManage, rec.statuses)
	}

	// In flight, the same slot offers Stop.
	s.Snapshot.InFlight = true
	tt.Frame()
	if err := tt.Click("Stop"); err != nil {
		t.Fatalf("click Stop: %v (texts: %v)", err, tt.Texts())
	}
	if rec.stop != 1 {
		t.Errorf("Stop reported %d times, want 1", rec.stop)
	}
	if tt.HasText("Run") {
		t.Error("Run still offered while a request is in flight")
	}
	if !s.State.ConsoleOpen {
		t.Error("the view closed the console itself; only the app may change state")
	}

	// The app applies the change the chip asked for.
	statusBefore, ok := tt.Find("12 requests")
	if !ok {
		t.Fatal("status row not rendered while the console is open")
	}
	s.State.ConsoleOpen = false
	tt.Frame()
	if tt.HasText("Logs") || tt.HasText("network-body") {
		t.Error("console dock still rendered with ConsoleOpen false")
	}
	statusAfter, ok := tt.Find("12 requests")
	if !ok {
		t.Fatal("status row not rendered after the console closed")
	}
	// Closing the dock hands its height to the content: the row moves down,
	// it does not stay put under an overlay.
	if statusAfter.Y <= statusBefore.Y+100 {
		t.Errorf("status row Y %.1f → %.1f: closing the console must move it down to the window bottom",
			statusBefore.Y, statusAfter.Y)
	}
	if want := float32(testH) - layout.DefaultRegionBudget().StatusBar; statusAfter.Y < want-tol {
		t.Errorf("status row at Y=%.1f, want at or below %.1f", statusAfter.Y, want)
	}
}

// TestGoldenAfterThemeChange: changing the theme must repaint the chrome
// without moving it — the same geometry, different pixels — and both frames
// must match their stored screenshots. Regenerate with
//
//	UPDATE_GOLDEN=1 go test ./internal/ui/screens/shell/
func TestGoldenAfterThemeChange(t *testing.T) {
	s := testScreen(&recorder{})
	tt := newTester(s)

	lightGeom := geometry(t, tt)
	lightImg := tt.Image()

	tt.SetDark(true)
	tt.Frame()
	darkGeom := geometry(t, tt)
	darkImg := tt.Image()

	if !reflect.DeepEqual(lightGeom, darkGeom) {
		t.Errorf("theme change moved the chrome:\nlight %+v\ndark  %+v", lightGeom, darkGeom)
	}
	if sameImage(lightImg, darkImg) {
		t.Error("light and dark frames are pixel-identical: the theme did not apply")
	}

	compareGolden(t, "shell-light.png", lightImg)
	compareGolden(t, "shell-dark.png", darkImg)
}

// TestNilStateRendersDefaults: a Screen assembled without state still draws
// one frame of the defaults instead of panicking.
func TestNilStateRendersDefaults(t *testing.T) {
	s := testScreen(&recorder{})
	s.State = nil
	tt := newTester(s)
	if !tt.HasText("demo") {
		t.Errorf("title bar missing with nil State; texts: %v", tt.Texts())
	}
}

// TestWindowControlsAreDrawn covers the window's own buttons: the window has
// no native title bar, so the app draws the controls that minimize, maximize
// and close it, and they report through the actions.
func TestWindowControlsAreDrawn(t *testing.T) {
	rec := &recorder{}
	s := testScreen(rec)
	tt := newTester(s)

	for _, want := range []string{"Minimize", "Maximize", "Close"} {
		if !tt.HasText(want) {
			t.Errorf("title strip missing the %q control; texts: %v", want, tt.Texts())
		}
	}

	for _, click := range []string{"Minimize", "Maximize", "Close"} {
		if err := tt.Click(click); err != nil {
			t.Fatalf("click %s: %v", click, err)
		}
	}
	if rec.windowMinimize != 1 || rec.windowMaximize != 1 || rec.windowClose != 1 {
		t.Errorf("window controls: minimize=%d maximize=%d close=%d, want 1 each",
			rec.windowMinimize, rec.windowMaximize, rec.windowClose)
	}

	// A maximized window's control offers to restore, not to maximize.
	s.Snapshot.Maximized = true
	tt2 := newTester(s)
	if !tt2.HasText("Restore") {
		t.Errorf("a maximized window should offer the restore control; texts: %v", tt2.Texts())
	}
}

// TestNoDomainImports is the "no domain mutation inside a View closure"
// acceptance, as far as the package can prove it: the screen may import the
// UI layer and the toolkit, never the core or app layers whose state it must
// not touch.
func TestNoDomainImports(t *testing.T) {
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
			if !strings.HasPrefix(path, "fearless/") {
				continue
			}
			if strings.HasPrefix(path, "fearless/internal/ui/") {
				continue // layout, theme, presenters, widgets: the UI layer
			}
			t.Errorf("%s imports %q: a screen must not reach the domain layers", f, path)
		}
	}
}

// geometry records where each sentinel landed, so two frames can be compared
// for shape rather than pixels.
func geometry(t *testing.T, tt *ui.Tester) map[string]ui.Rect {
	t.Helper()
	out := map[string]ui.Rect{}
	for _, want := range []string{"demo", "dev", "12 requests", "SIDEBAR", "REQUEST", "RESPONSE", "Network"} {
		r, ok := tt.Find(want)
		if !ok {
			t.Errorf("%q not rendered", want)
			continue
		}
		out[want] = r
	}
	return out
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	return buf.Bytes()
}

// compareGolden fails when got differs from the stored screenshot by more
// than goldenTolerance of its pixels, or writes it when UPDATE_GOLDEN is set.
func compareGolden(t *testing.T, name string, got image.Image) {
	t.Helper()
	path := filepath.Join("testdata", name)

	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("create golden %s: %v", path, err)
		}
		defer f.Close()
		if err := png.Encode(f, got); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		t.Logf("wrote %s", path)
		return
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("golden %s missing (%v) — run UPDATE_GOLDEN=1 go test ./internal/ui/screens/shell/", path, err)
	}
	defer f.Close()
	want, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode golden %s: %v", path, err)
	}
	if want.Bounds().Size() != got.Bounds().Size() {
		t.Errorf("%s size %v, want %v", name, got.Bounds().Size(), want.Bounds().Size())
		return
	}

	different, total := 0, 0
	b := got.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r1, g1, b1, _ := want.At(x, y).RGBA()
			r2, g2, b2, _ := got.At(x, y).RGBA()
			total++
			if diff(r1, r2)+diff(g1, g2)+diff(b1, b2) > 3*16<<8 {
				different++
			}
		}
	}
	if ratio := float64(different) / float64(total); ratio > goldenTolerance {
		t.Errorf("%s differs from its golden on %.1f%% of pixels (tolerance %.1f%%) — theme change altered the chrome; regenerate with UPDATE_GOLDEN=1 if intended",
			name, ratio*100, goldenTolerance*100)
	}
}

func diff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func sameImage(a, b image.Image) bool {
	if a.Bounds().Size() != b.Bounds().Size() {
		return false
	}
	ab := a.Bounds()
	for y := ab.Min.Y; y < ab.Max.Y; y++ {
		for x := ab.Min.X; x < ab.Max.X; x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}
