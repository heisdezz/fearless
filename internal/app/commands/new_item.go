package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fearless/internal/core/collection"
	"fearless/internal/core/events"
	"fearless/internal/core/format"
	"fearless/internal/core/model"
	"fearless/internal/core/persistence"
)

// NewItem creates a new request or folder item on disk and inserts it into the collection tree.
//
// The item is sorted into its folder using Bruno's sequence and alphabetical rules, stored in the
// workspace state, and announced via CollectionChanged with Added set.
func (s *CollectionService) NewItem(ctx context.Context, cmd NewItemCmd) (*model.Item, error) {
	if cmd.CollectionUID == "" {
		return nil, fmt.Errorf("%w: collection UID is required", ErrInvalidInput)
	}

	snap := s.store.Snapshot()
	col := snap.FindCollection(cmd.CollectionUID)
	if col == nil {
		return nil, fmt.Errorf("%w: %s", ErrCollectionNotFound, cmd.CollectionUID)
	}

	// 1. Resolve parent directory
	parentDir := col.Path
	if cmd.ParentFolderUID != "" {
		parentFolder := col.FindItem(cmd.ParentFolderUID)
		if parentFolder == nil || !parentFolder.IsFolder() {
			return nil, fmt.Errorf("%w: parent folder %s not found or not a folder", ErrItemNotFound, cmd.ParentFolderUID)
		}
		parentDir = parentFolder.Path
	}

	// 2. Resolve dialect codec and extension
	fmtName := cmd.Format
	if fmtName == "" {
		fmtName = col.Config.Format
	}
	if fmtName == "" {
		fmtName = model.FormatBru
	}

	codec, err := format.For(fmtName)
	if err != nil {
		return nil, fmt.Errorf("commands: resolve format codec %q: %w", fmtName, err)
	}

	ext := ".bru"
	if fmtName == model.FormatYML {
		ext = ".yml"
	}

	// 3. Resolve item name and file/directory name
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		if cmd.Type == model.ItemFolder {
			name = "New Folder"
		} else {
			name = "New Request"
		}
	}

	sanitized := sanitizeFilename(name)
	var (
		itemPath string
		newItem  *model.Item
	)

	if cmd.Type == model.ItemFolder {
		itemPath = filepath.Join(parentDir, sanitized)
		if err := os.MkdirAll(itemPath, 0o755); err != nil {
			return nil, fmt.Errorf("commands: create folder %s: %w", itemPath, err)
		}

		// Write initial folder.bru or folder.yml
		folderConfigName := "folder.bru"
		if fmtName == model.FormatYML {
			folderConfigName = "folder.yml"
		}
		cfgPath := filepath.Join(itemPath, folderConfigName)

		folderItem := &model.Item{
			Type: model.ItemFolder,
			Name: name,
		}
		data, err := codec.WriteFolder(folderItem)
		if err != nil {
			return nil, fmt.Errorf("commands: serialize folder %s: %w", name, err)
		}

		collection.SuppressPath(cfgPath)
		if err := persistence.WriteAtomic(cfgPath, data, 0o644); err != nil {
			return nil, fmt.Errorf("commands: write folder config: %w", err)
		}

		newItem = &model.Item{
			UID:      collection.GenerateUID(itemPath),
			Type:     model.ItemFolder,
			Name:     name,
			Path:     itemPath,
			Filename: sanitized,
		}
	} else {
		// Request item
		filename := sanitized
		if !strings.HasSuffix(filename, ext) {
			filename += ext
		}
		itemPath = filepath.Join(parentDir, filename)

		method := strings.ToUpper(strings.TrimSpace(cmd.Method))
		if method == "" {
			method = "GET"
		}

		reqURL := strings.TrimSpace(cmd.URL)
		req := &model.Request{
			Name:   name,
			Method: method,
			URL:    reqURL,
			Body:   &model.Body{Mode: model.BodyNone},
		}

		data, err := codec.WriteRequest(req)
		if err != nil {
			return nil, fmt.Errorf("commands: serialize request %s: %w", name, err)
		}

		collection.SuppressPath(itemPath)
		if err := persistence.WriteAtomic(itemPath, data, 0o644); err != nil {
			return nil, fmt.Errorf("commands: write request file %s: %w", itemPath, err)
		}

		newItem = &model.Item{
			UID:      collection.GenerateUID(itemPath),
			Type:     model.ItemHTTPRequest,
			Name:     name,
			Path:     itemPath,
			Filename: filename,
			Request:  req,
		}
	}

	// 4. Update workspace state
	s.store.Apply(func(state *model.WorkspaceState) {
		targetCol := state.FindCollection(cmd.CollectionUID)
		if targetCol == nil {
			return
		}

		if cmd.ParentFolderUID == "" {
			targetCol.Items = append(targetCol.Items, newItem)
			collection.SortItems(targetCol.Items)
		} else {
			parent := targetCol.FindItem(cmd.ParentFolderUID)
			if parent != nil {
				parent.Children = append(parent.Children, newItem)
				collection.SortItems(parent.Children)
			}
		}
	})

	// 5. Publish CollectionChanged event (Self, Added)
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin: events.OriginSelf,
			Added:  []string{newItem.UID},
		},
	})

	return newItem, nil
}

func sanitizeFilename(name string) string {
	r := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	clean := strings.TrimSpace(r.Replace(name))
	if clean == "" || clean == "." {
		return "item"
	}
	return clean
}
