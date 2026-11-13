package app

import (
	"path/filepath"
	"testing"

	"fearless/internal/core/collection"
	"fearless/internal/core/model"
)

// TestLoadCollectionDirPersistsImmediately guards the ordering bug: the session
// used to be written before SafeUpdate's callback ran, so opening a collection
// with a live window recorded an empty session.
func TestLoadCollectionDirPersistsImmediately(t *testing.T) {
	src := &model.Collection{
		Name:   "Persisted",
		Config: model.CollectionConfig{Format: model.FormatBru},
		Items: []*model.Item{{
			UID: "r1", Name: "R", Type: model.ItemHTTPRequest,
			Request: &model.Request{Method: "GET", URL: "https://example.com"},
		}},
	}
	dir, err := collection.CopyModelIntoStorage(src, t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	a := New()
	a.sessionPath = filepath.Join(t.TempDir(), "session.json")

	a.loadCollectionDir(dir)

	got := loadSession(a.sessionPath)
	if len(got.Collections) != 1 || got.Collections[0] != dir {
		t.Fatalf("session = %+v, want the opened collection %q", got, dir)
	}
}
