package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"fearless/internal/core/collection"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// MoveItem moves an item to a target folder (or root) and/or updates its sequence order.
//
// When moving across directories on disk, atomic rename with self-write suppression is used,
// the item UID is updated, and open editor tabs are remapped.
//
// Idempotency: Moving an item to its current location with its current sequence is a no-op.
func (s *CollectionService) MoveItem(ctx context.Context, cmd MoveItemCmd) error {
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

	// 1. Resolve destination directory
	destDir := col.Path
	if cmd.TargetFolderUID != "" {
		targetFolder := col.FindItem(cmd.TargetFolderUID)
		if targetFolder == nil || !targetFolder.IsFolder() {
			return fmt.Errorf("%w: target folder %s not found or not a folder", ErrItemNotFound, cmd.TargetFolderUID)
		}
		// Prevent moving folder into itself or its own subfolder
		if targetFolder.UID == item.UID || isDescendant(targetFolder, item) {
			return fmt.Errorf("%w: cannot move folder into itself or its descendants", ErrInvalidInput)
		}
		destDir = targetFolder.Path
	}

	currentDir := filepath.Dir(item.Path)
	movedDir := filepath.Clean(destDir) != filepath.Clean(currentDir)
	oldUID := item.UID
	newUID := oldUID

	// 2. Perform disk rename if moving across directories
	if movedDir {
		baseName := filepath.Base(item.Path)
		newPath := filepath.Join(destDir, baseName)

		collection.SuppressPath(item.Path)
		collection.SuppressPath(newPath)

		if err := os.Rename(item.Path, newPath); err != nil {
			return fmt.Errorf("commands: move item from %s to %s: %w", item.Path, newPath, err)
		}

		item.Path = newPath
		newUID = collection.GenerateUID(newPath)
		item.UID = newUID
	}

	// 3. Update sequence if requested
	if cmd.TargetSeq > 0 {
		seq := cmd.TargetSeq
		item.Seq = &seq
		if item.Request != nil {
			item.Request.Seq = &seq
		}
		_ = collection.SaveItem(col, item)
	}

	// 4. Update workspace state
	s.store.Apply(func(state *model.WorkspaceState) {
		targetCol := state.FindCollection(cmd.CollectionUID)
		if targetCol == nil {
			return
		}

		if movedDir {
			// Detach from previous parent
			targetCol.Items = removeItemTree(targetCol.Items, oldUID)

			// Attach to new parent
			if cmd.TargetFolderUID == "" {
				targetCol.Items = append(targetCol.Items, item)
				collection.SortItems(targetCol.Items)
			} else {
				parent := targetCol.FindItem(cmd.TargetFolderUID)
				if parent != nil {
					parent.Children = append(parent.Children, item)
					collection.SortItems(parent.Children)
				}
			}

			// Remap any open editor tabs that were viewing this item
			for _, tab := range state.Tabs {
				if tab.CollectionUID == cmd.CollectionUID && tab.ItemUID == oldUID {
					tab.ItemUID = newUID
				}
			}
		} else {
			// Sequence change only: re-sort folder
			if cmd.TargetFolderUID == "" {
				collection.SortItems(targetCol.Items)
			} else {
				parent := targetCol.FindItem(cmd.TargetFolderUID)
				if parent != nil {
					collection.SortItems(parent.Children)
				}
			}
		}
	})

	// 5. Publish CollectionChanged event (Self, Modified)
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin:   events.OriginSelf,
			Modified: []string{newUID},
		},
	})

	return nil
}

func isDescendant(candidate, root *model.Item) bool {
	if root == nil || candidate == nil {
		return false
	}
	for _, child := range root.Children {
		if child.UID == candidate.UID || isDescendant(candidate, child) {
			return true
		}
	}
	return false
}
