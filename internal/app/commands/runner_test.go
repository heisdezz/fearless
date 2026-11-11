package commands_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"fearless/internal/app/commands"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

func setupTestRunnerCollection(serverURL string) *model.Collection {
	return &model.Collection{
		UID:  "runner-col",
		Name: "Runner Test Collection",
		Items: []*model.Item{
			{
				UID:  "item-r1",
				Name: "Request 1",
				Type: model.ItemHTTPRequest,
				Tags: []string{"smoke", "auth"},
				Request: &model.Request{
					Method: "GET",
					URL:    serverURL + "/step-1",
				},
			},
			{
				UID:  "folder-r1",
				Name: "Folder Sub",
				Type: model.ItemFolder,
				Children: []*model.Item{
					{
						UID:  "item-r2",
						Name: "Request 2",
						Type: model.ItemHTTPRequest,
						Tags: []string{"regression"},
						Request: &model.Request{
							Method: "GET",
							URL:    serverURL + "/step-2",
						},
					},
					{
						UID:  "item-r3",
						Name: "Request 3",
						Type: model.ItemHTTPRequest,
						Tags: []string{"smoke"},
						Request: &model.Request{
							Method: "GET",
							URL:    serverURL + "/step-3",
						},
					},
				},
			},
		},
	}
}

// TestRunFolderSuccess asserts end-to-end execution of a collection folder.
func TestRunFolderSuccess(t *testing.T) {
	var steps []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		steps = append(steps, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("step-ok"))
	}))
	defer server.Close()

	svc, _, b := setupTestService(t)
	runnerSvc := commands.NewRunnerService(svc, nil, b)
	col := setupTestRunnerCollection(server.URL)

	summary, err := runnerSvc.RunFolder(context.Background(), commands.RunFolderCmd{
		Collection: col,
		Recursive:  true,
	})
	if err != nil {
		t.Fatalf("RunFolder failed: %v", err)
	}

	if summary.Total != 3 {
		t.Errorf("expected 3 total requests, got %d", summary.Total)
	}
	if summary.Passed != 3 {
		t.Errorf("expected 3 passed requests, got %d", summary.Passed)
	}
	if summary.Failed != 0 {
		t.Errorf("expected 0 failed requests, got %d", summary.Failed)
	}
	if len(steps) != 3 {
		t.Errorf("expected 3 requests hit server, got %d (%v)", len(steps), steps)
	}
}

// TestRunFolderTagFiltering asserts filtering by tags.
func TestRunFolderTagFiltering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	svc, _, b := setupTestService(t)
	runnerSvc := commands.NewRunnerService(svc, nil, b)
	col := setupTestRunnerCollection(server.URL)

	// Filter by tag "regression": only item-r2 should run
	summary, err := runnerSvc.RunFolder(context.Background(), commands.RunFolderCmd{
		Collection: col,
		Recursive:  true,
		FilterTags: []string{"regression"},
	})
	if err != nil {
		t.Fatalf("RunFolder failed: %v", err)
	}

	if summary.Total != 1 {
		t.Errorf("expected 1 request with tag 'regression', got %d", summary.Total)
	}
}

// TestRunFolderCancellation asserts cancelling a runner stops execution
// and publishes a terminal event with ScopeRunner.
func TestRunFolderCancellation(t *testing.T) {
	serverReceived := make(chan struct{})
	unblockServer := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-serverReceived:
		default:
			close(serverReceived)
		}
		<-unblockServer
		w.WriteHeader(http.StatusOK)
	}))
	defer func() {
		close(unblockServer)
		server.Close()
	}()

	svc, _, b := setupTestService(t)
	runnerSvc := commands.NewRunnerService(svc, nil, b)
	col := setupTestRunnerCollection(server.URL)

	ch, unsub := b.Subscribe()
	defer unsub()

	runnerUID := "test-run-cancel"
	go func() {
		_, _ = runnerSvc.RunFolder(context.Background(), commands.RunFolderCmd{
			RunnerUID:  runnerUID,
			Collection: col,
			Recursive:  true,
		})
	}()

	<-serverReceived

	// Cancel the runner
	if !runnerSvc.CancelRunner(runnerUID) {
		t.Fatalf("expected CancelRunner to return true")
	}

	// Wait for runner error event
	timeout := time.After(3 * time.Second)
	var runnerError *events.ErrorOccurred

loop:
	for {
		select {
		case ev := <-ch:
			if e, ok := ev.(events.ErrorOccurred); ok && e.Scope == events.ScopeRunner {
				runnerError = &e
				break loop
			}
		case <-timeout:
			t.Fatalf("timed out waiting for ScopeRunner ErrorOccurred")
		}
	}

	if runnerError == nil {
		t.Fatalf("expected runnerError, got nil")
	}
}

// TestRunFolderRunnerControl tests SetNextRequest, SkipRequest, and StopExecution.
func TestRunFolderRunnerControl(t *testing.T) {
	var stepCalls []string
	var runnerSvc *commands.RunnerService
	runnerUID := "ctrl-run-1"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stepCalls = append(stepCalls, r.URL.Path)
		if r.URL.Path == "/step-1" {
			// In step 1, jump directly to Request 3 (skipping Request 2)
			if ctrl, ok := runnerSvc.Controller(runnerUID); ok {
				ctrl.SetNextRequest("Request 3")
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	svc, _, b := setupTestService(t)
	runnerSvc = commands.NewRunnerService(svc, nil, b)
	col := setupTestRunnerCollection(server.URL)

	summary, err := runnerSvc.RunFolder(context.Background(), commands.RunFolderCmd{
		RunnerUID:  runnerUID,
		Collection: col,
		Recursive:  true,
	})
	if err != nil {
		t.Fatalf("RunFolder failed: %v", err)
	}

	if summary.Total != 2 {
		t.Errorf("expected 2 total executed requests after jumping, got %d", summary.Total)
	}
	if len(stepCalls) != 2 || stepCalls[0] != "/step-1" || stepCalls[1] != "/step-3" {
		t.Errorf("expected calls [/step-1, /step-3], got %v", stepCalls)
	}
}

// TestRunFolderBailOnFailure asserts that Bail=true stops execution on first failing request.
func TestRunFolderBailOnFailure(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc, _, b := setupTestService(t)
	runnerSvc := commands.NewRunnerService(svc, nil, b)
	col := setupTestRunnerCollection(server.URL)

	summary, err := runnerSvc.RunFolder(context.Background(), commands.RunFolderCmd{
		Collection: col,
		Recursive:  true,
		Bail:       true,
	})
	if err != nil {
		t.Fatalf("RunFolder failed: %v", err)
	}

	if summary.Total != 1 {
		t.Errorf("expected exactly 1 request before bail, got %d", summary.Total)
	}
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
	if count.Load() != 1 {
		t.Errorf("expected server hit only 1 time, got %d", count.Load())
	}
}
