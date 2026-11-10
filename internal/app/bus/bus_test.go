package bus

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"fearless/internal/core/events"
)

// drain reads everything currently queued without blocking.
func drain(ch <-chan events.Event) []events.Event {
	var out []events.Event
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, ev)
		default:
			return out
		}
	}
}

func TestPublishReachesEverySubscriber(t *testing.T) {
	b := New()
	defer b.Close()

	a, cancelA := b.Subscribe()
	defer cancelA()
	c, cancelC := b.Subscribe()
	defer cancelC()

	b.Publish(events.RequestQueued{UID: "r1"})

	for name, ch := range map[string]<-chan events.Event{"a": a, "c": c} {
		got := drain(ch)
		if len(got) != 1 {
			t.Fatalf("subscriber %s got %d events, want 1", name, len(got))
		}
		if got[0].Kind() != events.KindRequestQueued {
			t.Errorf("subscriber %s got kind %v", name, got[0].Kind())
		}
	}
}

// TestOrderingIsFIFOPerSubscriber covers the "per-request FIFO" rule.
func TestOrderingIsFIFOPerSubscriber(t *testing.T) {
	b := New()
	defer b.Close()

	ch, cancel := b.Subscribe()
	defer cancel()

	const n = 50
	for i := 0; i < n; i++ {
		b.Publish(events.ResponseChunk{UID: "r", Seq: uint64(i)})
	}

	got := drain(ch)
	if len(got) != n {
		t.Fatalf("got %d events, want %d", len(got), n)
	}
	for i, ev := range got {
		chunk := ev.(events.ResponseChunk)
		if chunk.Seq != uint64(i) {
			t.Fatalf("event %d has Seq %d; ordering broken", i, chunk.Seq)
		}
	}
}

// TestPublishNeverBlocks is the core back-pressure guarantee: a stalled
// subscriber must not stall the producer.
func TestPublishNeverBlocks(t *testing.T) {
	old := DefaultBufferSize
	DefaultBufferSize = 2
	defer func() { DefaultBufferSize = old }()

	b := New()
	defer b.Close()

	ch, cancel := b.Subscribe()
	defer cancel()
	_ = ch // never read: the queue fills and stays full

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			b.Publish(events.ResponseChunk{Seq: uint64(i)})
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a full subscriber queue")
	}

	if got := b.Stats().Dropped; got == 0 {
		t.Error("expected droppable chunks to be counted as dropped")
	}
}

// TestDroppableEventsAreDiscardedSilently checks chunks are lost rather than
// queued unboundedly.
func TestDroppableEventsAreDiscardedSilently(t *testing.T) {
	old := DefaultBufferSize
	DefaultBufferSize = 1
	defer func() { DefaultBufferSize = old }()

	b := New()
	defer b.Close()

	ch, cancel := b.Subscribe()
	defer cancel()

	b.Publish(events.ResponseChunk{Seq: 1}) // fills the queue
	b.Publish(events.ResponseChunk{Seq: 2}) // dropped

	if got := len(drain(ch)); got != 1 {
		t.Fatalf("queued %d chunks, want 1", got)
	}
	if b.SubscriberCount() != 1 {
		t.Error("dropping a chunk must not disconnect the subscriber")
	}
	if b.Stats().Dropped != 1 {
		t.Errorf("Dropped = %d, want 1", b.Stats().Dropped)
	}
}

// TestTerminalEventsAreNeverDropped is the counterpart rule: a lost terminal
// event would leave a request pending forever.
func TestTerminalEventsAreNeverDropped(t *testing.T) {
	old := DefaultBufferSize
	DefaultBufferSize = 1
	defer func() { DefaultBufferSize = old }()

	b := New()
	defer b.Close()

	ch, cancel := b.Subscribe()
	defer cancel()

	b.Publish(events.ResponseChunk{Seq: 1})             // fills the queue
	received := events.ResponseReceived{UID: "r"}       //
	b.Publish(received)                                 // cannot be queued
	b.Publish(events.ErrorOccurred{UID: "r", Msg: "x"}) // channel already closed

	// The subscriber that could not take a terminal event is disconnected: its
	// channel closes so its consumer terminates rather than hanging.
	got := drain(ch)
	if len(got) != 1 {
		t.Fatalf("got %d events, want the single chunk", len(got))
	}
	if b.Stats().Disconnected != 1 {
		t.Errorf("Disconnected = %d, want 1", b.Stats().Disconnected)
	}
	if b.SubscriberCount() != 0 {
		t.Errorf("overflowing subscriber should be removed, count = %d", b.SubscriberCount())
	}
}

// TestSlowSubscriberDoesNotAffectFastSubscriber makes sure a dropped chunk for
// one subscriber does not cost another its event.
func TestSlowSubscriberDoesNotAffectFastSubscriber(t *testing.T) {
	old := DefaultBufferSize
	DefaultBufferSize = 1
	defer func() { DefaultBufferSize = old }()

	b := New()
	defer b.Close()

	slow, cancelSlow := b.Subscribe()
	defer cancelSlow()
	_ = slow

	fast, cancelFast := b.Subscribe()
	defer cancelFast()

	b.Publish(events.ResponseChunk{Seq: 1})
	// Read the fast one so it always has room.
	drain(fast)
	b.Publish(events.ResponseChunk{Seq: 2})

	got := drain(fast)
	if len(got) != 1 {
		t.Fatalf("fast subscriber got %d events, want 1", len(got))
	}
}

func TestCancelIsIdempotentAndClosesChannel(t *testing.T) {
	b := New()
	defer b.Close()

	ch, cancel := b.Subscribe()
	cancel()
	cancel()
	cancel()

	if _, ok := <-ch; ok {
		t.Error("channel should be closed after cancel")
	}
	if b.SubscriberCount() != 0 {
		t.Errorf("subscriber count = %d after cancel", b.SubscriberCount())
	}
}

// TestPublishAfterCancelDoesNotPanic guards the send-on-closed-channel hazard.
func TestPublishAfterCancelDoesNotPanic(t *testing.T) {
	b := New()
	defer b.Close()

	_, cancel := b.Subscribe()
	cancel()
	b.Publish(events.RequestQueued{UID: "after-cancel"})
}

func TestCloseClosesAllSubscribers(t *testing.T) {
	b := New()
	var chans []<-chan events.Event
	for i := 0; i < 3; i++ {
		ch, _ := b.Subscribe()
		chans = append(chans, ch)
	}

	b.Close()
	b.Close() // idempotent

	for i, ch := range chans {
		if _, ok := <-ch; ok {
			t.Errorf("subscriber %d channel not closed", i)
		}
	}
	if b.SubscriberCount() != 0 {
		t.Errorf("subscriber count = %d after Close", b.SubscriberCount())
	}
}

func TestPublishAfterCloseIsNoop(t *testing.T) {
	b := New()
	b.Close()
	b.Publish(events.RequestQueued{UID: "x"}) // must not panic
}

func TestSubscribeAfterCloseYieldsClosedChannel(t *testing.T) {
	b := New()
	b.Close()

	ch, cancel := b.Subscribe()
	defer cancel()
	if _, ok := <-ch; ok {
		t.Error("a subscription to a closed bus should be closed immediately")
	}
}

func TestPublishNilIsIgnored(t *testing.T) {
	b := New()
	defer b.Close()
	b.Publish(nil) // must not panic

	ch, cancel := b.Subscribe()
	defer cancel()
	b.Publish(nil)
	if got := drain(ch); len(got) != 0 {
		t.Errorf("nil event was delivered: %v", got)
	}
}

func TestConcurrentPublishAndSubscribe(t *testing.T) {
	b := New()
	defer b.Close()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Publishers.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					b.Publish(events.ResponseChunk{UID: fmt.Sprintf("p%d", id)})
				}
			}
		}(i)
	}

	// Subscribers that come and go while publishing continues.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			ch, cancel := b.Subscribe()
			select {
			case <-ch:
			case <-time.After(time.Millisecond):
			}
			cancel()
		}
	}()

	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()

	if n := b.SubscriberCount(); n != 0 {
		t.Errorf("leaked %d subscribers", n)
	}
}

func TestStatsStartAtZero(t *testing.T) {
	b := New()
	defer b.Close()
	s := b.Stats()
	if s.Dropped != 0 || s.Disconnected != 0 {
		t.Errorf("fresh stats = %+v", s)
	}
}
