package app

import (
	"os"
	"path/filepath"
	"testing"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

// TestSessionRoundTrip: what is saved is what is read back.
func TestSessionRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	want := sessionState{Collections: []string{"/a/one", "/b/two"}, Active: "/b/two"}
	if err := saveSession(path, want); err != nil {
		t.Fatalf("saveSession: %v", err)
	}
	got := loadSession(path)
	if len(got.Collections) != 2 || got.Collections[0] != "/a/one" || got.Active != "/b/two" {
		t.Fatalf("loadSession = %+v, want %+v", got, want)
	}
}

// TestMissingSessionIsEmpty: a first run has no session file, and that must not
// be an error.
func TestMissingSessionIsEmpty(t *testing.T) {
	got := loadSession(filepath.Join(t.TempDir(), "nope.json"))
	if len(got.Collections) != 0 {
		t.Fatalf("expected an empty session, got %+v", got)
	}
}

// TestCorruptSessionIsEmpty: a truncated file costs the layout, not a crash.
func TestCorruptSessionIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadSession(path); len(got.Collections) != 0 {
		t.Fatalf("expected an empty session from corrupt data, got %+v", got)
	}
}

// TestPersistSessionRecordsOpenCollections: saving reads what the presenter
// holds, so a collection opened this run reopens next run.
func TestPersistSessionRecordsOpenCollections(t *testing.T) {
	a := New()
	a.sessionPath = filepath.Join(t.TempDir(), "session.json")
	a.CollectionPresenter.SetCollections([]*model.Collection{
		{UID: "u1", Name: "One", Path: "/tmp/one"},
		{UID: "u2", Name: "Two", Path: "/tmp/two"},
	})

	a.persistSession()

	got := loadSession(a.sessionPath)
	if len(got.Collections) != 2 {
		t.Fatalf("recorded %d collections, want 2", len(got.Collections))
	}
	if got.Collections[0] != "/tmp/one" || got.Collections[1] != "/tmp/two" {
		t.Errorf("paths = %v", got.Collections)
	}
}

// TestStoragePathComesFromPreferences: the import copy target is the folder the
// user chose at first run.
func TestStoragePathComesFromPreferences(t *testing.T) {
	a := New()
	if got := a.storagePath(); got != "" {
		t.Fatalf("storagePath = %q, want empty before first run", got)
	}

	s := model.DefaultSettings()
	s.General.StoragePath = "/data/collections"
	a.PreferencesPresenter = presenters.NewPreferencesPresenter(s, nil, nil, nil)
	if got := a.storagePath(); got != "/data/collections" {
		t.Fatalf("storagePath = %q, want /data/collections", got)
	}
}

// TestImportWithoutStorageStaysInMemory: before a storage folder is chosen the
// import still works for this run; it is the save that is impossible.
func TestImportWithoutStorageStaysInMemory(t *testing.T) {
	a := New()
	dir, err := a.storeImportedCollection(&model.Collection{Name: "X"})
	if err != nil {
		t.Fatalf("storeImportedCollection: %v", err)
	}
	if dir != "" {
		t.Errorf("expected no directory without storage, got %q", dir)
	}
}

// TestPersistSessionRecordsActiveCollection: the selected collection is
// remembered by path, so the same collection is focused after a restart.
// Selecting a request inside a collection counts; the UID alone would not
// survive, since UIDs are regenerated on load.
func TestPersistSessionRecordsActiveCollection(t *testing.T) {
	a := New()
	a.sessionPath = filepath.Join(t.TempDir(), "session.json")
	one := &model.Collection{UID: "u1", Name: "One", Path: "/tmp/one"}
	two := &model.Collection{UID: "u2", Name: "Two", Path: "/tmp/two"}
	two.Items = []*model.Item{{UID: "req-1", Name: "Login", Request: &model.Request{}}}
	a.CollectionPresenter.SetCollections([]*model.Collection{one, two})
	a.CollectionPresenter.Select("req-1")

	a.persistSession()

	if got := loadSession(a.sessionPath).Active; got != "/tmp/two" {
		t.Fatalf("Active = %q, want /tmp/two", got)
	}
}

// TestActiveCollectionPathEmptyWithoutSelection: nothing selected means no
// remembered collection, not a guess at the first one.
func TestActiveCollectionPathEmptyWithoutSelection(t *testing.T) {
	a := New()
	cols := []*model.Collection{{UID: "u1", Path: "/tmp/one"}}
	a.CollectionPresenter.SetCollections(cols)
	if got := a.activeCollectionPath(cols); got != "" {
		t.Fatalf("activeCollectionPath = %q, want empty", got)
	}
}
