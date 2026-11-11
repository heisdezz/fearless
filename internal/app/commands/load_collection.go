package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"fearless/internal/core/collection"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// LoadCollection loads a collection from the given directory path, attaches a file watcher,
// stores it in workspace state, and publishes a CollectionChanged event.
//
// Idempotency: Calling LoadCollection on an already opened directory path returns the existing
// collection's UID without reloading or re-attaching watchers.
func (s *CollectionService) LoadCollection(ctx context.Context, cmd LoadCollectionCmd) (string, error) {
	if cmd.Path == "" {
		return "", fmt.Errorf("%w: path is required", ErrInvalidInput)
	}

	absPath, err := filepath.Abs(cmd.Path)
	if err != nil {
		return "", fmt.Errorf("commands: resolve path %s: %w", cmd.Path, err)
	}

	// 1. Idempotency check: see if already opened in state
	snap := s.store.Snapshot()
	for _, c := range snap.Collections {
		if filepath.Clean(c.Path) == filepath.Clean(absPath) {
			return c.UID, nil
		}
	}

	// 2. Load collection tree from disk
	col, err := collection.LoadCollection(absPath)
	if err != nil {
		return "", fmt.Errorf("commands: load collection from %s: %w", absPath, err)
	}

	// 3. Attach file watcher for external edits
	s.mu.Lock()
	if existingWatcher, ok := s.watchers[col.UID]; ok {
		_ = existingWatcher.Close()
	}

	watcher, err := collection.NewWatcher(col, collection.WatcherOptions{
		DebounceDelay: 50 * time.Millisecond,
		OnChange: func(reloaded *model.Collection, err error) {
			if err != nil || reloaded == nil {
				return
			}
			// Update state store with external reload
			s.store.Apply(func(state *model.WorkspaceState) {
				for i, existing := range state.Collections {
					if existing.UID == reloaded.UID {
						state.Collections[i] = reloaded
						break
					}
				}
			})
			// Publish external change event
			s.publish(events.CollectionChanged{
				CollectionUID: reloaded.UID,
				Change: events.TreeChange{
					Origin: events.OriginExternal,
					Rescan: true,
				},
			})
		},
	})
	if err == nil {
		s.watchers[col.UID] = watcher
	}
	s.mu.Unlock()

	// 4. Update workspace state
	s.store.Apply(func(state *model.WorkspaceState) {
		found := false
		for i, c := range state.Collections {
			if c.UID == col.UID {
				state.Collections[i] = col
				found = true
				break
			}
		}
		if !found {
			state.Collections = append(state.Collections, col)
		}
	})

	// 5. Publish CollectionChanged event (Self, Added)
	s.publish(events.CollectionChanged{
		CollectionUID: col.UID,
		Change: events.TreeChange{
			Origin: events.OriginSelf,
			Added:  []string{col.UID},
		},
	})

	return col.UID, nil
}

// CloseCollection stops file watching, removes the collection from workspace state,
// closes any tabs belonging to this collection, and publishes a CollectionChanged event.
//
// Idempotency: Closing a collection that is not open returns nil without error.
func (s *CollectionService) CloseCollection(ctx context.Context, cmd CloseCollectionCmd) error {
	if cmd.CollectionUID == "" {
		return fmt.Errorf("%w: collection UID is required", ErrInvalidInput)
	}

	snap := s.store.Snapshot()
	col := snap.FindCollection(cmd.CollectionUID)
	if col == nil {
		return nil // idempotent
	}

	// 1. Close and deregister watcher
	s.mu.Lock()
	if w, ok := s.watchers[cmd.CollectionUID]; ok {
		_ = w.Close()
		delete(s.watchers, cmd.CollectionUID)
	}
	s.mu.Unlock()

	// 2. Remove collection and close associated tabs
	s.store.Apply(func(state *model.WorkspaceState) {
		var remainingCols []*model.Collection
		for _, c := range state.Collections {
			if c.UID != cmd.CollectionUID {
				remainingCols = append(remainingCols, c)
			}
		}
		state.Collections = remainingCols

		var remainingTabs []*model.Tab
		for _, t := range state.Tabs {
			if t.CollectionUID != cmd.CollectionUID {
				remainingTabs = append(remainingTabs, t)
			}
		}
		state.Tabs = remainingTabs

		if state.ActiveTab != "" && state.FindTab(state.ActiveTab) == nil {
			if len(state.Tabs) > 0 {
				state.ActiveTab = state.Tabs[len(state.Tabs)-1].UID
			} else {
				state.ActiveTab = ""
			}
		}
	})

	// 3. Publish CollectionChanged event (Self, Removed)
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin:  events.OriginSelf,
			Removed: []string{cmd.CollectionUID},
		},
	})

	return nil
}
