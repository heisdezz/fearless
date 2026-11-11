package layout

import (
	"testing"
)

func TestDefaultRegionBudget(t *testing.T) {
	b := DefaultRegionBudget()
	
	if b.TitleBar != 32 {
		t.Errorf("TitleBar = %v, want 32", b.TitleBar)
	}
	if b.SidebarMin != 220 {
		t.Errorf("SidebarMin = %v, want 220", b.SidebarMin)
	}
	if b.SidebarMax != 600 {
		t.Errorf("SidebarMax = %v, want 600", b.SidebarMax)
	}
	if b.SidebarCollapsed != 64 {
		t.Errorf("SidebarCollapsed = %v, want 64", b.SidebarCollapsed)
	}
	if b.CollectionHeader != 40 {
		t.Errorf("CollectionHeader = %v, want 40", b.CollectionHeader)
	}
	if b.StatusBar != 24 {
		t.Errorf("StatusBar = %v, want 24", b.StatusBar)
	}
	if b.ConsoleDock != 300 {
		t.Errorf("ConsoleDock = %v, want 300", b.ConsoleDock)
	}
	if b.MinWindowWidth != 900 {
		t.Errorf("MinWindowWidth = %v, want 900", b.MinWindowWidth)
	}
	if b.MinWindowHeight != 600 {
		t.Errorf("MinWindowHeight = %v, want 600", b.MinWindowHeight)
	}
}

func TestDefaultLayoutState(t *testing.T) {
	s := DefaultLayoutState()
	
	if s.SidebarWidth != 280 {
		t.Errorf("SidebarWidth = %v, want 280", s.SidebarWidth)
	}
	if !s.SidebarExpanded {
		t.Error("SidebarExpanded should be true")
	}
	if s.RequestResponseOrientation != OrientationStacked {
		t.Errorf("RequestResponseOrientation = %v, want OrientationStacked", s.RequestResponseOrientation)
	}
	if s.ConsoleOpen {
		t.Error("ConsoleOpen should be false")
	}
	if s.SplitRatio != 0.55 {
		t.Errorf("SplitRatio = %v, want 0.55", s.SplitRatio)
	}
	if s.WindowWidth != 1200 {
		t.Errorf("WindowWidth = %v, want 1200", s.WindowWidth)
	}
	if s.WindowHeight != 800 {
		t.Errorf("WindowHeight = %v, want 800", s.WindowHeight)
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		n       int
		singular string
		want    string
	}{
		{1, "request", "1 request"},
		{0, "request", "0 requests"},
		{5, "item", "0 items"},
		{1, "selected", "1 selected"},
	}
	
	for _, tc := range tests {
		got := formatCount(tc.n, tc.singular)
		if got != tc.want {
			t.Errorf("formatCount(%d, %q) = %q, want %q", tc.n, tc.singular, got, tc.want)
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