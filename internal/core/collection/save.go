package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fearless/internal/core/format"
	"fearless/internal/core/model"
	"fearless/internal/core/persistence"
)

// SaveItem commits in-progress draft edits to saved state, serializes the item using
// the appropriate dialect codec, and writes it atomically to disk with self-write suppression.
func SaveItem(col *model.Collection, item *model.Item) error {
	if col == nil || item == nil {
		return fmt.Errorf("collection: nil collection or item")
	}

	codec, err := resolveCodec(col)
	if err != nil {
		return err
	}

	// Promote Draft to saved Request if draft exists
	if item.Draft != nil {
		item.Request = item.Draft
		item.Draft = nil
	}

	targetPath := item.Path
	var data []byte

	if item.IsFolder() {
		data, err = codec.WriteFolder(item)
		if err != nil {
			return fmt.Errorf("collection: serialize folder %s: %w", item.Name, err)
		}
		// Folder settings go into folder.bru or folder.yml inside folder directory
		folderFileName := "folder.bru"
		if codec.Name() == model.FormatYML {
			folderFileName = "folder.yml"
		}
		if !strings.HasSuffix(targetPath, folderFileName) {
			targetPath = filepath.Join(targetPath, folderFileName)
		}
	} else {
		if item.Request == nil {
			return fmt.Errorf("collection: request item %s has nil Request", item.Name)
		}
		data, err = codec.WriteRequest(item.Request)
		if err != nil {
			return fmt.Errorf("collection: serialize request %s: %w", item.Name, err)
		}
	}

	// Suppress watcher reload on this path before writing
	SuppressPath(targetPath)

	if err := persistence.WriteAtomic(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("collection: write item %s: %w", targetPath, err)
	}

	return nil
}

// SaveEnvironment writes an environment file atomically to disk.
func SaveEnvironment(col *model.Collection, env *model.Environment) error {
	if col == nil || env == nil {
		return fmt.Errorf("collection: nil collection or environment")
	}

	codec, err := resolveCodec(col)
	if err != nil {
		return err
	}

	data, err := codec.WriteEnvironment(env)
	if err != nil {
		return fmt.Errorf("collection: serialize environment %s: %w", env.Name, err)
	}

	filename := env.Filename
	if filename == "" {
		ext := ".bru"
		if codec.Name() == model.FormatYML {
			ext = ".yml"
		}
		filename = env.Name + ext
	}

	targetPath := filepath.Join(col.Path, "environments", filename)
	SuppressPath(targetPath)

	if err := persistence.WriteAtomic(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("collection: write environment %s: %w", targetPath, err)
	}

	return nil
}

// SaveCollectionConfig writes bruno.json or opencollection.yml atomically to disk.
func SaveCollectionConfig(col *model.Collection) error {
	if col == nil {
		return fmt.Errorf("collection: nil collection")
	}

	codec, err := resolveCodec(col)
	if err != nil {
		return err
	}

	data, err := codec.WriteCollection(&col.Config)
	if err != nil {
		return fmt.Errorf("collection: serialize collection config: %w", err)
	}

	configName := format.BrunoJSONFilename
	if codec.Name() == model.FormatYML {
		configName = format.OpenCollectionYMLFilename
	}

	targetPath := filepath.Join(col.Path, configName)
	SuppressPath(targetPath)

	if err := persistence.WriteAtomic(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("collection: write config %s: %w", targetPath, err)
	}

	// Keep the root file's variables in step with the config, so a host set in
	// the UI is still there after the next load.
	if err := writeRootVars(col.Path, col); err != nil {
		return err
	}

	return nil
}

// ResequenceItems assigns sequential numbers (1, 2, 3...) to a slice of items and saves each item.
func ResequenceItems(col *model.Collection, items []*model.Item) error {
	for i, it := range items {
		seq := i + 1
		it.Seq = &seq
		if it.Request != nil {
			it.Request.Seq = &seq
		}
		if err := SaveItem(col, it); err != nil {
			return err
		}
	}
	return nil
}

// DeleteItem removes an item from disk and from the collection tree.
func DeleteItem(col *model.Collection, item *model.Item) error {
	if col == nil || item == nil {
		return fmt.Errorf("collection: nil collection or item")
	}

	SuppressPath(item.Path)

	if err := os.RemoveAll(item.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("collection: delete item from disk %s: %w", item.Path, err)
	}

	col.Items = removeItemFromSlice(col.Items, item.UID)
	return nil
}

func removeItemFromSlice(items []*model.Item, uid string) []*model.Item {
	var out []*model.Item
	for _, it := range items {
		if it.UID == uid {
			continue
		}
		if len(it.Children) > 0 {
			it.Children = removeItemFromSlice(it.Children, uid)
		}
		out = append(out, it)
	}
	return out
}

func resolveCodec(col *model.Collection) (format.Codec, error) {
	codec, err := format.Detect(col.Path)
	if err == nil && codec != nil {
		return codec, nil
	}

	// Fallback to FormatBru
	c, ok := format.Get(model.FormatBru)
	if !ok {
		return nil, fmt.Errorf("collection: default bru codec not registered")
	}
	return c, nil
}
