package commands

import (
	"context"
	"fmt"

	"fearless/internal/core/collection"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// DeleteItem deletes an item from disk, removes it from the collection tree,
// closes any active editor tabs associated with it, and publishes a CollectionChanged event.
//
// Idempotency: Deleting an already deleted or non-existent item returns nil without error.
func (s *CollectionService) DeleteItem(ctx context.Context, cmd DeleteItemCmd) error {
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
		return nil // idempotent: item already deleted
	}

	// Collect all UIDs being deleted (including recursive children of a folder)
	removedUIDs := make(map[string]bool)
	collectUIDs(item, removedUIDs)

	// Delete from disk and from core collection tree
	if err := collection.DeleteItem(col, item); err != nil {
		return fmt.Errorf("commands: delete item %s from disk: %w", item.Name, err)
	}

	// Update workspace state: remove item and close open tabs for deleted items
	s.store.Apply(func(state *model.WorkspaceState) {
		targetCol := state.FindCollection(cmd.CollectionUID)
		if targetCol != nil {
			targetCol.Items = removeItemTree(targetCol.Items, cmd.ItemUID)
		}

		var remainingTabs []*model.Tab
		for _, t := range state.Tabs {
			if t.CollectionUID == cmd.CollectionUID && removedUIDs[t.ItemUID] {
				continue
			}
			remainingTabs = append(remainingTabs, t)
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

	uidsList := make([]string, 0, len(removedUIDs))
	for uid := range removedUIDs {
		uidsList = append(uidsList, uid)
	}

	// Publish CollectionChanged event (Self, Removed)
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin:  events.OriginSelf,
			Removed: uidsList,
		},
	})

	return nil
}

func collectUIDs(it *model.Item, uids map[string]bool) {
	if it == nil {
		return
	}
	uids[it.UID] = true
	for _, child := range it.Children {
		collectUIDs(child, uids)
	}
}

func removeItemTree(items []*model.Item, targetUID string) []*model.Item {
	var out []*model.Item
	for _, it := range items {
		if it.UID == targetUID {
			continue
		}
		if len(it.Children) > 0 {
			it.Children = removeItemTree(it.Children, targetUID)
		}
		out = append(out, it)
	}
	return out
}
