package commands

import (
	"context"
	"fmt"
	"strings"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// SelectEnvironment switches the active environment for a collection.
//
// Passing an empty EnvironmentName clears the active environment (No Environment).
//
// Idempotency: Selecting the already-active environment is a safe no-op.
func (s *CollectionService) SelectEnvironment(ctx context.Context, cmd SelectEnvironmentCmd) error {
	if cmd.CollectionUID == "" {
		return fmt.Errorf("%w: collection UID is required", ErrInvalidInput)
	}

	snap := s.store.Snapshot()
	col := snap.FindCollection(cmd.CollectionUID)
	if col == nil {
		return fmt.Errorf("%w: %s", ErrCollectionNotFound, cmd.CollectionUID)
	}

	envName := strings.TrimSpace(cmd.EnvironmentName)

	// Idempotency check: already active
	if col.ActiveEnvironment == envName {
		return nil
	}

	var targetUID string
	if envName != "" {
		found := false
		for _, env := range col.Environments {
			if strings.EqualFold(env.Name, envName) {
				found = true
				targetUID = env.UID
				envName = env.Name // use canonical case
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: %s", ErrEnvironmentNotFound, envName)
		}
	}

	// Update workspace state
	s.store.Apply(func(state *model.WorkspaceState) {
		targetCol := state.FindCollection(cmd.CollectionUID)
		if targetCol != nil {
			targetCol.ActiveEnvironment = envName
		}
	})

	// Publish CollectionChanged event (Self, Modified)
	var modified []string
	if targetUID != "" {
		modified = []string{targetUID}
	}
	s.publish(events.CollectionChanged{
		CollectionUID: cmd.CollectionUID,
		Change: events.TreeChange{
			Origin:   events.OriginSelf,
			Modified: modified,
		},
	})

	return nil
}
