package bus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

func TestStoreStartsFromInitialState(t *testing.T) {
	initial := &model.WorkspaceState{ActiveTab: "t1"}
	s := NewStore(initial, nil)
	defer s.Close()

	if got := s.Snapshot().ActiveTab; got != "t1" {
		t.Errorf("ActiveTab = %q", got)
	}
	// The caller keeps ownership of the value it passed in.
	initial.ActiveTab = "mutated"
	if got := s.Snapshot().ActiveTab; got != "t1" {
		t.Error("store aliased the caller's initial state")
	}
}

func TestNewStoreWithNilState(t *testing.T) {
	s := NewStore(nil, nil)
	defer s.Close()
	if s.Snapshot() == nil {
		t.Fatal("snapshot must never be nil")
	}
}

func TestApplyMutatesAndPublishes(t *testing.T) {
	s := NewStore(nil, nil)
	defer s.Close()

	before := s.Snapshot()
	s.Apply(func(st *model.WorkspaceState) { st.ActiveTab = "abc" })

	after := s.Snapshot()
	if after.ActiveTab != "abc" {
		t.Fatalf("ActiveTab = %q", after.ActiveTab)
	}
	if before == after {
		t.Fatal("mutating must publish a new snapshot, not edit the old one")
	}
	if before.ActiveTab != "" {
		t.Error("the previously published snapshot must be unaffected")
	}
}

// TestSnapshotIsCopyOnWrite pins the actual contract: a published snapshot is
// never mutated by the store, so a reader that holds one keeps seeing a
// consistent view. It is *not* defensively copied on every read — that would
// cost a deep copy per frame. Readers must treat it as read-only.
//
// What must hold is that later mutations do not alter an earlier snapshot, and
// that mutating through Apply never touches the caller's initial value.
func TestSnapshotIsCopyOnWrite(t *testing.T) {
	initial := &model.WorkspaceState{
		Tabs:        []*model.Tab{{UID: "t1", Title: "Original"}},
		Collections: []*model.Collection{{UID: "c1", Name: "Coll", Items: []*model.Item{{UID: "i1", Name: "Item"}}}},
		Responses:   map[string]*model.Result{"i1": {Status: 200}},
		Pending:     map[string]bool{"i1": true},
	}
	s := NewStore(initial, nil)
	defer s.Close()

	// A reader takes a snapshot and keeps it.
	snap := s.Snapshot()

	// The store then evolves.
	s.Apply(func(st *model.WorkspaceState) {
		st.ActiveTab = "t2"
		st.Tabs[0].Title = "Changed"
		st.Collections[0].Name = "Changed"
		st.Collections[0].Items[0].Name = "Changed"
		st.Responses["i1"].Status = 500
		st.Pending["i1"] = false
	})

	// The held snapshot is unchanged: it is a distinct value, not a live view.
	if snap.ActiveTab != "" {
		t.Errorf("older snapshot saw a later ActiveTab: %q", snap.ActiveTab)
	}
	if snap.Tabs[0].Title != "Original" {
		t.Errorf("older snapshot saw a later tab title: %q", snap.Tabs[0].Title)
	}
	if snap.Collections[0].Name != "Coll" {
		t.Errorf("older snapshot saw a later collection name: %q", snap.Collections[0].Name)
	}
	if snap.Collections[0].Items[0].Name != "Item" {
		t.Errorf("older snapshot saw a later item name: %q", snap.Collections[0].Items[0].Name)
	}
	if snap.Responses["i1"].Status != 200 {
		t.Errorf("older snapshot saw a later status: %d", snap.Responses["i1"].Status)
	}
	if !snap.Pending["i1"] {
		t.Error("older snapshot saw a later pending flag")
	}

	// The caller's initial value is never aliased by the store.
	if initial.ActiveTab != "" {
		t.Errorf("store aliased the caller's initial state: %q", initial.ActiveTab)
	}
	if initial.Tabs[0].Title != "Original" {
		// The store deep-copied on construction, so the caller's own value must
		// not have been touched either.
		t.Errorf("caller's tab was mutated: %q", initial.Tabs[0].Title)
	}
}

// TestDraftAndSavedAreIndependent checks that cloning keeps the draft duality
// intact: editing a draft must not touch the saved request.
func TestDraftAndSavedAreIndependent(t *testing.T) {
	saved := &model.Request{URL: "https://example.test/saved", Method: "GET"}
	s := NewStore(&model.WorkspaceState{
		Collections: []*model.Collection{{
			UID:   "c1",
			Items: []*model.Item{{UID: "i1", Request: saved, Draft: saved.Clone()}},
		}},
	}, nil)
	defer s.Close()

	s.Apply(func(st *model.WorkspaceState) {
		st.Collections[0].Items[0].Draft.URL = "https://example.test/draft"
	})

	item := s.Snapshot().Collections[0].Items[0]
	if item.Draft.URL != "https://example.test/draft" {
		t.Errorf("draft URL = %q", item.Draft.URL)
	}
	if item.Request.URL != "https://example.test/saved" {
		t.Errorf("saved URL changed to %q", item.Request.URL)
	}
	if saved.URL != "https://example.test/saved" {
		t.Error("the original request value was mutated")
	}
}

// TestApplyIsSerialised proves the single-owner rule: concurrent mutators never
// interleave, so a read-modify-write is safe without a lock at the call site.
func TestApplyIsSerialised(t *testing.T) {
	s := NewStore(nil, nil)
	defer s.Close()

	const writers = 8
	const perWriter = 100
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWriter; j++ {
				// Unlocked read-modify-write: correct only because Apply
				// serialises. A lost update would show as a low count.
				s.Apply(func(st *model.WorkspaceState) {
					st.SidebarWidth++
				})
			}
		}()
	}
	wg.Wait()

	if got := s.Snapshot().SidebarWidth; got != writers*perWriter {
		t.Fatalf("SidebarWidth = %v, want %d; updates were lost",
			got, writers*perWriter)
	}
}

func TestApplyNilMutatorIsNoop(t *testing.T) {
	s := NewStore(nil, nil)
	defer s.Close()
	s.Apply(nil) // must not panic or deadlock
}

func TestApplyContextCancelled(t *testing.T) {
	s := NewStore(nil, nil)
	defer s.Close()

	// Saturate the queue with a mutator that blocks until released, so a second
	// Apply cannot be accepted.
	release := make(chan struct{})
	blocking := make(chan struct{})
	go func() {
		close(blocking)
		_ = s.ApplyContext(context.Background(), func(*model.WorkspaceState) { <-release })
	}()
	<-blocking
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := s.ApplyContext(ctx, func(*model.WorkspaceState) {})

	close(release)

	if err == nil {
		t.Skip("queue accepted the work before cancellation; timing-dependent")
	}
	if err != context.Canceled {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestApplyAfterCloseReturnsError(t *testing.T) {
	s := NewStore(nil, nil)
	s.Close()

	err := s.ApplyContext(context.Background(), func(*model.WorkspaceState) {})
	if !IsClosed(err) {
		t.Fatalf("err = %v, want the closed sentinel", err)
	}
}

func TestSnapshotWorksAfterClose(t *testing.T) {
	s := NewStore(&model.WorkspaceState{ActiveTab: "t1"}, nil)
	s.Close()
	if got := s.Snapshot().ActiveTab; got != "t1" {
		t.Errorf("Snapshot after Close = %q", got)
	}
}

func TestCloseIsIdempotentAndDrains(t *testing.T) {
	s := NewStore(nil, nil)

	var applied atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.ApplyContext(context.Background(), func(*model.WorkspaceState) {
				applied.Add(1)
			}); err == nil {
				return
			}
		}()
	}
	wg.Wait()
	s.Close()
	s.Close()

	if got := applied.Load(); got != 20 {
		t.Errorf("applied %d mutators, want 20", got)
	}
}

// TestStorePublishesThroughBus checks the store is the single entry point for
// both state and events in the app layer.
func TestStorePublishesThroughBus(t *testing.T) {
	b := New()
	defer b.Close()
	s := NewStore(nil, b)
	defer s.Close()

	ch, cancel := s.Subscribe()
	defer cancel()

	s.Publish(events.RequestQueued{UID: "r1"})

	got := drain(ch)
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
}

func TestStoreWithNilBusSubscribeIsClosed(t *testing.T) {
	s := NewStore(nil, nil)
	defer s.Close()

	ch, cancel := s.Subscribe()
	defer cancel()
	if _, ok := <-ch; ok {
		t.Error("a store without a bus should return a closed channel")
	}
	s.Publish(events.RequestQueued{}) // must not panic
}

func TestCloneResultCopiesInlineBody(t *testing.T) {
	s := NewStore(&model.WorkspaceState{
		Responses: map[string]*model.Result{
			"i1": {Body: model.BodyRef{Inline: []byte("payload"), Size: 7}},
		},
	}, nil)
	defer s.Close()

	// A mutator editing the body must not reach the caller's original value, and
	// an older snapshot must keep its own bytes.
	snap := s.Snapshot()
	s.Apply(func(st *model.WorkspaceState) {
		st.Responses["i1"].Body.Inline[0] = 'X'
	})

	if got := string(snap.Responses["i1"].Body.Inline); got != "payload" {
		t.Errorf("older snapshot's body changed: %q", got)
	}
	if got := string(s.Snapshot().Responses["i1"].Body.Inline); got != "Xayload" {
		t.Errorf("the mutator's change was lost: %q", got)
	}
}
