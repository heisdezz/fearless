package environment

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fearless/internal/core/format"
	"fearless/internal/core/model"
)

// DirName is the directory, relative to a collection root, that holds
// environment files.
//
// Both dialects use it. Source: bruno-electron/src/ipc/network/index.js
// (reads `path.join(collectionPath, 'environments')`).
const DirName = "environments"

// MaxFileBytes caps a single environment file read.
//
// Environment files are hand-written and tiny; a cap turns an accidental
// catastrophic path (a symlink to a disk image) into an error instead of a
// memory exhaustion.
const MaxFileBytes = 4 << 20

// ErrNoEnvironments reports a collection directory with no environments folder.
var ErrNoEnvironments = errors.New("environment: no environments directory")

// Loader reads environment files from a collection directory.
type Loader struct {
	// Format selects the codec. A `.bru` collection and an OpenCollection YAML
	// collection name their files differently, so the dialect has to be known.
	Format model.CollectionFormat
	// Root is the collection directory.
	Root string
}

// Load reads every environment in the collection.
//
// Files are returned in name order so the environment picker is stable between
// runs. A single unreadable or malformed file fails the whole load: a collection
// whose environment cannot be parsed should not appear to have one fewer
// environment, because the user would then select a different one by mistake.
func (l Loader) Load() ([]*model.Environment, error) {
	codec, err := format.For(l.Format)
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(l.Root, DirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrNoEnvironments, dir)
		}
		return nil, fmt.Errorf("environment: read %s: %w", dir, err)
	}

	ext := extensionFor(l.Format)

	var environments []*model.Environment
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ext) {
			continue
		}
		path := filepath.Join(dir, entry.Name())

		data, err := readCapped(path)
		if err != nil {
			return nil, err
		}
		env, err := codec.ParseEnvironment(data)
		if err != nil {
			return nil, fmt.Errorf("environment: parse %s: %w", entry.Name(), err)
		}
		if env == nil {
			return nil, fmt.Errorf("environment: %s produced no environment", entry.Name())
		}

		env.Filename = entry.Name()
		env.Format = l.Format
		// A file with no name is identified by its filename, which is how Bruno
		// falls back when the `name:` line is missing.
		if strings.TrimSpace(env.Name) == "" {
			env.Name = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		}
		environments = append(environments, env)
	}

	sort.SliceStable(environments, func(i, j int) bool {
		return strings.ToLower(environments[i].Name) < strings.ToLower(environments[j].Name)
	})
	return environments, nil
}

// LoadAndResolve loads the collection's environments and resolves the one whose
// UID matches.
//
// A uid that matches nothing, including the empty uid, yields (nil, nil, nil):
// "no environment selected" is a valid state, not an error.
func (l Loader) LoadAndResolve(uid string) ([]*model.Environment, *Resolution, error) {
	environments, err := l.Load()
	if err != nil {
		return nil, nil, err
	}
	target := Select(environments, uid)
	if target == nil {
		return environments, nil, nil
	}
	resolution, err := Resolve(environments, target)
	if err != nil {
		return environments, nil, err
	}
	return environments, &resolution, nil
}

// Save writes one environment back to its file, creating the directory if
// needed.
//
// The file name comes from the environment's Filename when set, and otherwise
// from its Name. Renaming an environment therefore writes a new file rather than
// renaming the old one; callers that support rename must move the file
// themselves, because a leftover file would otherwise reappear as a phantom
// environment.
func (l Loader) Save(env *model.Environment) (string, error) {
	if env == nil {
		return "", errors.New("environment: nil environment")
	}
	codec, err := format.For(l.Format)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(l.Root, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("environment: create %s: %w", dir, err)
	}

	name := env.Filename
	if name == "" {
		base := sanitiseFilename(env.Name)
		if base == "" {
			return "", errors.New("environment: cannot derive a file name")
		}
		name = base + extensionFor(l.Format)
		env.Filename = name
	}
	path := filepath.Join(dir, name)

	data, err := codec.WriteEnvironment(env)
	if err != nil {
		return "", fmt.Errorf("environment: write %s: %w", name, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("environment: write %s: %w", path, err)
	}
	return path, nil
}

// Remove deletes an environment's file.
//
// The path is confined to the collection's environments directory, so a Filename
// containing `..` cannot delete something outside the collection.
func (l Loader) Remove(env *model.Environment) error {
	if env == nil || env.Filename == "" {
		return nil
	}
	dir := filepath.Join(l.Root, DirName)
	path := filepath.Join(dir, filepath.Base(env.Filename))
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("environment: remove %s: %w", path, err)
	}
	return nil
}

// extensionFor returns the environment file extension for a dialect.
func extensionFor(f model.CollectionFormat) string {
	switch f {
	case model.FormatYML:
		return ".yml"
	case model.FormatBru:
		return ".bru"
	default:
		// An unknown dialect is a programming error the codec lookup already
		// caught; `.bru` keeps the return total.
		return ".bru"
	}
}

// sanitiseFilename reduces a display name to a safe file stem.
//
// Names are user-visible and may contain anything; the filesystem decides what
// is legal, so separators and control characters become dashes and everything
// else is kept. Without this, an environment named "Prod/US" would try to write
// into a subdirectory that does not exist.
func sanitiseFilename(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "\x00", "")
	cleaned := strings.TrimSpace(replacer.Replace(name))
	cleaned = strings.Trim(cleaned, ".")
	return cleaned
}

// readCapped reads a file, refusing anything larger than MaxFileBytes.
//
// Environment files are hand-written and tiny, so the size check is a guard
// against a catastrophic path resolving somewhere unexpected rather than a real
// limit on legitimate input.
func readCapped(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("environment: stat %s: %w", path, err)
	}
	if info.Size() > MaxFileBytes {
		return nil, fmt.Errorf("environment: %s is %d bytes, over the %d byte limit", path, info.Size(), MaxFileBytes)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("environment: read %s: %w", path, err)
	}
	if len(data) > MaxFileBytes {
		return nil, fmt.Errorf("environment: %s grew past the %d byte limit", path, MaxFileBytes)
	}
	return data, nil
}
