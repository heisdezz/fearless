package collection

import (
	"os"
	"path/filepath"

	"fearless/internal/core/model"
)

// Reconcile merges a freshly loaded collection from disk into the current in-memory collection:
//  1. In-progress drafts (item.Draft != nil) are preserved across external reloads.
//  2. Deleted files on disk are removed from the item tree.
//  3. Renamed files remap their UIDs and retain existing drafts.
//  4. Active environment selection is maintained.
func Reconcile(current *model.Collection, incoming *model.Collection) *model.Collection {
	if current == nil {
		return incoming
	}
	if incoming == nil {
		return current
	}

	// Index existing items
	existingByUID := make(map[string]*model.Item)
	existingByPath := make(map[string]*model.Item)
	var deletedDraftItems []*model.Item

	_ = current.Walk(func(it *model.Item) error {
		existingByUID[it.UID] = it
		if it.Path != "" {
			existingByPath[filepath.Clean(it.Path)] = it
		}
		// If file on disk no longer exists, record item in case it was renamed
		if it.Path != "" {
			if _, err := os.Stat(it.Path); os.IsNotExist(err) {
				deletedDraftItems = append(deletedDraftItems, it)
			}
		}
		return nil
	})

	// Reconcile each item in the incoming collection
	_ = incoming.Walk(func(incomingItem *model.Item) error {
		var matched *model.Item

		// 1. Match by UID
		if it, ok := existingByUID[incomingItem.UID]; ok {
			matched = it
		}

		// 2. Match by Path
		if matched == nil && incomingItem.Path != "" {
			if it, ok := existingByPath[filepath.Clean(incomingItem.Path)]; ok {
				matched = it
			}
		}

		// 3. Match renamed items (check if an existing deleted item has the same name or draft)
		if matched == nil && len(deletedDraftItems) > 0 {
			for idx, delItem := range deletedDraftItems {
				// Same folder directory and name, or matching draft
				if filepath.Dir(delItem.Path) == filepath.Dir(incomingItem.Path) &&
					(delItem.Name == incomingItem.Name || delItem.Draft != nil) {
					matched = delItem
					// Remove from deleted list so it's not matched twice
					deletedDraftItems = append(deletedDraftItems[:idx], deletedDraftItems[idx+1:]...)
					break
				}
			}
		}

		// Apply preserved draft and examples if matched
		if matched != nil {
			if matched.Draft != nil {
				incomingItem.Draft = matched.Draft
			}
			if len(matched.Examples) > 0 && len(incomingItem.Examples) == 0 {
				incomingItem.Examples = matched.Examples
			}
		}

		return nil
	})

	// Maintain active environment selection
	if incoming.ActiveEnvironment == "" && current.ActiveEnvironment != "" {
		incoming.ActiveEnvironment = current.ActiveEnvironment
	}

	// Update dirty state if any draft exists
	hasDraft := false
	_ = incoming.Walk(func(it *model.Item) error {
		if it.Draft != nil {
			hasDraft = true
		}
		return nil
	})
	incoming.Dirty = hasDraft

	return incoming
}
