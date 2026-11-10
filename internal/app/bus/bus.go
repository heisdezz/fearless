// Package bus carries events from the application layer to the presentation
// layer, and owns the single-writer state actor.
//
// It replaces the IPC channel that separates Bruno's renderer from its main
// process. The contract (docs/implementation/go-architecture.md §4):
//
//   - Per-request ordering is preserved; global ordering is not guaranteed.
//   - Publish never blocks the producer. A slow subscriber loses droppable
//     events (chunks, console lines) rather than stalling execution.
//   - Terminal events and state transitions are never dropped. A subscriber that
//     falls too far behind is disconnected instead, which is loud rather than
//     quietly lossy.
//   - Events are values; no pointer into core-owned state crosses the boundary.
//
// The bus is deliberately framework-agnostic and free of GUI imports: presenters
// subscribe to it from the UI layer, but the bus itself knows nothing about them.
package bus

import (
	"sync"
	"sync/atomic"

	"fearless/internal/core/events"
)

// DefaultBufferSize is the per-subscriber queue depth.
//
// 256 is enough to absorb a burst of chunks between frames while keeping the
// memory cost trivial. It is a variable rather than a constant only so tests can
// shrink it to force the overflow paths.
var DefaultBufferSize = 256

// Bus fans events out to subscribers.
//
// A Bus is safe for concurrent use by any number of publishers and subscribers.
type Bus struct {
	mu     sync.RWMutex
	subs   map[uint64]*subscription
	nextID atomic.Uint64

	// closed prevents publishing after Close, so a producer cannot write into a
	// torn-down UI.
	closed atomic.Bool

	// dropped counts events discarded because a subscriber could not keep up.
	dropped atomic.Uint64
	// disconnected counts subscribers removed for falling too far behind.
	disconnected atomic.Uint64
}

// New creates a Bus with no subscribers.
func New() *Bus {
	return &Bus{subs: make(map[uint64]*subscription)}
}

// Publish delivers ev to every current subscriber.
//
// It never blocks: each subscriber's queue is non-blocking. A droppable event is
// discarded for a subscriber whose queue is full. A non-droppable event that
// cannot be queued causes that subscriber to be disconnected rather than the
// event to be lost silently — a presenter that missed a state transition would
// render stale data forever, so failing loudly is the safer contract.
//
// Publishing to a closed Bus is a no-op.
func (b *Bus) Publish(ev events.Event) {
	if ev == nil || b.closed.Load() {
		return
	}

	b.mu.RLock()
	targets := make([]*subscription, 0, len(b.subs))
	for _, s := range b.subs {
		targets = append(targets, s)
	}
	b.mu.RUnlock()

	if len(targets) == 0 {
		return
	}

	droppable := ev.Kind().IsDroppable()
	for _, s := range targets {
		switch s.offer(ev, droppable) {
		case offerDropped:
			b.dropped.Add(1)
		case offerOverflow:
			b.remove(s)
			b.disconnected.Add(1)
		}
	}
}

// Subscribe returns a channel of events and a cancel function.
//
// The channel is closed when the subscriber is cancelled or disconnected, so a
// range loop terminates cleanly. Callers must always invoke cancel; a
// subscription that is never cancelled keeps its goroutine and buffer alive.
func (b *Bus) Subscribe() (<-chan events.Event, func()) {
	s := &subscription{
		id: b.nextID.Add(1),
		ch: make(chan events.Event, DefaultBufferSize),
	}
	if b.closed.Load() {
		close(s.ch)
		return s.ch, func() {}
	}

	b.mu.Lock()
	b.subs[s.id] = s
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.remove(s)
		})
	}
	return s.ch, cancel
}

// remove detaches a subscription and closes its channel exactly once.
func (b *Bus) remove(s *subscription) {
	b.mu.Lock()
	_, present := b.subs[s.id]
	delete(b.subs, s.id)
	b.mu.Unlock()

	if present {
		s.shutdown()
	}
}

// SubscriberCount reports how many subscribers are attached. Useful for tests and
// for a health indicator.
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}

// Stats reports delivery problems observed so far.
type Stats struct {
	// Dropped counts droppable events discarded under back-pressure.
	Dropped uint64
	// Disconnected counts subscribers removed for overflowing on a
	// non-droppable event.
	Disconnected uint64
}

// Stats returns a snapshot of delivery counters.
func (b *Bus) Stats() Stats {
	return Stats{
		Dropped:      b.dropped.Load(),
		Disconnected: b.disconnected.Load(),
	}
}

// Close detaches every subscriber and closes their channels.
//
// Publishing afterwards is a no-op. Close is idempotent.
func (b *Bus) Close() {
	if b.closed.Swap(true) {
		return
	}
	b.mu.Lock()
	subs := make([]*subscription, 0, len(b.subs))
	for _, s := range b.subs {
		subs = append(subs, s)
	}
	b.subs = make(map[uint64]*subscription)
	b.mu.Unlock()

	for _, s := range subs {
		s.shutdown()
	}
}

// subscription is one subscriber's queue.
type subscription struct {
	id uint64
	ch chan events.Event

	// mu guards close(s.ch). A plain closed-channel check is not enough: between
	// the check and the send, a concurrent shutdown could close the channel and
	// the send would panic. Holding a read lock across both makes the pair atomic
	// with respect to closing.
	mu     sync.RWMutex
	closed bool
	once   sync.Once
}

// shutdown closes the channel exactly once, so a concurrent cancel and Close
// cannot double-close it.
func (s *subscription) shutdown() {
	s.once.Do(func() {
		s.mu.Lock()
		s.closed = true
		close(s.ch)
		s.mu.Unlock()
	})
}

// offerOutcome describes what happened to an event for one subscriber.
type offerOutcome int

const (
	// offerOK means the event was queued.
	offerOK offerOutcome = iota
	// offerDropped means a droppable event was discarded.
	offerDropped
	// offerOverflow means a non-droppable event could not be queued.
	offerOverflow
)

// offer attempts a non-blocking send.
//
// The read lock is held only for the duration of a non-blocking channel
// operation, so it cannot introduce contention in practice: the send either
// succeeds immediately or falls through to default.
func (s *subscription) offer(ev events.Event, droppable bool) offerOutcome {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return offerDropped
	}

	select {
	case s.ch <- ev:
		return offerOK
	default:
		if droppable {
			return offerDropped
		}
		return offerOverflow
	}
}
