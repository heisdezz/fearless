package app

import (
	"fmt"
	"strings"

	"fearless/internal/core/collection"
	"fearless/internal/core/model"
)

// storagePath is the folder collections are saved in, from preferences.
func (a *App) storagePath() string {
	if a.PreferencesPresenter == nil {
		return ""
	}
	return strings.TrimSpace(a.PreferencesPresenter.Settings().General.StoragePath)
}

// collectionForUID finds the open collection that contains uid. uid may be the
// collection itself, a folder or a request: selecting any of them selects its
// collection, which is what the environment screen and the environment
// selector read.
func (a *App) collectionForUID(uid string) *model.Collection {
	if a.CollectionPresenter == nil || uid == "" {
		return nil
	}
	cols := a.CollectionPresenter.Collections()
	for _, c := range cols {
		if c == nil {
			continue
		}
		if c.UID == uid || c.FindItem(uid) != nil {
			return c
		}
	}
	// No match means the uid belongs to something else (a tab, a stale
	// selection). Fall back to the collection that is open, because an
	// environment screen with no collection cannot do anything at all.
	if len(cols) > 0 {
		return cols[0]
	}
	return nil
}

// bindEnvironment points the environment presenter at the collection that owns
// uid. Environments, the active environment and the selector all read from the
// bound collection, so without this the editor shows "No collection selected"
// even though a collection is open.
func (a *App) bindEnvironment(uid string) {
	if a.EnvironmentPresenter == nil {
		return
	}
	col := a.collectionForUID(uid)
	if col == nil {
		return
	}
	if a.EnvironmentPresenter.Collection() == col {
		return
	}
	a.EnvironmentPresenter.SetCollection(col)
}

// storeImportedCollection writes an imported collection into the storage
// folder and returns the directory it now lives in.
//
// An import arrives either as a directory on disk or as a parsed model with no
// files (cURL, Postman, OpenAPI). Both are copied in, so reopening works from a
// single known root and the collection belongs to the user rather than to
// wherever it came from.
//
// With no storage folder configured it returns "" and no error: the caller
// keeps the in-memory collection, which is the best available answer before the
// first-run prompt is answered.
func (a *App) storeImportedCollection(col *model.Collection) (string, error) {
	if col == nil {
		return "", nil
	}
	root := a.storagePath()
	if root == "" {
		return "", nil
	}

	var (
		dst string
		err error
	)
	if col.Path != "" {
		dst, err = collection.CopyIntoStorage(col.Path, root)
	} else {
		dst, err = collection.CopyModelIntoStorage(col, root)
	}
	if err != nil {
		return "", fmt.Errorf("import: %w", err)
	}
	return dst, nil
}

// persistSession records the open collections so the next launch can restore
// them. It reads them from the presenter, the single source of truth for what
// is open.
func (a *App) persistSession() {
	if a.CollectionPresenter == nil {
		return
	}
	cols := a.CollectionPresenter.Collections()
	s := sessionState{}
	for _, c := range cols {
		if c != nil && c.Path != "" {
			s.Collections = append(s.Collections, c.Path)
		}
	}
	if active := a.activeCollectionPath(cols); active != "" {
		s.Active = active
	}
	// A failed save costs the next launch's layout, not the user's work.
	_ = saveSession(a.sessionPath, s)
}

// activeCollectionPath reports the path of the collection containing the
// current selection, or "". The selection may be a request or a folder, so it
// is looked up rather than compared against collection UIDs.
func (a *App) activeCollectionPath(cols []*model.Collection) string {
	if a.CollectionPresenter == nil || len(cols) == 0 {
		return ""
	}
	uid := a.CollectionPresenter.SelectedUID()
	if uid == "" {
		return ""
	}
	for _, c := range cols {
		if c == nil {
			continue
		}
		if c.UID == uid {
			return c.Path
		}
		if c.FindItem(uid) != nil {
			return c.Path
		}
	}
	return ""
}

// collectionByPath finds an open collection by its directory.
func (a *App) collectionByPath(path string) *model.Collection {
	if a.CollectionPresenter == nil {
		return nil
	}
	for _, c := range a.CollectionPresenter.Collections() {
		if c != nil && c.Path == path {
			return c
		}
	}
	return nil
}

// restoreSession reopens the collections recorded by the last run.
//
// A path that no longer exists is skipped rather than reported: a collection on
// a removable drive, or one the user deleted, should not block startup. It runs
// before the window exists, so SafeUpdate executes synchronously and the
// presenter is populated by the time this returns; the session file is
// rewritten at the end, which drops the stale paths.
func (a *App) restoreSession() {
	s := loadSession(a.sessionPath)
	if len(s.Collections) == 0 {
		return
	}
	for _, path := range s.Collections {
		a.loadCollectionDir(path)
	}
	if s.Active != "" {
		if col := a.collectionByPath(s.Active); col != nil {
			a.CollectionPresenter.Select(col.UID)
		}
	}
	// Rewrite the session so paths that no longer load drop out. This runs
	// before the window exists, so the loads above already executed and the
	// presenter holds exactly what reopened.
	a.persistSession()
}
