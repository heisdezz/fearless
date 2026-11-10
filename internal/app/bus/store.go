package bus

import (
	"context"
	"sync"
	"sync/atomic"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// Store is the single-writer state actor.
//
// One goroutine owns the mutable state. Writers submit a mutator through Apply
// and are acknowledged when it has run; readers take an immutable snapshot that
// no later mutation can affect. This reproduces Bruno's Redux store semantics
// without reducers, middleware or action types.
//
// Snapshots are deep copies. That costs an allocation per read, which is the
// deliberate trade: a shallow copy would let a presenter mutate state it does
// not own, and the resulting bugs are far more expensive than the copy. The UI
// reads one snapshot per frame, not per widget.
type Store struct {
	// current holds the latest published snapshot as an immutable value.
	current atomic.Pointer[model.WorkspaceState]

	// ops serialises mutators. Buffered so a burst of commands does not queue
	// behind the actor's own bookkeeping.
	ops chan operation

	// quit stops the actor loop.
	quit chan struct{}
	// done closes when the loop has exited. Distinct from quit because quit
	// means "stop" and done means "stopped": ApplyContext waits on done so a
	// caller cannot hang after Close.
	done      chan struct{}
	closeOnce sync.Once

	// bus publishes change notifications.
	bus *Bus
}

// operation is one unit of work for the actor.
type operation struct {
	mutate func(*model.WorkspaceState)
	done   chan struct{}
}

// NewStore starts an actor over initialState and returns it.
//
// The initial state is deep-copied so the caller retains ownership of the value
// it passed in. If initialState is nil, an empty state is used.
func NewStore(initialState *model.WorkspaceState, b *Bus) *Store {
	state := cloneState(initialState)
	if state == nil {
		state = &model.WorkspaceState{}
	}
	s := &Store{
		ops:  make(chan operation, 64),
		quit: make(chan struct{}),
		done: make(chan struct{}),
		bus:  b,
	}
	s.current.Store(state)
	go s.run()
	return s
}

// run is the owner goroutine: every mutation happens here and nowhere else.
func (s *Store) run() {
	defer close(s.done)
	for {
		select {
		case op := <-s.ops:
			s.apply(op)
		case <-s.quit:
			// Drain anything already queued so a mutator accepted before Close
			// still completes and its caller does not block forever.
			for {
				select {
				case op := <-s.ops:
					s.apply(op)
				default:
					return
				}
			}
		}
	}
}

// apply runs one mutator against a private copy and publishes the result.
//
// Mutating a copy rather than the live snapshot is what makes published
// snapshots immutable: a reader holding the previous pointer never observes a
// change.
func (s *Store) apply(op operation) {
	prev := s.current.Load()
	next := cloneState(prev)
	if next == nil {
		next = &model.WorkspaceState{}
	}
	op.mutate(next)
	s.current.Store(next)

	if op.done != nil {
		close(op.done)
	}
}

// Apply runs mutate against the state and returns when it has been applied.
//
// It blocks on the actor, so it must not be called from the actor goroutine
// itself. Pass a nil-aware mutator: the state is never nil.
func (s *Store) Apply(mutate func(*model.WorkspaceState)) {
	s.ApplyContext(context.Background(), mutate)
}

// ApplyContext is Apply with cancellation.
//
// It returns ctx.Err() if the context is done before the actor accepts the work.
// The mutator is not started in that case. Once started, a mutator always runs to
// completion: a half-applied state is worse than a slow one.
func (s *Store) ApplyContext(ctx context.Context, mutate func(*model.WorkspaceState)) error {
	if mutate == nil {
		return nil
	}

	// Fail fast once the actor has stopped. Without this the send below could
	// still succeed into the buffered channel and then wait forever for a
	// mutator that no goroutine will ever run.
	select {
	case <-s.done:
		return errClosed
	default:
	}

	op := operation{mutate: mutate, done: make(chan struct{})}
	select {
	case s.ops <- op:
	case <-s.done:
		return errClosed
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-op.done:
		return nil
	case <-s.done:
		// The actor exited before running this operation. If it did run,
		// op.done is closed and the first case wins; either way we do not hang.
		select {
		case <-op.done:
			return nil
		default:
			return errClosed
		}
	case <-ctx.Done():
		// The work may still complete; we simply stop waiting for it.
		return ctx.Err()
	}
}

// Snapshot returns the current immutable state.
//
// The returned value must be treated as read-only. It stays valid indefinitely:
// later mutations publish a new value rather than editing this one.
func (s *Store) Snapshot() *model.WorkspaceState {
	return s.current.Load()
}

// Subscribe attaches to the store's bus.
//
// A store created with a nil bus returns a closed channel, so callers can always
// range over the result without a nil check.
func (s *Store) Subscribe() (<-chan events.Event, func()) {
	if s.bus == nil {
		ch := make(chan events.Event)
		close(ch)
		return ch, func() {}
	}
	return s.bus.Subscribe()
}

// Publish sends an event through the store's bus.
func (s *Store) Publish(ev events.Event) {
	if s.bus != nil {
		s.bus.Publish(ev)
	}
}

// Close stops the actor after draining queued work. It is idempotent and safe to
// call concurrently. Reads after Close keep working; Apply returns errClosed.
func (s *Store) Close() {
	s.closeOnce.Do(func() {
		close(s.quit)
	})
	<-s.done
}
