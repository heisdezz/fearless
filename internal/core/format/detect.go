package format

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"fearless/internal/core/model"
)

const (
	// BrunoJSONFilename is the collection configuration file for the .bru dialect.
	BrunoJSONFilename = "bruno.json"

	// OpenCollectionYMLFilename is the collection configuration file for the OpenCollection YAML dialect.
	OpenCollectionYMLFilename = "opencollection.yml"
)

// formatPriority returns a sorting weight for deterministic detection ordering.
// FormatBru (the primary/native Bruno format) takes precedence over FormatYML.
func formatPriority(name model.CollectionFormat) int {
	switch name {
	case model.FormatBru:
		return 1
	case model.FormatYML:
		return 2
	default:
		return 100
	}
}

// orderedCodecs returns the registered codecs in deterministic priority order.
func orderedCodecs() []Codec {
	registryMu.RLock()
	defer registryMu.RUnlock()

	list := make([]Codec, 0, len(codecs))
	for _, c := range codecs {
		list = append(list, c)
	}

	sort.Slice(list, func(i, j int) bool {
		pi := formatPriority(list[i].Name())
		pj := formatPriority(list[j].Name())
		if pi != pj {
			return pi < pj
		}
		return list[i].Name() < list[j].Name()
	})
	return list
}

// Detect inspects dir and returns the first registered Codec that matches it.
//
// Detection order is deterministic: FormatBru is tested before FormatYML.
// If both dialects are present in the same directory, FormatBru is chosen deterministically.
// If no registered codec detects a collection, ErrNoCollection is returned.
func Detect(dir string) (Codec, error) {
	for _, c := range orderedCodecs() {
		detected, err := c.Detect(dir)
		if err != nil {
			return nil, fmt.Errorf("detect format %q: %w", c.Name(), err)
		}
		if detected {
			return c, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNoCollection, dir)
}

// DetectFormat detects and returns the model.CollectionFormat for dir.
func DetectFormat(dir string) (model.CollectionFormat, error) {
	c, err := Detect(dir)
	if err != nil {
		return "", err
	}
	return c.Name(), nil
}

// FileExists reports whether path exists and is a regular file or symlink.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DetectFile reports whether a named collection config file exists in dir.
func DetectFile(dir, filename string) bool {
	return FileExists(filepath.Join(dir, filename))
}
