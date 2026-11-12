package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fearless/internal/core/format"
	"fearless/internal/core/model"
)

// CopyIntoStorage writes the collection directory src into storageRoot under a
// free folder name and returns the destination.
//
// This is how an imported collection stops being memory-only: parsing gives a
// model, but the app opens collections by path, and a reopened collection must
// find real files. The copy is recursive and skips nothing — a collection is
// its files, so a partial copy would be a different collection.
//
// An existing destination is never overwritten: a second import of the same
// name lands in "<name> (2)" rather than clobbering the first.
func CopyIntoStorage(src, storageRoot string) (string, error) {
	if storageRoot == "" {
		return "", fmt.Errorf("collection: no storage folder configured")
	}
	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return "", fmt.Errorf("collection: resolve %s: %w", src, err)
	}
	info, err := os.Stat(srcAbs)
	if err != nil {
		return "", fmt.Errorf("collection: stat %s: %w", srcAbs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("collection: %s is not a directory", srcAbs)
	}
	if err := os.MkdirAll(storageRoot, 0o755); err != nil {
		return "", fmt.Errorf("collection: create storage folder: %w", err)
	}

	dst := uniqueDir(storageRoot, sanitizeDirName(filepath.Base(srcAbs)))
	if err := copyTree(srcAbs, dst); err != nil {
		// Do not leave a half-written collection behind.
		_ = os.RemoveAll(dst)
		return "", err
	}
	return dst, nil
}

// CopyModelIntoStorage writes a parsed collection that has no files on disk
// yet, such as one converted from a pasted cURL, Postman or OpenAPI document.
// It returns the destination directory.
//
// It writes what a reopened collection needs: the collection configuration, one
// file per request, and nested folders for nested items. A collection made this
// way loads with Load, which is what "imported" has to mean once the app
// restarts.
func CopyModelIntoStorage(col *model.Collection, storageRoot string) (string, error) {
	if col == nil {
		return "", fmt.Errorf("collection: nil collection")
	}
	if storageRoot == "" {
		return "", fmt.Errorf("collection: no storage folder configured")
	}
	if err := os.MkdirAll(storageRoot, 0o755); err != nil {
		return "", fmt.Errorf("collection: create storage folder: %w", err)
	}

	name := strings.TrimSpace(col.Name)
	if name == "" {
		name = "Collection"
	}
	dst := uniqueDir(storageRoot, sanitizeDirName(name))
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return "", fmt.Errorf("collection: create %s: %w", dst, err)
	}

	codec, err := codecForFormat(col.Config.Format)
	if err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}

	cfg := col.Config
	if strings.TrimSpace(cfg.Name) == "" {
		cfg.Name = name
	}
	cfgBytes, err := codec.WriteCollection(&cfg)
	if err != nil {
		_ = os.RemoveAll(dst)
		return "", fmt.Errorf("collection: write config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dst, configFilename(col.Config.Format)), cfgBytes, 0o644); err != nil {
		_ = os.RemoveAll(dst)
		return "", fmt.Errorf("collection: write config file: %w", err)
	}

	if err := writeItems(dst, col.Items, codec); err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	return dst, nil
}

// writeItems writes items recursively into dir.
func writeItems(dir string, items []*model.Item, codec format.Codec) error {
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.IsFolder() {
			sub := filepath.Join(dir, sanitizeDirName(item.Name))
			if err := os.MkdirAll(sub, 0o755); err != nil {
				return fmt.Errorf("collection: create folder %s: %w", sub, err)
			}
			if err := writeItems(sub, item.Children, codec); err != nil {
				return err
			}
			continue
		}
		if item.Request == nil {
			continue
		}
		data, err := codec.WriteRequest(item.Request)
		if err != nil {
			return fmt.Errorf("collection: serialize %s: %w", item.Name, err)
		}
		path := filepath.Join(dir, requestFilename(codec, item.Name))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("collection: write %s: %w", path, err)
		}
	}
	return nil
}

// codecForFormat returns the codec for a dialect, defaulting to bru.
func codecForFormat(f model.CollectionFormat) (format.Codec, error) {
	if c, ok := format.Get(f); ok && c != nil {
		return c, nil
	}
	if c, ok := format.Get(model.FormatBru); ok && c != nil {
		return c, nil
	}
	return nil, fmt.Errorf("collection: no codec for format %q", f)
}

// configFilename is the collection configuration file for a dialect.
func configFilename(f model.CollectionFormat) string {
	if f == model.FormatYML {
		return format.OpenCollectionYMLFilename
	}
	return format.BrunoJSONFilename
}

// requestFilename is the file name for a request: its name with the dialect's
// extension, made safe for the filesystem.
func requestFilename(codec format.Codec, name string) string {
	base := sanitizeDirName(name)
	if codec.Name() == model.FormatYML {
		return base + ".yml"
	}
	return base + ".bru"
}

// uniqueDir returns a path under root for name that does not exist yet.
func uniqueDir(root, name string) string {
	candidate := filepath.Join(root, name)
	if !exists(candidate) {
		return candidate
	}
	for i := 2; ; i++ {
		candidate = filepath.Join(root, fmt.Sprintf("%s (%d)", name, i))
		if !exists(candidate) {
			return candidate
		}
	}
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// sanitizeDirName makes a name safe as a file or folder name.
func sanitizeDirName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.NewReplacer("/", "-", "\\", "-", ":", "-", "\x00", "").Replace(name)
	name = strings.Trim(name, ". ")
	if name == "" {
		return "Collection"
	}
	return name
}

// copyTree copies src into dst recursively, preserving file modes.
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("collection: walk %s: %w", path, err)
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("collection: read %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, info.Mode().Perm()); err != nil {
			return fmt.Errorf("collection: write %s: %w", target, err)
		}
		return nil
	})
}
