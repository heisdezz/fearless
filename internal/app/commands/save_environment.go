package commands

import (
	"context"
	"fmt"
	"strings"

	"fearless/internal/core/collection"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// SaveEnvironment writes modifications to an environment to disk atomically and updates
// the environment in workspace state.
//
// Idempotency: Saving unchanged environment state writes cleanly without side-effects.
func (s *CollectionService) SaveEnvironment(ctx context.Context, cmd SaveEnvironmentCmd) error {
	if cmd.CollectionUID == "" || cmd.Environment == nil {
		return fmt.Errorf("%w: collection UID and environment are required", ErrInvalidInput)
	}
	if strings.TrimSpace(cmd.Environment.Name) == "" {
		return fmt.Errorf("%w: environment name is required", ErrInvalidInput)
	}

	snap := s.store.Snapshot()
	col := snap.FindCollection(cmd.CollectionUID)
	if col == nil {
		return fmt.Errorf("%w: %s", ErrCollectionNotFound, cmd.CollectionUID)
	}

	// Persist to disk atomically using core collection helper with self-write suppression
	if err := collection.SaveEnvironment(col, cmd.Environment); err != nil {
		return fmt.Errorf("commands: write environment %s to disk: %w", cmd.Environment.Name, err)
	}

	// Update workspace state
	s.store.Apply(func(state *model.WorkspaceState) {
		targetCol := state.FindCollection(cmd.CollectionUID)
		if targetCol == nil {
			return
		}

		replaced := false
		for i, env := range targetCol.Environments {
			if (cmd.Environment.UID != "" && env.UID == cmd.Environment.UID) ||
				strings.EqualFold(env.Name, cmd.Environment.Name) {
				targetCol.Environments[i] = cmd.Environment
				replaced = true
				break
			}
		}
		if !replaced {
			targetCol.Environments = append(targetCol.Environments, cmd.Environment)
		}
	})

	// Publish CollectionChanged event (Self, Modified)
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin:   events.OriginSelf,
			Modified: []string{cmd.Environment.UID},
		},
	})

	return nil
}
