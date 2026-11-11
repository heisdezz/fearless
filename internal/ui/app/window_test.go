package app

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestApp_CreateWindowOptions(t *testing.T) {
	a := New()

	// 1. Default options
	def := a.CreateWindowOptions()
	if def.Title != "fearless" {
		t.Errorf("expected default title 'fearless', got %q", def.Title)
	}
	if def.Width != 1280 || def.Height != 800 {
		t.Errorf("expected 1280x800, got %dx%d", def.Width, def.Height)
	}
	if def.StateKey != "fearless.main" {
		t.Errorf("expected fearless.main, got %q", def.StateKey)
	}
	if def.Content == nil {
		t.Error("expected non-nil Content view")
	}

	// 2. Custom override options
	customContent := ui.View(func(c *ui.Context) {
		ui.Text(c, "custom")
	})

	customOpts := mygo.WindowOptions{
		Title:    "Custom Fearless",
		Width:    1024,
		Height:   768,
		StateKey: "custom.key",
		Content:  customContent,
	}

	opts := a.CreateWindowOptions(customOpts)
	if opts.Title != "Custom Fearless" {
		t.Errorf("expected 'Custom Fearless', got %q", opts.Title)
	}
	if opts.Width != 1024 || opts.Height != 768 {
		t.Errorf("expected 1024x768, got %dx%d", opts.Width, opts.Height)
	}
	if opts.StateKey != "custom.key" {
		t.Errorf("expected custom.key, got %q", opts.StateKey)
	}
	if opts.Content == nil {
		t.Error("expected custom content")
	}
}
