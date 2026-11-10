package commands_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fearless/internal/app/commands"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

// TestCancelRequestByRequestUID asserts Acceptance Criterion:
// A cancelled request yields exactly one terminal event emitted as
// ErrorOccurred{Err: context.Canceled, Scope: ScopeSend, Class: ErrorCancelled}.
func TestCancelRequestByRequestUID(t *testing.T) {
	unblockServer := make(chan struct{})
	serverReceived := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(serverReceived)
		<-unblockServer
		w.WriteHeader(http.StatusOK)
	}))
	defer func() {
		close(unblockServer)
		server.Close()
	}()

	svc, store, b := setupTestService(t)
	store.Apply(func(s *model.WorkspaceState) {
		it := s.FindCollection("col-1").FindItem("item-1")
		it.Request.URL = server.URL
	})

	ch, unsub := b.Subscribe()
	defer unsub()

	reqUID, err := svc.Send(context.Background(), commands.SendRequestCmd{
		CollectionUID: "col-1",
		ItemUID:       "item-1",
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Wait until server starts handling or request is in flight
	select {
	case <-serverReceived:
	case <-time.After(2 * time.Second):
		t.Fatalf("server never received request")
	}

	if !svc.InFlight("item-1") {
		t.Fatalf("expected item-1 to be in flight")
	}

	// Cancel by RequestUID
	cancelled := svc.Cancel(commands.CancelRequestCmd{RequestUID: reqUID})
	if !cancelled {
		t.Fatalf("expected Cancel to return true")
	}

	// Wait for terminal event
	var queuedSeen int
	var terminalError *events.ErrorOccurred
	var terminalResponseSeen int
	timeout := time.After(3 * time.Second)

loop:
	for {
		select {
		case ev := <-ch:
			switch e := ev.(type) {
			case events.RequestQueued:
				if e.UID == reqUID {
					queuedSeen++
				}
			case events.ErrorOccurred:
				if e.UID == reqUID {
					terminalError = &e
					break loop
				}
			case events.ResponseReceived:
				if e.UID == reqUID {
					terminalResponseSeen++
					break loop
				}
			}
		case <-timeout:
			t.Fatalf("timed out waiting for terminal event after cancel")
		}
	}

	if queuedSeen != 1 {
		t.Errorf("expected 1 RequestQueued, got %d", queuedSeen)
	}
	if terminalResponseSeen != 0 {
		t.Errorf("expected 0 ResponseReceived after cancel, got %d", terminalResponseSeen)
	}
	if terminalError == nil {
		t.Fatalf("expected terminal ErrorOccurred, got nil")
	}

	if !errors.Is(terminalError.Err, context.Canceled) {
		t.Errorf("expected context.Canceled in terminalError.Err, got %v", terminalError.Err)
	}
	if terminalError.Scope != events.ScopeSend {
		t.Errorf("expected scope %q, got %q", events.ScopeSend, terminalError.Scope)
	}
	if terminalError.Class != events.ErrorCancelled {
		t.Errorf("expected class %q, got %q", events.ErrorCancelled, terminalError.Class)
	}

	// Verify active count drops back to 0
	time.Sleep(20 * time.Millisecond)
	if svc.ActiveCount() != 0 {
		t.Errorf("expected 0 active requests, got %d", svc.ActiveCount())
	}
}

// TestCancelRequestByItemUID asserts cancellation targeted by ItemUID.
func TestCancelRequestByItemUID(t *testing.T) {
	unblockServer := make(chan struct{})
	serverReceived := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(serverReceived)
		<-unblockServer
		w.WriteHeader(http.StatusOK)
	}))
	defer func() {
		close(unblockServer)
		server.Close()
	}()

	svc, store, b := setupTestService(t)
	store.Apply(func(s *model.WorkspaceState) {
		it := s.FindCollection("col-1").FindItem("item-1")
		it.Request.URL = server.URL
	})

	ch, unsub := b.Subscribe()
	defer unsub()

	reqUID, err := svc.Send(context.Background(), commands.SendRequestCmd{
		CollectionUID: "col-1",
		ItemUID:       "item-1",
	})
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	<-serverReceived

	cancelled := svc.Cancel(commands.CancelRequestCmd{ItemUID: "item-1"})
	if !cancelled {
		t.Fatalf("expected cancel by item UID to return true")
	}

	timeout := time.After(3 * time.Second)
	var terminalError *events.ErrorOccurred

loop:
	for {
		select {
		case ev := <-ch:
			if errEv, ok := ev.(events.ErrorOccurred); ok && errEv.UID == reqUID {
				terminalError = &errEv
				break loop
			}
		case <-timeout:
			t.Fatalf("timed out waiting for cancel terminal event")
		}
	}

	if terminalError == nil || !terminalError.Cancelled() {
		t.Errorf("expected terminal cancelled error, got %+v", terminalError)
	}
}

// TestSendCancelsPriorInFlightItemRequest asserts that sending the same item
// while a prior execution is still in flight automatically aborts the prior one.
func TestSendCancelsPriorInFlightItemRequest(t *testing.T) {
	unblockServer := make(chan struct{})
	firstReceived := make(chan struct{})
	var secondReceived atomicBool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !secondReceived.get() {
			close(firstReceived)
			<-unblockServer
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok-2"))
	}))
	defer func() {
		close(unblockServer)
		server.Close()
	}()

	svc, store, b := setupTestService(t)
	store.Apply(func(s *model.WorkspaceState) {
		it := s.FindCollection("col-1").FindItem("item-1")
		it.Request.URL = server.URL
	})

	ch, unsub := b.Subscribe()
	defer unsub()

	// Launch first request
	firstUID, err := svc.Send(context.Background(), commands.SendRequestCmd{
		CollectionUID: "col-1",
		ItemUID:       "item-1",
	})
	if err != nil {
		t.Fatalf("first Send failed: %v", err)
	}

	<-firstReceived

	// Launch second request for SAME itemUID
	secondReceived.set(true)
	secondUID, err := svc.Send(context.Background(), commands.SendRequestCmd{
		CollectionUID: "col-1",
		ItemUID:       "item-1",
	})
	if err != nil {
		t.Fatalf("second Send failed: %v", err)
	}

	var firstTerminalError *events.ErrorOccurred
	var secondTerminalResponse *events.ResponseReceived
	timeout := time.After(4 * time.Second)

	for firstTerminalError == nil || secondTerminalResponse == nil {
		select {
		case ev := <-ch:
			switch e := ev.(type) {
			case events.ErrorOccurred:
				if e.UID == firstUID {
					firstTerminalError = &e
				}
			case events.ResponseReceived:
				if e.UID == secondUID {
					secondTerminalResponse = &e
				}
			}
		case <-timeout:
			t.Fatalf("timeout: firstTerminalError=%v, secondTerminalResponse=%v", firstTerminalError, secondTerminalResponse)
		}
	}

	if firstTerminalError == nil || !firstTerminalError.Cancelled() {
		t.Errorf("expected first request to be cancelled, got %+v", firstTerminalError)
	}
	if secondTerminalResponse == nil {
		t.Errorf("expected second request to succeed, got nil")
	}
}

// TestCancelNonExistent asserts cancelling a non-existent request returns false.
func TestCancelNonExistent(t *testing.T) {
	svc, _, _ := setupTestService(t)
	if svc.Cancel(commands.CancelRequestCmd{RequestUID: "no-such-req"}) {
		t.Errorf("expected false when cancelling non-existent request UID")
	}
	if svc.Cancel(commands.CancelRequestCmd{ItemUID: "no-such-item"}) {
		t.Errorf("expected false when cancelling non-existent item UID")
	}
}

type atomicBool struct {
	val bool
}

func (a *atomicBool) set(v bool) { a.val = v }
func (a *atomicBool) get() bool  { return a.val }
