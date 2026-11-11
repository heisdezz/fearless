package theme

import (
	"go/parser"
	"go/token"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/theme"
)

// Bruno's request.methods, per mode, as the browser renders them. The hexes
// are the HSL palette of light.js:2 / dark.js:3 run through the table at
// light.js:262 / dark.js:270 (see brunoLightMethods/brunoDarkMethods), and
// were cross-checked against an independent HSL→sRGB conversion.
var (
	wantLightMethods = map[string]string{
		"get":     "#2e8a54", // hues.GREEN
		"post":    "#8d43b1", // hues.PURPLE
		"put":     "#c67a10", // hues.ORANGE
		"delete":  "#ce4f3b", // hues.RED
		"patch":   "#8d43b1", // hues.PURPLE
		"options": "#2e8a87", // hues.TEAL
		"head":    "#3089a6", // hues.CYAN
	}
	wantDarkMethods = map[string]string{
		"get":     "#73e89a", // hues.GREEN
		"post":    "#79c8f6", // hues.INDIGO
		"put":     "#f6ab79", // hues.ORANGE
		"delete":  "#e06552", // lighten(0.08, hues.RED)
		"patch":   "#f6ab79", // hues.ORANGE
		"options": "#52e0c9", // hues.TEAL
		"head":    "#7ddff2", // hues.CYAN
	}
	// colors.text.muted: MethodBadge's fallback for a method with no entry.
	wantMuted = [2]string{"#838383", "#aaaaaa"}
)

func methodsFor(dark bool) map[string]string {
	if dark {
		return wantDarkMethods
	}
	return wantLightMethods
}

// TestMethodColorTableIsExact is the acceptance test for "the method colour
// table is exact": all seven verbs in both appearances, value for value,
// plus the structure Bruno gives the table — post and patch share a colour
// in light, put and patch share one in dark, and post's hue itself changes
// between modes.
func TestMethodColorTableIsExact(t *testing.T) {
	for _, dark := range []bool{false, true} {
		mode := "light"
		if dark {
			mode = "dark"
		}
		a := NewAdapter()
		for method, want := range methodsFor(dark) {
			if got := a.MethodColor(method, dark); got != ui.Hex(want) {
				t.Errorf("%s MethodColor(%q) = %v, want %s (%s)", mode, method, got, want, mode)
			}
		}
	}

	a := NewAdapter()
	// The two appearances are different tables, not one table reused.
	for method := range wantLightMethods {
		if a.MethodColor(method, false) == a.MethodColor(method, true) {
			t.Errorf("method %q has the same colour in light and dark; the tables must differ", method)
		}
	}
	// Bruno's sharing structure.
	if a.MethodColor("post", false) != a.MethodColor("patch", false) {
		t.Error("light: post and patch must share hues.PURPLE")
	}
	if a.MethodColor("put", true) != a.MethodColor("patch", true) {
		t.Error("dark: put and patch must share hues.ORANGE")
	}
	if a.MethodColor("post", false) == a.MethodColor("post", true) {
		t.Error("post must change hue between light (PURPLE) and dark (INDIGO)")
	}
	if a.MethodColor("delete", false) == a.MethodColor("delete", true) {
		t.Error("delete must be lightened in dark mode")
	}
}

// TestMethodColorFallbackAndCase: the lookup is case-insensitive the way
// MethodBadge normalizes, and a verb with no entry falls back to the theme's
// colors.text.muted per mode, not to one hardcoded grey.
func TestMethodColorFallbackAndCase(t *testing.T) {
	a := NewAdapter()

	if got, want := a.MethodColor("GET", false), ui.Hex(wantLightMethods["get"]); got != want {
		t.Errorf("uppercase GET = %v, want %v (lookup must normalize)", got, want)
	}
	if got, want := a.MethodColor("Get", true), ui.Hex(wantDarkMethods["get"]); got != want {
		t.Errorf("mixed-case Get (dark) = %v, want %v", got, want)
	}
	for _, method := range []string{"trace", "connect", "UNKNOWN", ""} {
		if got, want := a.MethodColor(method, false), ui.Hex(wantMuted[0]); got != want {
			t.Errorf("light MethodColor(%q) = %v, want muted %s", method, got, wantMuted[0])
		}
		if got, want := a.MethodColor(method, true), ui.Hex(wantMuted[1]); got != want {
			t.Errorf("dark MethodColor(%q) = %v, want muted %s", method, got, wantMuted[1])
		}
	}
}

// TestHSL pins the conversion itself, including the hue wrap and the
// half-up rounding the browser applies.
func TestHSL(t *testing.T) {
	tests := []struct {
		h, s, l float64
		want    string
	}{
		{0, 100, 50, "#ff0000"},
		{120, 100, 50, "#00ff00"},
		{240, 100, 50, "#0000ff"},
		{0, 0, 100, "#ffffff"},
		{0, 0, 0, "#000000"},
		{360 + 145, 50, 36, "#2e8a54"}, // hue wraps to 145 (hues.GREEN)
	}
	for _, tc := range tests {
		if got := hsl(tc.h, tc.s, tc.l); got != ui.Hex(tc.want) {
			t.Errorf("hsl(%v, %v, %v) = %v, want %s", tc.h, tc.s, tc.l, got, tc.want)
		}
	}
	// -170° wraps to 190°, which is not a palette entry of its own — it must
	// still convert to the same colour as 190.
	if got, base := hsl(-170, 50, 36), hsl(190, 50, 36); got != base {
		t.Errorf("hsl(-170, 50, 36) = %v, want the same as hsl(190, 50, 36) = %v", got, base)
	}
	if got := hsl(190, 50, 36); got != ui.Hex("#2e7a8a") {
		t.Errorf("hsl(190, 50, 36) = %v, want #2e7a8a", got)
	}
}

// TestSetBrunoThemeOverridesDeclaredMode: a snapshot repaints only the
// appearance its Mode names, merges over the built-in table, prefers
// request.methods, and clearing restores the built-in table.
func TestSetBrunoThemeOverridesDeclaredMode(t *testing.T) {
	a := NewAdapter()
	bt := &BrunoTheme{
		Name:    "dark only",
		Mode:    "dark",
		Methods: map[string]NRGBA{"get": {Hex: "#00ff00"}, "post": {Hex: "#ff0000"}},
	}
	bt.Colors.Text.Muted = NRGBA{Hex: "#6c6f85"}
	a.SetBrunoTheme(bt)

	if got := a.MethodColor("get", true); got != ui.Hex("#00ff00") {
		t.Errorf("dark get = %v, want the imported #00ff00", got)
	}
	if got := a.MethodColor("put", true); got != ui.Hex(wantDarkMethods["put"]) {
		t.Errorf("dark put = %v, want the built-in %s (entries merge, they do not replace)", got, wantDarkMethods["put"])
	}
	if got := a.MethodColor("get", false); got != ui.Hex(wantLightMethods["get"]) {
		t.Errorf("light get = %v, want the built-in %s: a dark theme must not repaint light mode", got, wantLightMethods["get"])
	}
	if got := a.MethodColor("trace", true); got != ui.Hex("#6c6f85") {
		t.Errorf("dark fallback = %v, want the imported muted #6c6f85", got)
	}
	if got := a.MethodColor("trace", false); got != ui.Hex(wantMuted[0]) {
		t.Errorf("light fallback = %v, want the built-in %s", got, wantMuted[0])
	}

	// Clearing restores Bruno's built-in table.
	a.SetBrunoTheme(nil)
	if got := a.MethodColor("get", true); got != ui.Hex(wantDarkMethods["get"]) {
		t.Errorf("after clearing, dark get = %v, want %s", got, wantDarkMethods["get"])
	}
	if got := a.MethodColor("trace", true); got != ui.Hex(wantMuted[1]) {
		t.Errorf("after clearing, dark fallback = %v, want %s", got, wantMuted[1])
	}
}

// TestSetBrunoThemePrefersRequestMethods: Bruno's own themes put the table
// under request.methods; when both exist, that one wins.
func TestSetBrunoThemePrefersRequestMethods(t *testing.T) {
	a := NewAdapter()
	a.SetBrunoTheme(&BrunoTheme{
		Mode:    "light",
		Methods: map[string]NRGBA{"get": {Hex: "#111111"}},
		Request: RequestColors{Methods: map[string]NRGBA{"get": {Hex: "#222222"}}},
	})
	if got := a.MethodColor("get", false); got != ui.Hex("#222222") {
		t.Errorf("light get = %v, want request.methods' #222222", got)
	}
}

// TestLoadBrunoThemeFile loads the shared fixture (mode "dark") and checks
// that it repaints dark mode only, that its colours come from
// request.methods, and that bad input is reported rather than panicking.
func TestLoadBrunoThemeFile(t *testing.T) {
	a := NewAdapter()
	if err := a.LoadBrunoTheme("../../../testdata/test_theme.json"); err != nil {
		t.Fatalf("LoadBrunoTheme: %v", err)
	}
	if got := a.MethodColor("get", true); got != ui.Hex("#34a853") {
		t.Errorf("dark get = %v, want the fixture's #34a853", got)
	}
	if got := a.MethodColor("get", false); got != ui.Hex(wantLightMethods["get"]) {
		t.Errorf("light get = %v, want the built-in %s: the fixture is a dark theme", got, wantLightMethods["get"])
	}
	if got := a.MethodColor("trace", true); got != ui.Hex("#6c6f85") {
		t.Errorf("dark fallback = %v, want the fixture's muted #6c6f85", got)
	}

	// Corrupt input is an error, not a panic.
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := a.LoadBrunoTheme(bad); err == nil {
		t.Error("corrupt theme file must return an error")
	}
	if err := a.LoadBrunoTheme(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("missing theme file must return an error")
	}
}

// TestApplyFollowsSystem is the "light/dark follows system" acceptance: one
// call to Apply at the top of the view is enough for the frame to take the
// light or dark tokens and the matching method colour when the system
// appearance flips — no component asks for a theme of its own.
func TestApplyFollowsSystem(t *testing.T) {
	resetGlobal(t)

	type sample struct {
		dark bool
		get  ui.Color
		bg   ui.Color
	}
	var samples []sample

	view := func(c *ui.Context) {
		Apply(c)
		samples = append(samples, sample{
			dark: c.Theme().Dark,
			get:  MethodColorFor(c, "get"),
			bg:   TokensOf(c).BgBase,
		})
		ui.Text(c, "theme probe")
	}

	tt := ui.NewTester(view, 320, 200)

	samples = nil
	tt.Frame()
	if len(samples) == 0 {
		t.Fatal("view never rendered")
	}
	for _, s := range samples {
		if s.dark {
			t.Fatal("first frame must be light when the system is light")
		}
		if s.get != ui.Hex(wantLightMethods["get"]) {
			t.Errorf("light frame get = %v, want %s", s.get, wantLightMethods["get"])
		}
		if s.bg != lobe.Light().BgBase {
			t.Errorf("light frame BgBase = %v, want Lobe's %v", s.bg, lobe.Light().BgBase)
		}
	}

	tt.SetDark(true)
	samples = nil
	tt.Frame()
	if len(samples) == 0 {
		t.Fatal("view never rendered after SetDark")
	}
	for _, s := range samples {
		if !s.dark {
			t.Fatal("frame must be dark after the system turns dark")
		}
		if s.get != ui.Hex(wantDarkMethods["get"]) {
			t.Errorf("dark frame get = %v, want %s", s.get, wantDarkMethods["get"])
		}
		if s.bg != lobe.Dark().BgBase {
			t.Errorf("dark frame BgBase = %v, want Lobe's %v", s.bg, lobe.Dark().BgBase)
		}
	}
	if samples[0].bg == lobe.Light().BgBase {
		t.Error("dark frame kept the light background: Apply must follow the system")
	}
}

// TestApplyRebuildsAfterThemeChange: a theme imported after the first frame
// takes effect on the next Apply, and clearing it puts Lobe's tokens back —
// the config is rebuilt, not cached forever.
func TestApplyRebuildsAfterThemeChange(t *testing.T) {
	resetGlobal(t)

	var bg ui.Color
	view := func(c *ui.Context) {
		Apply(c)
		bg = TokensOf(c).BgBase
		ui.Text(c, "rebuild probe")
	}
	tt := ui.NewTester(view, 320, 200)
	tt.SetDark(true)

	tt.Frame()
	if bg != lobe.Dark().BgBase {
		t.Fatalf("dark frame BgBase = %v, want Lobe's %v", bg, lobe.Dark().BgBase)
	}

	if err := LoadBrunoTheme("../../../testdata/test_theme.json"); err != nil {
		t.Fatalf("LoadBrunoTheme: %v", err)
	}
	tt.Frame()
	if want := ui.Hex("#1e1e2e"); bg != want {
		t.Errorf("after import, dark BgBase = %v, want the fixture's %v", bg, want)
	}

	SetBrunoTheme(nil)
	tt.Frame()
	if bg != lobe.Dark().BgBase {
		t.Errorf("after clearing, dark BgBase = %v, want Lobe's %v", bg, lobe.Dark().BgBase)
	}
}

// TestNoOtherPackageImports is the "no component is restyled individually"
// acceptance, as far as this package can prove it: the theme layer reaches
// nothing — no widget, no layout, no core — it only exposes tokens and
// colours, so components can only read from it, never be painted by it.
func TestNoOtherPackageImports(t *testing.T) {
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
			if strings.HasPrefix(path, "fearless/") {
				t.Errorf("%s imports %q: the theme layer must not reach into any component", f, path)
			}
			if !strings.Contains(path, ".") {
				continue // standard library
			}
			if strings.HasPrefix(path, "github.com/egoist/mygo") ||
				strings.HasPrefix(path, "github.com/lobehub/lobe-mygo") {
				continue
			}
			t.Errorf("%s imports %q: only the standard library, MyGo and lobe-mygo are allowed here", f, path)
		}
	}
}

// TestBrunoThemeModeIsCaseInsensitive closes the review finding that the
// mode match was case-sensitive: the mode field is hand-edited JSON, so
// "Dark"/"DARK"/" dark " must select the dark table and its token mapping,
// while "light"/"LIGHT"/"" stay on the light side.
func TestBrunoThemeModeIsCaseInsensitive(t *testing.T) {
	for _, mode := range []string{"dark", "Dark", "DARK", " dark "} {
		a := NewAdapter()
		a.SetBrunoTheme(&BrunoTheme{
			Mode:       mode,
			Methods:    map[string]NRGBA{"get": {Hex: "#00ff00"}},
			Background: BackgroundColors{Base: NRGBA{Hex: "#1e1e2e"}},
		})
		if got := a.MethodColor("get", true); got != ui.Hex("#00ff00") {
			t.Errorf("mode %q: dark get = %v, want the imported #00ff00", mode, got)
		}
		if got := a.MethodColor("get", false); got != ui.Hex(wantLightMethods["get"]) {
			t.Errorf("mode %q: light get = %v, want the built-in %s", mode, got, wantLightMethods["get"])
		}
		// The same match governs which tokens the snapshot is mapped into.
		a.buildConfig()
		if got := a.cfg.Tokens(true).BgBase; got != ui.Hex("#1e1e2e") {
			t.Errorf("mode %q: dark BgBase = %v, want the imported #1e1e2e", mode, got)
		}
		if got := a.cfg.Tokens(false).BgBase; got != lobe.Light().BgBase {
			t.Errorf("mode %q: light BgBase = %v, want Lobe's untouched %v", mode, got, lobe.Light().BgBase)
		}
	}

	for _, mode := range []string{"light", "LIGHT", "", "  "} {
		a := NewAdapter()
		a.SetBrunoTheme(&BrunoTheme{
			Mode:       mode,
			Methods:    map[string]NRGBA{"get": {Hex: "#123456"}},
			Background: BackgroundColors{Base: NRGBA{Hex: "#1e1e2e"}},
		})
		if got := a.MethodColor("get", false); got != ui.Hex("#123456") {
			t.Errorf("mode %q: light get = %v, want the imported #123456", mode, got)
		}
		if got := a.MethodColor("get", true); got != ui.Hex(wantDarkMethods["get"]) {
			t.Errorf("mode %q: dark get = %v, want the built-in %s", mode, got, wantDarkMethods["get"])
		}
		a.buildConfig()
		if got := a.cfg.Tokens(false).BgBase; got != ui.Hex("#1e1e2e") {
			t.Errorf("mode %q: light BgBase = %v, want the imported #1e1e2e", mode, got)
		}
		if got := a.cfg.Tokens(true).BgBase; got != lobe.Dark().BgBase {
			t.Errorf("mode %q: dark BgBase = %v, want Lobe's untouched %v", mode, got, lobe.Dark().BgBase)
		}
	}
}

func TestNRGBAColor(t *testing.T) {
	tests := []struct {
		name string
		n    NRGBA
		want color.NRGBA
	}{
		{"hex full", NRGBA{Hex: "#ff0000"}, color.NRGBA{R: 255, G: 0, B: 0, A: 255}},
		{"hex with alpha", NRGBA{Hex: "#00ff0080"}, color.NRGBA{R: 0, G: 255, B: 0, A: 128}},
		{"raw values", NRGBA{R: 100, G: 150, B: 200, A: 255}, color.NRGBA{R: 100, G: 150, B: 200, A: 255}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.n.Color()
			if got != tc.want {
				t.Fatalf("Color() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseHexInvalid(t *testing.T) {
	n := NRGBA{Hex: "invalid"}
	c := n.Color()
	if c != (color.NRGBA{}) {
		t.Fatalf("invalid hex should return zero color, got %v", c)
	}
}

// resetGlobal gives a test the package-level singleton from scratch and puts
// a clean one back afterwards, so global-state tests cannot leak into each
// other.
func resetGlobal(t *testing.T) {
	t.Helper()
	clearGlobal()
	t.Cleanup(clearGlobal)
}

func clearGlobal() {
	globalAdapter = nil
	once = sync.Once{}
}

func TestGlobalFunctions(t *testing.T) {
	resetGlobal(t)

	a1 := Global()
	a2 := Global()
	if a1 != a2 {
		t.Fatal("Global() should return singleton")
	}

	// A theme without a mode is a light-theme import.
	SetBrunoTheme(&BrunoTheme{
		Methods: map[string]NRGBA{"get": {Hex: "#123456"}},
	})
	if MethodColor("get", false) != ui.Hex("#123456") {
		t.Fatal("global MethodColor not updated")
	}
	if MethodColor("get", true) != ui.Hex(wantDarkMethods["get"]) {
		t.Fatal("a light import must leave the dark table alone")
	}

	if err := LoadBrunoTheme("/nonexistent"); err == nil {
		t.Fatal("expected error for a missing theme file")
	}
}
