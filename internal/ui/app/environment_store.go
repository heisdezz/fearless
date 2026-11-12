package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fearless/internal/core/collection"
	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

// environmentStore persists environments for the collection open in the UI.
//
// The environment presenter edits a draft and hands it back to a saver; it
// knows nothing about files. This adapter is the file side: an environment is
// a file under <collection>/environments, so saving one writes that file and
// the in-memory model changes at once.
//
// The UI owns its collections in memory rather than forking commands, so this
// sits directly on collection.SaveEnvironment instead of going through
// CollectionService, which works on a bus store the UI does not populate.
type environmentStore struct {
	// collection returns the currently bound collection, read at call time so
	// switching collections needs no re-wiring.
	collection func() *model.Collection
}

var (
	_ presenters.EnvSelectorPort = (*environmentStore)(nil)
	_ presenters.EnvSaverPort    = (*environmentStore)(nil)
	_ presenters.EnvCreatorPort  = (*environmentStore)(nil)
	_ presenters.EnvDeleterPort  = (*environmentStore)(nil)
)

// SelectEnvironment sets the collection's active environment. The presenter
// stores the same value on the model itself, so there is no file to write here:
// the active environment is remembered with the collection's open state.
func (s *environmentStore) SelectEnvironment(_ context.Context, _ string, envName string) error {
	col := s.collection()
	if col == nil {
		return nil
	}
	col.ActiveEnvironment = strings.TrimSpace(envName)
	return nil
}

// SaveEnvironment writes env to disk. A collection without a path exists only
// in memory (an import before the storage folder was chosen); the presenter has
// already put the draft into the model, so there is nothing to write and
// reporting success keeps the in-memory edit.
func (s *environmentStore) SaveEnvironment(_ context.Context, _ string, env *model.Environment) error {
	col := s.collection()
	if col == nil || env == nil || col.Path == "" {
		return nil
	}
	if err := collection.SaveEnvironment(col, env); err != nil {
		return fmt.Errorf("save environment %q: %w", env.Name, err)
	}
	return nil
}

// CreateEnvironment adds a named environment to the collection and writes it.
// The presenter appends the returned environment to the collection itself.
func (s *environmentStore) CreateEnvironment(_ context.Context, _ string, name string) (*model.Environment, error) {
	col := s.collection()
	if col == nil {
		return nil, nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = presenters.NextEnvironmentName(col.Environments)
	}
	env := &model.Environment{Name: name}
	if col.Path == "" {
		return env, nil
	}
	if err := collection.SaveEnvironment(col, env); err != nil {
		return nil, fmt.Errorf("create environment %q: %w", name, err)
	}
	// SaveEnvironment names the file after the environment, so record it the
	// way a load would, and future saves rewrite the same file.
	env.Filename = environmentFilename(col, env)
	return env, nil
}

// DeleteEnvironment removes an environment by name, file included. A collection
// with no path has no file to remove.
func (s *environmentStore) DeleteEnvironment(_ context.Context, _ string, name string) error {
	col := s.collection()
	if col == nil || name == "" {
		return nil
	}
	env := findEnvironment(col, name)
	if env == nil {
		return nil // already gone
	}
	if col.Path != "" {
		path := filepath.Join(col.Path, "environments", environmentFilename(col, env))
		collection.SuppressPath(path)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete environment %q: %w", name, err)
		}
	}
	if col.ActiveEnvironment == env.Name {
		col.ActiveEnvironment = ""
	}
	return nil
}

func findEnvironment(col *model.Collection, name string) *model.Environment {
	for _, env := range col.Environments {
		if env != nil && strings.EqualFold(env.Name, name) {
			return env
		}
	}
	return nil
}

// environmentFilename is the file an environment lives in, matching the naming
// collection.SaveEnvironment uses when it writes one.
func environmentFilename(col *model.Collection, env *model.Environment) string {
	if env.Filename != "" {
		return env.Filename
	}
	ext := ".bru"
	if col.Config.Format == model.FormatYML {
		ext = ".yml"
	}
	return env.Name + ext
}
