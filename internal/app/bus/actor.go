package bus

import (
	"context"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// StateActor is the interface commands and presenters depend on.
//
// It is declared here, next to the implementation, because both layers consume
// it: commands write through Apply, presenters read through Snapshot and
// Subscribe. Keeping the interface small is what lets a test substitute a fake
// actor without touching the GUI.
//
// Note: earlier design notes wrote Apply as taking a *model.Draft. No such type
// exists; see decisions D-007. The actor owns the whole workspace state, which
// is what go-architecture.md §3 describes.
type StateActor interface {
	// Snapshot returns the current published state. It is copy-on-write: the
	// returned value never changes, so a reader may hold it for a frame. It must
	// be treated as read-only.
	Snapshot() *model.WorkspaceState
	// Apply runs mutate on the actor goroutine and returns when it has been
	// applied. Mutators are serialised, so an unlocked read-modify-write inside
	// one is safe.
	Apply(mutate func(*model.WorkspaceState))
	// Subscribe returns the event stream and a cancel function.
	Subscribe() (<-chan events.Event, func())
}

// Ensure Store implements the interface.
var _ StateActor = (*Store)(nil)

// ApplyContext is implemented by Store but deliberately kept out of StateActor:
// most callers have no cancellation to offer, and widening the interface would
// force every fake to implement it.
var (
	_ interface {
		ApplyContext(context.Context, func(*model.WorkspaceState)) error
	} = (*Store)(nil)
)
