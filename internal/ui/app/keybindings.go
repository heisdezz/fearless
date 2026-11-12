// Package app provides window creation, application menu bar, keybinding
// routing, and root view composition over the fearless GUI screen components.
package app

import (
	"fmt"
	"sort"
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
)

// Standard action identifiers for fearless keybindings.
const (
	ActionSendRequest   = "request.send"
	ActionCancelRequest = "request.cancel"
	ActionSave          = "file.save"
	ActionNewRequest    = "file.new_request"
	ActionNewCollection = "file.new_collection"
	ActionImport        = "file.import"
	ActionCloseTab      = "tab.close"
	ActionNextTab       = "tab.next"
	ActionPrevTab       = "tab.prev"
	ActionSelectTab1    = "tab.select_1"
	ActionSelectTab2    = "tab.select_2"
	ActionSelectTab3    = "tab.select_3"
	ActionSelectTab4    = "tab.select_4"
	ActionSelectTab5    = "tab.select_5"
	ActionSelectTab6    = "tab.select_6"
	ActionSelectTab7    = "tab.select_7"
	ActionSelectTab8    = "tab.select_8"
	ActionSearch        = "edit.search"
	ActionGlobalSearch  = "edit.global_search"
	ActionPreferences   = "app.preferences"
	ActionToggleSidebar = "view.toggle_sidebar"
	ActionChangeLayout  = "view.change_layout"
	ActionToggleConsole = "view.toggle_console"
	ActionZoomIn        = "view.zoom_in"
	ActionZoomOut       = "view.zoom_out"
	ActionResetZoom     = "view.reset_zoom"
)

// Binding describes one registered keyboard shortcut.
type Binding struct {
	ID           string
	Label        string
	DefaultChord string
	Chord        string
	Modifiers    ui.Modifiers
	Key          ui.Key
	Valid        bool
	Handler      func()
}

// KeybindingRouter routes keyboard shortcuts using display.HotkeyBinding and
// Context.Shortcut, maintaining customizable key mappings.
type KeybindingRouter struct {
	mu       sync.RWMutex
	bindings map[string]*Binding
	order    []string
}

// NewKeybindingRouter creates an empty router.
func NewKeybindingRouter() *KeybindingRouter {
	return &KeybindingRouter{
		bindings: make(map[string]*Binding),
	}
}

// Register adds a new shortcut binding with its default chord and handler.
func (r *KeybindingRouter) Register(id, label, defaultChord string, handler func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	b := &Binding{
		ID:           id,
		Label:        label,
		DefaultChord: defaultChord,
		Chord:        defaultChord,
		Handler:      handler,
	}

	mods, key, ok := display.HotkeyBinding(defaultChord, display.HotkeyAuto)
	if ok {
		b.Modifiers = mods
		b.Key = key
		b.Valid = true
	}

	if _, exists := r.bindings[id]; !exists {
		r.order = append(r.order, id)
	}
	r.bindings[id] = b
}

// SetHandler updates the execution handler for an existing binding.
func (r *KeybindingRouter) SetHandler(id string, handler func()) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if b, ok := r.bindings[id]; ok {
		b.Handler = handler
	}
}

// SetChord updates the chord for an action. Returns an error if the chord cannot be parsed.
func (r *KeybindingRouter) SetChord(id, chord string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.bindings[id]
	if !ok {
		return fmt.Errorf("unknown action: %s", id)
	}

	if chord == "" {
		b.Chord = ""
		b.Valid = false
		return nil
	}

	mods, key, valid := display.HotkeyBinding(chord, display.HotkeyAuto)
	if !valid {
		return fmt.Errorf("invalid hotkey syntax: %s", chord)
	}

	b.Chord = chord
	b.Modifiers = mods
	b.Key = key
	b.Valid = true
	return nil
}

// ResetToDefault restores an action to its default chord.
func (r *KeybindingRouter) ResetToDefault(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.bindings[id]
	if !ok {
		return fmt.Errorf("unknown action: %s", id)
	}
	return r.setChordLocked(b, b.DefaultChord)
}

func (r *KeybindingRouter) setChordLocked(b *Binding, chord string) error {
	if chord == "" {
		b.Chord = ""
		b.Valid = false
		return nil
	}
	mods, key, ok := display.HotkeyBinding(chord, display.HotkeyAuto)
	if !ok {
		return fmt.Errorf("invalid hotkey syntax: %s", chord)
	}
	b.Chord = chord
	b.Modifiers = mods
	b.Key = key
	b.Valid = true
	return nil
}

// GetChord returns the configured chord for an action.
func (r *KeybindingRouter) GetChord(id string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if b, ok := r.bindings[id]; ok {
		return b.Chord
	}
	return ""
}

// FormatChord formats the hotkey binding for display using display.FormatHotkey.
func (r *KeybindingRouter) FormatChord(id string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.bindings[id]
	if !ok || !b.Valid {
		return ""
	}

	formatted, ok := display.FormatHotkey(b.Modifiers, b.Key, display.HotkeyAuto)
	if ok {
		return formatted
	}
	return b.Chord
}

// CheckConflict returns whether chord conflicts with any other action's binding.
func (r *KeybindingRouter) CheckConflict(id, chord string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mods, key, ok := display.HotkeyBinding(chord, display.HotkeyAuto)
	if !ok {
		return "", false
	}

	for otherID, b := range r.bindings {
		if otherID == id || !b.Valid {
			continue
		}
		if b.Modifiers == mods && b.Key == key {
			return otherID, true
		}
	}
	return "", false
}

// Conflicts returns all existing action IDs that conflict with the given candidate chord.
func (r *KeybindingRouter) Conflicts(chord string, exceptID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mods, key, ok := display.HotkeyBinding(chord, display.HotkeyAuto)
	if !ok {
		return nil
	}

	var conflicts []string
	for id, b := range r.bindings {
		if id == exceptID || !b.Valid {
			continue
		}
		if b.Modifiers == mods && b.Key == key {
			conflicts = append(conflicts, id)
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

// Bindings returns a snapshot of all registered bindings in registration order.
func (r *KeybindingRouter) Bindings() []Binding {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]Binding, 0, len(r.order))
	for _, id := range r.order {
		if b, ok := r.bindings[id]; ok {
			res = append(res, *b)
		}
	}
	return res
}

// Route checks Context.Shortcut for each valid binding and invokes the matching handler.
// Returns true if a shortcut was triggered.
func (r *KeybindingRouter) Route(c *ui.Context) bool {
	if c == nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, id := range r.order {
		b := r.bindings[id]
		if b == nil || !b.Valid || b.Handler == nil {
			continue
		}

		if c.Shortcut(b.Modifiers, b.Key) {
			b.Handler()
			return true
		}
	}

	return false
}

// DefaultKeybindingRouter creates a router pre-populated with default Bruno/fearless bindings.
func DefaultKeybindingRouter() *KeybindingRouter {
	r := NewKeybindingRouter()

	// Requests
	r.Register(ActionSendRequest, "Send Request", "mod+enter", nil)
	r.Register(ActionCancelRequest, "Cancel Request", "escape", nil)

	// File / Collection
	r.Register(ActionSave, "Save", "mod+s", nil)
	r.Register(ActionNewRequest, "New Request", "mod+n", nil)
	r.Register(ActionNewCollection, "New Collection", "mod+shift+n", nil)
	r.Register(ActionImport, "Import Collection", "mod+o", nil)

	// Tabs
	r.Register(ActionCloseTab, "Close Tab", "mod+w", nil)
	r.Register(ActionNextTab, "Next Tab", "mod+shift+]", nil)
	r.Register(ActionPrevTab, "Previous Tab", "mod+shift+[", nil)
	r.Register(ActionSelectTab1, "Select Tab 1", "mod+1", nil)
	r.Register(ActionSelectTab2, "Select Tab 2", "mod+2", nil)
	r.Register(ActionSelectTab3, "Select Tab 3", "mod+3", nil)
	r.Register(ActionSelectTab4, "Select Tab 4", "mod+4", nil)
	r.Register(ActionSelectTab5, "Select Tab 5", "mod+5", nil)
	r.Register(ActionSelectTab6, "Select Tab 6", "mod+6", nil)
	r.Register(ActionSelectTab7, "Select Tab 7", "mod+7", nil)
	r.Register(ActionSelectTab8, "Select Tab 8", "mod+8", nil)

	// Search
	r.Register(ActionSearch, "Search", "mod+f", nil)
	r.Register(ActionGlobalSearch, "Global Search", "mod+k", nil)

	// View
	r.Register(ActionToggleSidebar, "Toggle Sidebar", "mod+\\", nil)
	r.Register(ActionChangeLayout, "Change Orientation", "mod+j", nil)
	r.Register(ActionToggleConsole, "Toggle Console", "mod+t", nil)
	r.Register(ActionZoomIn, "Zoom In", "mod+=", nil)
	r.Register(ActionZoomOut, "Zoom Out", "mod+-", nil)
	r.Register(ActionResetZoom, "Reset Zoom", "mod+0", nil)

	// Preferences
	r.Register(ActionPreferences, "Preferences", "mod+,", nil)

	return r
}
