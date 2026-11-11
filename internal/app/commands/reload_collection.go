package commands

import (
	"context"
	"fmt"

	"fearless/internal/core/collection"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// ReloadCollection reads the collection tree from disk and reconciles it with in-memory state,
// preserving any uncommitted drafts, remapping renamed items, removing deleted items,
// and publishing a CollectionChanged event with Rescan: true.
//
// Idempotency: Reloading an already up-to-date collection produces consistent state without side-effects.
func (s *CollectionService) ReloadCollection(ctx context.Context, cmd ReloadCollectionCmd) error {
	if cmd.CollectionUID == "" {
		return fmt.Errorf("%w: collection UID is required", ErrInvalidInput)
	}

	snap := s.store.Snapshot()
	currentCol := snap.FindCollection(cmd.CollectionUID)
	if currentCol == nil {
		return fmt.Errorf("%w: %s", ErrCollectionNotFound, cmd.CollectionUID)
	}

	freshCol, err := collection.LoadCollection(currentCol.Path)
	if err != nil {
		return fmt.Errorf("commands: reload collection %s: %w", currentCol.Path, err)
	}

	// Reconcile in-memory state with disk read (preserve drafts, remap renames)
	reconciled := collection.Reconcile(currentCol, freshCol)

	// Update workspace state
	s.store.Apply(func(state *model.WorkspaceState) {
		for i, c := range state.Collections {
			if c.UID == cmd.CollectionUID {
				state.Collections[i] = reconciled
				break
			}
		}
	})

	// Publish CollectionChanged event (Self, Rescan)
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin: events.OriginSelf,
			Rescan: true,
		},
	})

	return nil
}
