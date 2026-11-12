package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

func settingsWithStorage() model.Settings {
	s := model.DefaultSettings()
	s.General.StoragePath = "/home/user/collections"
	return s
}

// recordingSaver captures what the app persists.
type recordingSaver struct {
	saved model.Settings
}

func (r *recordingSaver) SavePreferences(_ context.Context, s model.Settings) error {
	r.saved = s
	return nil
}

// eventually polls until cond holds or the deadline passes.
func eventually(cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

// TestFirstRunShowsStoragePrompt: with no storage folder configured the app
// must open the required dialog before anything else.
func TestFirstRunShowsStoragePrompt(t *testing.T) {
	a := New()

	a.promptStorageLocationIfNeeded()

	if !a.DialogQueue.Active() {
		t.Fatal("first run must open the storage dialog")
	}
	if got := a.DialogQueue.Kind().String(); got != "storage-location" {
		t.Fatalf("kind = %v, want storage-location", got)
	}
	if !a.DialogQueue.Required() {
		t.Error("the storage dialog must not be cancellable")
	}

	// It must be on screen, not merely queued.
	tt := ui.NewTester(a.RootView, 1000, 700)
	tt.Frame()
	if !tt.HasText("Choose where to save your work") {
		t.Errorf("storage dialog not rendered; texts: %v", tt.Texts())
	}
}

// TestNoStoragePromptWhenConfigured: a saved folder means the app opens
// straight into the workspace.
func TestNoStoragePromptWhenConfigured(t *testing.T) {
	a := New()
	a.PreferencesPresenter = presenters.NewPreferencesPresenter(settingsWithStorage(), nil, nil, nil)

	a.promptStorageLocationIfNeeded()

	if a.DialogQueue.Active() {
		t.Fatalf("a configured app must not ask again (kind %v)", a.DialogQueue.Kind())
	}
}

// TestStorageChoiceIsRemembered: answering the dialog writes the folder, so the
// next launch does not ask.
func TestStorageChoiceIsRemembered(t *testing.T) {
	saver := &recordingSaver{}
	a := New()
	a.PreferencesPresenter = presenters.NewPreferencesPresenter(model.DefaultSettings(), saver, nil, nil)

	a.promptStorageLocationIfNeeded()
	chosen := filepath.Join(t.TempDir(), "collections")
	a.DialogQueue.SetValue("location", chosen)
	a.DialogQueue.OK()

	if !eventually(func() bool { return !a.DialogQueue.Active() }) {
		t.Fatalf("dialog did not close after a valid folder (phase %v)", a.DialogQueue.Phase())
	}
	if got := saver.saved.General.StoragePath; got != chosen {
		t.Errorf("saved StoragePath = %q, want %q", got, chosen)
	}
	if a.PreferencesPresenter.NeedsStoragePath() {
		t.Error("after answering, the app must not need a storage path")
	}
}

// TestStorageChoiceRejectsUnusableFolder: a path that cannot be created keeps
// the dialog open with an explanation rather than silently failing.
func TestStorageChoiceRejectsUnusableFolder(t *testing.T) {
	saver := &recordingSaver{}
	a := New()
	a.PreferencesPresenter = presenters.NewPreferencesPresenter(model.DefaultSettings(), saver, nil, nil)

	a.promptStorageLocationIfNeeded()

	// A path under a regular file cannot be created as a directory.
	dir := t.TempDir()
	file := filepath.Join(dir, "occupied")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	a.DialogQueue.SetValue("location", filepath.Join(file, "child"))
	a.DialogQueue.OK()

	if !eventually(func() bool { return a.DialogQueue.Phase().String() == "failed" }) {
		t.Fatalf("phase = %v, want failed", a.DialogQueue.Phase())
	}
	if !strings.Contains(a.DialogQueue.Message(), "cannot use that folder") {
		t.Errorf("message = %q, want it to explain the folder was unusable", a.DialogQueue.Message())
	}
	if saver.saved.General.StoragePath != "" {
		t.Error("a rejected folder must not be recorded")
	}
}
