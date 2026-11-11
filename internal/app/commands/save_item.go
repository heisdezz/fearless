package commands

import (
	"context"
	"fmt"

	"fearless/internal/core/collection"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// SaveItem promotes in-progress draft edits to the saved request and writes the item
// to disk atomically using the appropriate dialect codec with self-write suppression.
//
// Idempotency: Saving an item that has no uncommitted draft (clean) is a safe no-op.
func (s *CollectionService) SaveItem(ctx context.Context, cmd SaveItemCmd) error {
	if cmd.CollectionUID == "" || cmd.ItemUID == "" {
		return fmt.Errorf("%w: collection UID and item UID are required", ErrInvalidInput)
	}

	snap := s.store.Snapshot()
	col := snap.FindCollection(cmd.CollectionUID)
	if col == nil {
		return fmt.Errorf("%w: %s", ErrCollectionNotFound, cmd.CollectionUID)
	}

	item := col.FindItem(cmd.ItemUID)
	if item == nil {
		return fmt.Errorf("%w: %s", ErrItemNotFound, cmd.ItemUID)
	}

	// Idempotency: clean item with no draft does not need disk rewrite
	if !item.Dirty() && item.Request != nil {
		return nil
	}

	// Save to disk using core collection logic (promotes Draft -> Request, clears Draft)
	if err := collection.SaveItem(col, item); err != nil {
		return fmt.Errorf("commands: save item %s: %w", item.Name, err)
	}

	// Update in-memory workspace store
	s.store.Apply(func(state *model.WorkspaceState) {
		targetCol := state.FindCollection(cmd.CollectionUID)
		if targetCol == nil {
			return
		}
		targetItem := targetCol.FindItem(cmd.ItemUID)
		if targetItem == nil {
			return
		}
		if targetItem.Draft != nil {
			targetItem.Request = targetItem.Draft
			targetItem.Draft = nil
		}
	})

	// Publish CollectionChanged event (Self, Modified)
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin:   events.OriginSelf,
			Modified: []string{cmd.ItemUID},
		},
	})

	return nil
}
