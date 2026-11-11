package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestKeybindingRouter_DefaultsAndLookup(t *testing.T) {
	r := DefaultKeybindingRouter()
	if r == nil {
		t.Fatal("expected non-nil default keybinding router")
	}

	chord := r.GetChord(ActionSendRequest)
	if chord != "mod+enter" {
		t.Errorf("expected mod+enter for ActionSendRequest, got %q", chord)
	}

	bindings := r.Bindings()
	if len(bindings) == 0 {
		t.Fatal("expected registered bindings in default router")
	}

	var foundSave bool
	for _, b := range bindings {
		if b.ID == ActionSave {
			foundSave = true
			if !b.Valid {
				t.Error("expected ActionSave binding to be valid")
			}
			if b.DefaultChord != "mod+s" {
				t.Errorf("expected mod+s, got %s", b.DefaultChord)
			}
		}
	}
	if !foundSave {
		t.Error("expected ActionSave in registered bindings")
	}
}

func TestKeybindingRouter_SetChordAndReset(t *testing.T) {
	r := NewKeybindingRouter()
	r.Register("test.action", "Test Action", "mod+k", nil)

	if chord := r.GetChord("test.action"); chord != "mod+k" {
		t.Fatalf("expected mod+k, got %q", chord)
	}

	// Change chord
	if err := r.SetChord("test.action", "mod+shift+p"); err != nil {
		t.Fatalf("unexpected error setting chord: %v", err)
	}
	if chord := r.GetChord("test.action"); chord != "mod+shift+p" {
		t.Errorf("expected mod+shift+p, got %q", chord)
	}

	// Format chord
	formatted := r.FormatChord("test.action")
	if formatted == "" {
		t.Error("expected formatted chord to not be empty")
	}

	// Reset
	if err := r.ResetToDefault("test.action"); err != nil {
		t.Fatalf("unexpected error resetting chord: %v", err)
	}
	if chord := r.GetChord("test.action"); chord != "mod+k" {
		t.Errorf("expected reset to mod+k, got %q", chord)
	}

	// Invalid action
	if err := r.SetChord("non.existent", "mod+a"); err == nil {
		t.Error("expected error for unknown action")
	}
	if err := r.ResetToDefault("non.existent"); err == nil {
		t.Error("expected error for unknown action reset")
	}

	// Invalid syntax
	if err := r.SetChord("test.action", "invalid-chord-syntax???"); err == nil {
		t.Error("expected error for invalid chord syntax")
	}

	// Clear chord
	if err := r.SetChord("test.action", ""); err != nil {
		t.Errorf("clearing chord should succeed, got: %v", err)
	}
	if chord := r.GetChord("test.action"); chord != "" {
		t.Errorf("expected empty chord, got %q", chord)
	}
	if formatted := r.FormatChord("test.action"); formatted != "" {
		t.Errorf("expected empty formatted string for invalid chord, got %q", formatted)
	}
}

func TestKeybindingRouter_Conflicts(t *testing.T) {
	r := NewKeybindingRouter()
	r.Register("action.1", "Action 1", "mod+k", nil)
	r.Register("action.2", "Action 2", "mod+p", nil)

	// Check conflict with action.1
	confID, hasConflict := r.CheckConflict("action.2", "mod+k")
	if !hasConflict || confID != "action.1" {
		t.Errorf("expected conflict with action.1, got %s, %v", confID, hasConflict)
	}

	// Check no conflict when self
	_, hasConflict = r.CheckConflict("action.1", "mod+k")
	if hasConflict {
		t.Error("action should not conflict with itself")
	}

	// Check conflict list
	conflicts := r.Conflicts("mod+k", "action.3")
	if len(conflicts) != 1 || conflicts[0] != "action.1" {
		t.Errorf("expected [action.1], got %v", conflicts)
	}

	// Invalid syntax returns no conflict
	if _, hasConflict := r.CheckConflict("action.1", "???"); hasConflict {
		t.Error("invalid syntax should return false for conflict")
	}
	if confs := r.Conflicts("???", ""); len(confs) != 0 {
		t.Errorf("expected 0 conflicts for invalid chord, got %v", confs)
	}
}

func TestKeybindingRouter_Route(t *testing.T) {
	r := NewKeybindingRouter()
	var executed bool
	r.Register("test.run", "Run", "mod+r", func() {
		executed = true
	})

	// Nil context
	if r.Route(nil) {
		t.Error("Route(nil) should return false")
	}

	// Context with no matching shortcut
	tester := ui.NewTester(func(c *ui.Context) {
		handled := r.Route(c)
		if handled {
			t.Error("should not match without shortcut input")
		}
	}, 800, 600)
	tester.Frame()

	if executed {
		t.Error("handler should not have executed without shortcut press")
	}
}
