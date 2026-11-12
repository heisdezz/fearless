package presenters

import (
	"context"
	"testing"

	"fearless/internal/core/model"
)

// TestNeedsStoragePathIsTheFirstRunSignal: an empty StoragePath means the app
// has never been told where to save, which is exactly when the startup dialog
// must appear.
func TestNeedsStoragePathIsTheFirstRunSignal(t *testing.T) {
	if !NewPreferencesPresenter(model.DefaultSettings(), nil, nil, nil).NeedsStoragePath() {
		t.Error("a fresh install has no storage path, so NeedsStoragePath must be true")
	}

	configured := model.DefaultSettings()
	configured.General.StoragePath = "/home/user/collections"
	if NewPreferencesPresenter(configured, nil, nil, nil).NeedsStoragePath() {
		t.Error("a configured storage path must not trigger the dialog")
	}
}

// TestSetStoragePathPersistsImmediately: the first-run choice is written at
// once, not left in a draft, so the next launch does not ask again.
func TestSetStoragePathPersistsImmediately(t *testing.T) {
	saver := &fakePreferencesSaver{}
	p := NewPreferencesPresenter(model.DefaultSettings(), saver, nil, nil)

	if err := p.SetStoragePath(context.Background(), "/data/collections"); err != nil {
		t.Fatalf("SetStoragePath: %v", err)
	}

	if got := saver.savedSettings.General.StoragePath; got != "/data/collections" {
		t.Errorf("saved StoragePath = %q, want /data/collections", got)
	}
	if p.IsDirty() {
		t.Error("SetStoragePath must commit, not leave a draft")
	}
	if p.NeedsStoragePath() {
		t.Error("after SetStoragePath the app must not need one again")
	}
}
