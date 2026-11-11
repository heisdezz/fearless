package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fearless/internal/core/collection"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

var (
	// ErrEnvironmentNotFound is returned when an environment name or UID cannot be resolved.
	ErrEnvironmentNotFound = errors.New("environment not found")
)

// Consumer interfaces for environment management per interface-design.md §2.

// EnvSelector selects the active environment for a collection.
type EnvSelector interface {
	Execute(ctx context.Context, cmd SelectEnvironmentCmd) error
}

// EnvSaver persists modifications to an environment on disk and in workspace state.
type EnvSaver interface {
	Execute(ctx context.Context, cmd SaveEnvironmentCmd) error
}

// EnvCreator creates a new environment file and adds it to the collection.
type EnvCreator interface {
	Execute(ctx context.Context, cmd NewEnvironmentCmd) (*model.Environment, error)
}

// EnvDeleter removes an environment file from disk and collection state.
type EnvDeleter interface {
	Execute(ctx context.Context, cmd DeleteEnvironmentCmd) error
}

// Command payloads.

// SelectEnvironmentCmd specifies the active environment to select.
// EnvironmentName can be empty to clear the active environment selection.
type SelectEnvironmentCmd struct {
	CollectionUID   string
	EnvironmentName string
}

// SaveEnvironmentCmd specifies the environment to update and save.
type SaveEnvironmentCmd struct {
	CollectionUID string
	Environment   *model.Environment
}

// NewEnvironmentCmd specifies details for creating a new environment.
type NewEnvironmentCmd struct {
	CollectionUID string
	Name          string
	Variables     []model.Variable
	Color         string
	Extends       string
	Format        model.CollectionFormat
}

// DeleteEnvironmentCmd specifies which environment to delete.
type DeleteEnvironmentCmd struct {
	CollectionUID   string
	EnvironmentName string
	EnvironmentUID  string
}

// NewEnvironment creates an environment file on disk and inserts it into the collection.
func (s *CollectionService) NewEnvironment(ctx context.Context, cmd NewEnvironmentCmd) (*model.Environment, error) {
	if cmd.CollectionUID == "" {
		return nil, fmt.Errorf("%w: collection UID is required", ErrInvalidInput)
	}

	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: environment name is required", ErrInvalidInput)
	}

	snap := s.store.Snapshot()
	col := snap.FindCollection(cmd.CollectionUID)
	if col == nil {
		return nil, fmt.Errorf("%w: %s", ErrCollectionNotFound, cmd.CollectionUID)
	}

	// Check for existing environment with the same name
	for _, env := range col.Environments {
		if strings.EqualFold(env.Name, name) {
			return nil, fmt.Errorf("commands: environment %q already exists", name)
		}
	}

	// Determine dialect format and file extension
	fmtName := cmd.Format
	if fmtName == "" {
		fmtName = col.Config.Format
	}
	if fmtName == "" {
		fmtName = model.FormatBru
	}

	ext := ".bru"
	if fmtName == model.FormatYML {
		ext = ".yml"
	}

	filename := sanitizeFilename(name) + ext
	envsDir := filepath.Join(col.Path, "environments")
	if err := os.MkdirAll(envsDir, 0o755); err != nil {
		return nil, fmt.Errorf("commands: create environments directory: %w", err)
	}

	envPath := filepath.Join(envsDir, filename)
	newEnv := &model.Environment{
		UID:       collection.GenerateUID(envPath),
		Name:      name,
		Color:     cmd.Color,
		Extends:   cmd.Extends,
		Variables: cmd.Variables,
		Format:    fmtName,
		Filename:  filename,
	}

	// Persist atomically to disk
	if err := collection.SaveEnvironment(col, newEnv); err != nil {
		return nil, fmt.Errorf("commands: save new environment %s: %w", name, err)
	}

	// Update workspace state
	s.store.Apply(func(state *model.WorkspaceState) {
		targetCol := state.FindCollection(cmd.CollectionUID)
		if targetCol == nil {
			return
		}
		targetCol.Environments = append(targetCol.Environments, newEnv)
	})

	// Publish CollectionChanged event (Self, Added)
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin: events.OriginSelf,
			Added:  []string{newEnv.UID},
		},
	})

	return newEnv, nil
}

// DeleteEnvironment deletes an environment file from disk, removes it from collection state,
// unselects it if it was active, and publishes a CollectionChanged event.
//
// Idempotency: Deleting an environment that is not present succeeds cleanly.
func (s *CollectionService) DeleteEnvironment(ctx context.Context, cmd DeleteEnvironmentCmd) error {
	if cmd.CollectionUID == "" {
		return fmt.Errorf("%w: collection UID is required", ErrInvalidInput)
	}
	if cmd.EnvironmentName == "" && cmd.EnvironmentUID == "" {
		return fmt.Errorf("%w: environment name or UID is required", ErrInvalidInput)
	}

	snap := s.store.Snapshot()
	col := snap.FindCollection(cmd.CollectionUID)
	if col == nil {
		return fmt.Errorf("%w: %s", ErrCollectionNotFound, cmd.CollectionUID)
	}

	var targetEnv *model.Environment
	for _, env := range col.Environments {
		if (cmd.EnvironmentUID != "" && env.UID == cmd.EnvironmentUID) ||
			(cmd.EnvironmentName != "" && strings.EqualFold(env.Name, cmd.EnvironmentName)) {
			targetEnv = env
			break
		}
	}

	if targetEnv == nil {
		return nil // idempotent: already deleted
	}

	// Delete from disk with self-write suppression
	envPath := filepath.Join(col.Path, "environments", targetEnv.Filename)
	collection.SuppressPath(envPath)
	if err := os.Remove(envPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("commands: delete environment file %s: %w", envPath, err)
	}

	// Update workspace state
	s.store.Apply(func(state *model.WorkspaceState) {
		targetCol := state.FindCollection(cmd.CollectionUID)
		if targetCol == nil {
			return
		}

		var remaining []*model.Environment
		for _, env := range targetCol.Environments {
			if env.UID != targetEnv.UID {
				remaining = append(remaining, env)
			}
		}
		targetCol.Environments = remaining

		// Unselect if this was the active environment
		if strings.EqualFold(targetCol.ActiveEnvironment, targetEnv.Name) {
			targetCol.ActiveEnvironment = ""
		}
	})

	// Publish CollectionChanged event (Self, Removed)
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin:  events.OriginSelf,
			Removed: []string{targetEnv.UID},
		},
	})

	return nil
}
