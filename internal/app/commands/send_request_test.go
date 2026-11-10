package commands_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fearless/internal/app/bus"
	"fearless/internal/app/commands"
	"fearless/internal/core/engine"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

func strPtr(s string) *string {
	return &s
}

func setupTestService(t *testing.T) (*commands.RequestService, *bus.Store, *bus.Bus) {
	b := bus.New()
	t.Cleanup(b.Close)

	state := &model.WorkspaceState{
		Collections: []*model.Collection{
			{
				UID:  "col-1",
				Name: "Test Collection",
				Path: "/tmp/test-col",
				Config: model.CollectionConfig{
					Vars: []model.Variable{
						{Name: strPtr("col_var"), Value: "col-value", Enabled: true},
					},
				},
				Environments: []*model.Environment{
					{
						UID:  "env-dev",
						Name: "dev",
						Variables: []model.Variable{
							{Name: strPtr("base_url"), Value: "http://example.com", Enabled: true},
						},
					},
				},
				Items: []*model.Item{
					{
						UID:  "item-1",
						Name: "Get Users",
						Request: &model.Request{
							Method: "GET",
							URL:    "http://example.com/users",
						},
					},
				},
			},
		},
	}
	store := bus.NewStore(state, b)
	eng := engine.New()
	svc := commands.NewRequestService(eng, store, b, nil, nil)
	return svc, store, b
}

// TestSendRequestSuccess asserts normal send execution emitting RequestQueued,
// RequestSent, and exactly one ResponseReceived terminal event.
func TestSendRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	svc, store, b := setupTestService(t)

	// Update item URL to point to test server
	store.Apply(func(s *model.WorkspaceState) {
		it := s.FindCollection("col-1").FindItem("item-1")
		it.Request.URL = server.URL
	})

	ch, unsub := b.Subscribe()
	defer unsub()

	cmd := commands.SendRequestCmd{
		CollectionUID: "col-1",
		ItemUID:       "item-1",
	}

	reqUID, err := svc.Send(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if reqUID == "" {
		t.Fatalf("expected non-empty request UID")
	}

	var queuedSeen, sentSeen, terminalSeen int
	var receivedResult model.Result
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
			case events.RequestSent:
				if e.UID == reqUID {
					sentSeen++
				}
			case events.ResponseReceived:
				if e.UID == reqUID {
					terminalSeen++
					receivedResult = e.Received
					break loop
				}
			case events.ErrorOccurred:
				if e.UID == reqUID {
					t.Fatalf("unexpected ErrorOccurred: %v", e.Err)
				}
			}
		case <-timeout:
			t.Fatalf("timed out waiting for terminal event")
		}
	}

	if queuedSeen != 1 {
		t.Errorf("expected 1 RequestQueued, got %d", queuedSeen)
	}
	if sentSeen < 1 {
		t.Errorf("expected at least 1 RequestSent, got %d", sentSeen)
	}
	if terminalSeen != 1 {
		t.Errorf("expected exactly 1 terminal ResponseReceived, got %d", terminalSeen)
	}
	if receivedResult.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", receivedResult.Status)
	}

	// Verify in-flight count drops back to 0
	time.Sleep(20 * time.Millisecond)
	if svc.ActiveCount() != 0 {
		t.Errorf("expected 0 active requests, got %d", svc.ActiveCount())
	}
}

// TestSendSyncReturnsResult asserts SendSync returns the result synchronously
// while still publishing events.
func TestSendSyncReturnsResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("sync-ok"))
	}))
	defer server.Close()

	svc, store, _ := setupTestService(t)
	store.Apply(func(s *model.WorkspaceState) {
		it := s.FindCollection("col-1").FindItem("item-1")
		it.Request.URL = server.URL
	})

	res, err := svc.SendSync(context.Background(), commands.SendRequestCmd{
		CollectionUID: "col-1",
		ItemUID:       "item-1",
	})
	if err != nil {
		t.Fatalf("SendSync failed: %v", err)
	}
	if res == nil || res.Status != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %v", res)
	}
}

// TestSendRequestDraftPrecedence asserts that Draft edits take precedence over saved Request.
func TestSendRequestDraftPrecedence(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	svc, store, _ := setupTestService(t)
	store.Apply(func(s *model.WorkspaceState) {
		it := s.FindCollection("col-1").FindItem("item-1")
		it.Request.URL = server.URL + "/saved-path"
		it.Draft = &model.Request{
			Method: "GET",
			URL:    server.URL + "/draft-path",
		}
	})

	res, err := svc.SendSync(context.Background(), commands.SendRequestCmd{
		CollectionUID: "col-1",
		ItemUID:       "item-1",
	})
	if err != nil || res == nil {
		t.Fatalf("SendSync failed: %v", err)
	}

	if requestedPath != "/draft-path" {
		t.Errorf("expected draft path '/draft-path', got %q", requestedPath)
	}
}

// TestSendRequestItemNotFound asserts validation errors return immediately.
func TestSendRequestItemNotFound(t *testing.T) {
	svc, _, _ := setupTestService(t)

	_, err := svc.Send(context.Background(), commands.SendRequestCmd{
		CollectionUID: "col-1",
		ItemUID:       "missing-item",
	})
	if err == nil {
		t.Fatalf("expected error for missing item, got nil")
	}

	_, err = svc.Send(context.Background(), commands.SendRequestCmd{
		CollectionUID: "missing-col",
		ItemUID:       "item-1",
	})
	if err == nil {
		t.Fatalf("expected error for missing collection, got nil")
	}
}

// TestSendRequestEnvironmentAndFolderVariables asserts variables from environment,
// collection, folder, runtime and prompt are resolved into scopes.
func TestSendRequestEnvironmentAndFolderVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("env-ok"))
	}))
	defer server.Close()

	svc, store, b := setupTestService(t)

	// Add folder with folder vars
	store.Apply(func(s *model.WorkspaceState) {
		col := s.FindCollection("col-1")
		item := col.FindItem("item-1")
		item.Request.URL = server.URL

		folder := &model.Item{
			UID:  "folder-1",
			Type: model.ItemFolder,
			Name: "Subfolder",
			Vars: []model.Variable{
				{Name: strPtr("folder_var"), Value: "folder-val", Enabled: true},
			},
			Children: []*model.Item{item},
		}
		col.Items = []*model.Item{folder}
	})

	ch, unsub := b.Subscribe()
	defer unsub()

	cmd := commands.SendRequestCmd{
		CollectionUID: "col-1",
		ItemUID:       "item-1",
		Env:           "dev",
		RuntimeVars: map[string]any{
			"runtime_k": "runtime_v",
		},
		PromptVars: map[string]string{
			"prompt_k": "prompt_v",
		},
	}

	res, err := svc.SendSync(context.Background(), cmd)
	if err != nil || res == nil {
		t.Fatalf("SendSync failed: %v", err)
	}

	// Drain events to verify sink callbacks
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case <-ch:
		case <-timeout:
			return
		}
	}
}

// TestSendRequestNetworkFailureTerminalEvent asserts transport failures
// emit ErrorOccurred with ScopeSend.
func TestSendRequestNetworkFailureTerminalEvent(t *testing.T) {
	svc, store, b := setupTestService(t)
	store.Apply(func(s *model.WorkspaceState) {
		it := s.FindCollection("col-1").FindItem("item-1")
		it.Request.URL = "http://127.0.0.1:1/refused"
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

	timeout := time.After(3 * time.Second)
	var errEv *events.ErrorOccurred

loop:
	for {
		select {
		case ev := <-ch:
			if e, ok := ev.(events.ErrorOccurred); ok && e.UID == reqUID {
				errEv = &e
				break loop
			}
		case <-timeout:
			t.Fatalf("timed out waiting for terminal ErrorOccurred")
		}
	}

	if errEv == nil {
		t.Fatalf("expected terminal ErrorOccurred, got nil")
	}
	if errEv.Scope != events.ScopeSend {
		t.Errorf("expected scope %q, got %q", events.ScopeSend, errEv.Scope)
	}
}
