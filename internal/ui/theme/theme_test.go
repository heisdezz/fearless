package theme

import (
	"image/color"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestDefaultMethodColors(t *testing.T) {
	a := NewAdapter()

	tests := []struct {
		method string
		want   ui.Color
	}{
		{"get", ui.Hex("#34a853")},
		{"post", ui.Hex("#fbbc04")},
		{"put", ui.Hex("#ea4335")},
		{"delete", ui.Hex("#ea4335")},
		{"patch", ui.Hex("#ea4335")},
		{"options", ui.Hex("#4285f4")},
		{"head", ui.Hex("#9aa0a6")},
		{"unknown", ui.Hex("#6b7280")},
	}

	for _, tc := range tests {
		t.Run(tc.method, func(t *testing.T) {
			got := a.MethodColor(tc.method)
			if got != tc.want {
				t.Fatalf("MethodColor(%q) = %v, want %v", tc.method, got, tc.want)
			}
		})
	}
}

func TestSetBrunoTheme(t *testing.T) {
	a := NewAdapter()

	bt := &BrunoTheme{
		Name: "Test Theme",
		Mode: "dark",
		Methods: map[string]NRGBA{
			"get":  {Hex: "#00ff00"},
			"post": {Hex: "#ff0000"},
		},
	}

	a.SetBrunoTheme(bt)

	if a.MethodColor("get") != ui.Hex("#00ff00") {
		t.Fatalf("get method color not updated")
	}
	if a.MethodColor("post") != ui.Hex("#ff0000") {
		t.Fatalf("post method color not updated")
	}
	if a.MethodColor("put") != ui.Hex("#ea4335") {
		t.Fatalf("put method color should remain default")
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

func TestLoadBrunoThemeInvalidPath(t *testing.T) {
	a := NewAdapter()
	err := a.LoadBrunoTheme("/nonexistent/path/theme.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestParseHexInvalid(t *testing.T) {
	n := NRGBA{Hex: "invalid"}
	c := n.Color()
	if c != (color.NRGBA{}) {
		t.Fatalf("invalid hex should return zero color, got %v", c)
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Reset global for test
	globalAdapter = nil
	once = sync.Once{}

	// Test Global returns same instance
	a1 := Global()
	a2 := Global()
	if a1 != a2 {
		t.Fatal("Global() should return singleton")
	}

	// Test SetBrunoTheme via global
	SetBrunoTheme(&BrunoTheme{
		Methods: map[string]NRGBA{"get": {Hex: "#123456"}},
	})
	if MethodColor("get") != ui.Hex("#123456") {
		t.Fatal("global MethodColor not updated")
	}

	// Test LoadBrunoTheme via global (invalid path)
	err := LoadBrunoTheme("/nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadBrunoThemeFile(t *testing.T) {
	a := NewAdapter()
	err := a.LoadBrunoTheme("../../../testdata/test_theme.json")
	if err != nil {
		t.Fatalf("LoadBrunoTheme failed: %v", err)
	}

	if a.MethodColor("get") != ui.Hex("#34a853") {
		t.Fatal("get method color not loaded")
	}
	if a.MethodColor("post") != ui.Hex("#fbbc04") {
		t.Fatal("post method color not loaded")
	}
}