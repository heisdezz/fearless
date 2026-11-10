package collection

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fearless/internal/core/format"
	"fearless/internal/core/format/bru"
	"fearless/internal/core/format/yamlcodec"
	"fearless/internal/core/model"
)

var registerOnce sync.Once

func init() {
	registerOnce.Do(func() {
		if _, ok := format.Get(model.FormatBru); !ok {
			format.Register(bru.New())
		}
		if _, ok := format.Get(model.FormatYML); !ok {
			format.Register(yamlcodec.New())
		}
	})
}

// DefaultIgnorePatterns are standard paths skipped during collection tree traversal.
var DefaultIgnorePatterns = []string{
	".git",
	"node_modules",
	".DS_Store",
	".tmp-*",
}

// Load loads a collection tree from the specified directory.
func Load(dir string) (*model.Collection, error) {
	return LoadCollection(dir)
}

// LoadCollection loads a complete collection: detecting format, parsing collection config,
// environments, and recursively building the sorted item hierarchy.
func LoadCollection(dir string) (*model.Collection, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("collection: resolve path %s: %w", dir, err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("collection: stat %s: %w", absDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("collection: %s is not a directory", absDir)
	}

	// 1. Detect format codec
	codec, err := format.Detect(absDir)
	if err != nil {
		return nil, fmt.Errorf("collection: detect format in %s: %w", absDir, err)
	}

	// 2. Read collection configuration
	configName := format.BrunoJSONFilename
	if codec.Name() == model.FormatYML {
		configName = format.OpenCollectionYMLFilename
	}
	configPath := filepath.Join(absDir, configName)
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("collection: read %s: %w", configName, err)
	}

	cfg, err := codec.ParseCollection(configBytes)
	if err != nil {
		return nil, fmt.Errorf("collection: parse %s: %w", configName, err)
	}

	name := cfg.Name
	if name == "" {
		name = filepath.Base(absDir)
	}

	col := &model.Collection{
		UID:    GenerateUID(absDir),
		Path:   absDir,
		Name:   name,
		Config: *cfg,
	}

	// 3. Parse root collection file (e.g. collection.bru / collection.yml) if present
	rootFileName := "collection.bru"
	if codec.Name() == model.FormatYML {
		rootFileName = "collection.yml"
	}
	rootFilePath := filepath.Join(absDir, rootFileName)
	if rootBytes, err := os.ReadFile(rootFilePath); err == nil {
		if rootItem, err := codec.ParseFolder(rootBytes); err == nil && rootItem != nil {
			// Attach root-level folder settings to collection
			// If needed by UI/Engine
		}
	}

	// 4. Parse environments (environments/ directory)
	envsDir := filepath.Join(absDir, "environments")
	if envEntries, err := os.ReadDir(envsDir); err == nil {
		for _, entry := range envEntries {
			if entry.IsDir() {
				continue
			}
			ext := filepath.Ext(entry.Name())
			if ext != ".bru" && ext != ".yml" && ext != ".yaml" {
				continue
			}
			envPath := filepath.Join(envsDir, entry.Name())
			envBytes, err := os.ReadFile(envPath)
			if err != nil {
				continue
			}
			env, err := codec.ParseEnvironment(envBytes)
			if err == nil && env != nil {
				env.UID = GenerateUID(envPath)
				if env.Name == "" {
					env.Name = strings.TrimSuffix(entry.Name(), ext)
				}
				env.Filename = entry.Name()
				env.Format = codec.Name()
				col.Environments = append(col.Environments, env)
			}
		}
	}

	// 5. Build ignore filter
	ignores := append([]string(nil), DefaultIgnorePatterns...)
	ignores = append(ignores, cfg.Ignore...)

	// 6. Recursively load items in collection directory
	items, err := loadFolderItems(absDir, absDir, codec, ignores)
	if err != nil {
		return nil, err
	}
	SortItems(items)
	col.Items = items

	return col, nil
}

// loadFolderItems recursively scans a folder, creating folder and request items.
func loadFolderItems(currentDir, rootDir string, codec format.Codec, ignores []string) ([]*model.Item, error) {
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, fmt.Errorf("collection: read dir %s: %w", currentDir, err)
	}

	var items []*model.Item

	for _, entry := range entries {
		name := entry.Name()

		// Skip ignored directories/files
		if shouldIgnore(name, currentDir == rootDir, ignores) {
			continue
		}

		fullPath := filepath.Join(currentDir, name)

		if entry.IsDir() {
			// Subdirectory: represents a folder item
			subFolderItems, err := loadFolderItems(fullPath, rootDir, codec, ignores)
			if err != nil {
				return nil, err
			}
			SortItems(subFolderItems)

			folderItem := &model.Item{
				UID:      GenerateUID(fullPath),
				Type:     model.ItemFolder,
				Name:     name,
				Path:     fullPath,
				Filename: name,
				Children: subFolderItems,
			}

			// Check for folder config file (folder.bru or folder.yml)
			folderConfigName := "folder.bru"
			if codec.Name() == model.FormatYML {
				folderConfigName = "folder.yml"
			}
			folderConfigPath := filepath.Join(fullPath, folderConfigName)
			if fBytes, err := os.ReadFile(folderConfigPath); err == nil {
				if parsedFolder, err := codec.ParseFolder(fBytes); err == nil && parsedFolder != nil {
					if parsedFolder.Name != "" {
						folderItem.Name = parsedFolder.Name
					}
					folderItem.Seq = parsedFolder.Seq
					folderItem.Auth = parsedFolder.Auth
					folderItem.Headers = parsedFolder.Headers
					folderItem.Vars = parsedFolder.Vars
					folderItem.Script = parsedFolder.Script
					folderItem.Tags = parsedFolder.Tags
				}
			}

			items = append(items, folderItem)
		} else {
			// Regular file: check if it's a request file
			ext := filepath.Ext(name)
			if ext != ".bru" && ext != ".yml" && ext != ".yaml" {
				continue
			}

			// Skip metadata/config files
			if name == format.BrunoJSONFilename ||
				name == format.OpenCollectionYMLFilename ||
				name == "collection.bru" ||
				name == "collection.yml" ||
				name == "folder.bru" ||
				name == "folder.yml" ||
				name == "MANIFEST.txt" {
				continue
			}

			data, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, fmt.Errorf("collection: read file %s: %w", fullPath, err)
			}

			req, err := codec.ParseRequest(data)
			if err != nil {
				// If a file has syntax errors, record inert/unparseable item or return error
				return nil, fmt.Errorf("collection: parse request %s: %w", fullPath, err)
			}

			itemName := req.Name
			if itemName == "" {
				itemName = strings.TrimSuffix(name, ext)
			}

			item := &model.Item{
				UID:      GenerateUID(fullPath),
				Type:     model.ItemHTTPRequest,
				Name:     itemName,
				Path:     fullPath,
				Filename: name,
				Seq:      req.Seq,
				Tags:     req.Tags,
				Request:  req,
			}

			items = append(items, item)
		}
	}

	return items, nil
}

// shouldIgnore checks whether a file or directory name matches the ignore list.
func shouldIgnore(name string, isRoot bool, ignores []string) bool {
	if isRoot && name == "environments" {
		return true
	}
	if strings.HasPrefix(name, ".tmp-") {
		return true
	}

	for _, pattern := range ignores {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if name == pattern {
			return true
		}
		matched, _ := filepath.Match(pattern, name)
		if matched {
			return true
		}
	}
	return false
}

// FindItemByPath searches for an item in the collection matching the absolute path.
func FindItemByPath(col *model.Collection, path string) *model.Item {
	if col == nil {
		return nil
	}
	clean := filepath.Clean(path)
	var found *model.Item
	_ = col.Walk(func(it *model.Item) error {
		if filepath.Clean(it.Path) == clean {
			found = it
			return errors.New("found")
		}
		return nil
	})
	return found
}
