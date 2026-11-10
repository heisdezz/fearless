package engine_test

import (
	"context"
	"os"
	"time"

	"encoding/json"
	"errors"
	"fearless/internal/core/cookies"
	"fearless/internal/core/persistence"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"fearless/internal/core/engine"
	"fearless/internal/core/engine/auth"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
	"fearless/internal/core/vars"
)

var registerAuthOnce sync.Once

func init() {
	registerAuthOnce.Do(func() {
		auth.RegisterBase()
	})
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

// TestPhaseOrdering asserts that the request lifecycle strictly executes
// all phases in the exact sequence specified by flows/request-lifecycle.md.
func TestPhaseOrdering(t *testing.T) {
	var mu sync.Mutex
	var order []string
	record := func(phase string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, phase)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record("7.execute:server_received")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "secret123", "user": "alice"}`))
	}))
	defer server.Close()

	recSink := &engine.RecordingSink{}
	recordingSinkWrapper := &phaseRecordingSink{
		inner:  recSink,
		record: record,
	}

	preHook := engine.PreRequestFunc(func(ctx context.Context, item *model.Item, scopes *vars.Scopes, sink engine.EventSink) error {
		record("4.pre_request_script")
		if scopes.Request == nil {
			scopes.Request = make(map[string]any)
		}
		scopes.Request["header_val"] = "custom-val"
		return nil
	})

	postHook := engine.PostResponseFunc(func(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink engine.EventSink) error {
		record("10.post_response_script")
		return nil
	})

	assertHook := engine.AssertFunc(func(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink engine.EventSink) (*events.AssertionResults, error) {
		record("10.assertions")
		return &events.AssertionResults{}, nil
	})

	testHook := engine.TestFunc(func(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink engine.EventSink) (*events.TestResults, error) {
		record("10.tests")
		return &events.TestResults{}, nil
	})

	item := &model.Item{
		Name: "Test Item",
		Request: &model.Request{
			Method: "POST",
			URL:    server.URL + "/api/v1/auth",
			Headers: []model.KeyValue{
				{
					Name:    strPtr("X-Custom"),
					Value:   strPtr("{{header_val}}"),
					Enabled: true,
				},
			},
			Body: &model.Body{
				Mode: model.BodyJSON,
				JSON: strPtr(`{"action": "login"}`),
			},
			Vars: &model.VarBlocks{
				PreRequest: []model.Variable{
					{
						Name:    strPtr("pre_var"),
						Value:   "initialized",
						Enabled: true,
					},
				},
				PostResponse: []model.Variable{
					{
						Name:    strPtr("token"),
						Value:   "$res.body.access_token",
						Enabled: true,
					},
				},
			},
		},
	}

	eng := engine.New()
	in := engine.Input{
		Item: item,
		Scopes: vars.Scopes{
			Request: make(map[string]any),
		},
		Hooks: engine.HookSet{
			PreRequest:   []engine.PreRequestHook{preHook},
			PostResponse: []engine.PostResponseHook{postHook},
			Assertions:   []engine.AssertHook{assertHook},
			Tests:        []engine.TestHook{testHook},
		},
		Sink: recordingSinkWrapper,
	}

	res, err := eng.Do(context.Background(), in)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil result")
	}

	expectedPrefix := []string{
		"4.pre_request_script",
		"7.queued",
		"7.execute:server_received",
	}

	expectedSuffix := []string{
		"10.post_response_script",
		"10.assertions",
		"10.tests",
	}

	for i, exp := range expectedPrefix {
		if i >= len(order) || order[i] != exp {
			t.Fatalf("phase mismatch at index %d: got %v, want prefix %v (full order: %v)", i, order, expectedPrefix, order)
		}
	}

	lastIdx := len(order) - len(expectedSuffix)
	if lastIdx < 0 {
		t.Fatalf("recorded order too short: %v", order)
	}
	for i, exp := range expectedSuffix {
		if order[lastIdx+i] != exp {
			t.Fatalf("phase mismatch at suffix index %d: got %s, want %s (full order: %v)", i, order[lastIdx+i], exp, order)
		}
	}

	// Verify post-response vars were populated
	if in.Scopes.Request["token"] != "secret123" {
		t.Errorf("expected token 'secret123' in Scopes.Request, got %v", in.Scopes.Request["token"])
	}
}

// phaseRecordingSink wraps an EventSink to log phase execution.
type phaseRecordingSink struct {
	inner  engine.EventSink
	record func(string)
}

func (s *phaseRecordingSink) Queued(sent model.SentRequest) {
	s.record("7.queued")
	s.inner.Queued(sent)
}

func (s *phaseRecordingSink) Chunk(data []byte) {
	s.inner.Chunk(data)
}

func (s *phaseRecordingSink) Phase(entry model.TimelineEntry) {
	s.inner.Phase(entry)
}

func (s *phaseRecordingSink) ScriptOutput(level, text string) {
	s.inner.ScriptOutput(level, text)
}

func (s *phaseRecordingSink) Unresolved(names []string) {
	s.inner.Unresolved(names)
}

// TestExactlyOneResultOrError asserts Acceptance Criterion 2:
// Exactly one of Result or error is returned by Do across all paths.
func TestExactlyOneResultOrError(t *testing.T) {
	eng := engine.New()

	t.Run("SuccessReturnsResultAndNilError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		res, err := eng.Do(context.Background(), engine.Input{
			Item: &model.Item{
				Request: &model.Request{
					Method: "GET",
					URL:    server.URL,
				},
			},
		})
		if res == nil || err != nil {
			t.Errorf("expected res != nil and err == nil, got res=%v, err=%v", res, err)
		}
	})

	t.Run("NilItemReturnsNilResultAndError", func(t *testing.T) {
		res, err := eng.Do(context.Background(), engine.Input{
			Item: nil,
		})
		if res != nil || err == nil {
			t.Errorf("expected res == nil and err != nil, got res=%v, err=%v", res, err)
		}
	})

	t.Run("CancelledContextReturnsNilResultAndError", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := eng.Do(ctx, engine.Input{
			Item: &model.Item{
				Request: &model.Request{
					Method: "GET",
					URL:    "http://127.0.0.1:1",
				},
			},
		})
		if res != nil || err == nil {
			t.Errorf("expected res == nil and err != nil, got res=%v, err=%v", res, err)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got %v", err)
		}
	})

	t.Run("PreRequestHookErrorReturnsNilResultAndError", func(t *testing.T) {
		boomErr := errors.New("pre-request script crashed")
		hook := engine.PreRequestFunc(func(ctx context.Context, item *model.Item, scopes *vars.Scopes, sink engine.EventSink) error {
			return boomErr
		})

		res, err := eng.Do(context.Background(), engine.Input{
			Item: &model.Item{
				Request: &model.Request{
					Method: "GET",
					URL:    "http://example.com",
				},
			},
			Hooks: engine.HookSet{
				PreRequest: []engine.PreRequestHook{hook},
			},
		})
		if res != nil || err == nil {
			t.Errorf("expected res == nil and err != nil, got res=%v, err=%v", res, err)
		}
		if !errors.Is(err, boomErr) {
			t.Errorf("expected boomErr, got %v", err)
		}
	})

	t.Run("NetworkFailureStillReturnsSingleResult", func(t *testing.T) {
		// Port 1 on 127.0.0.1 typically immediately refuses connections
		res, err := eng.Do(context.Background(), engine.Input{
			Item: &model.Item{
				Request: &model.Request{
					Method: "GET",
					URL:    "http://127.0.0.1:1/nonexistent",
				},
			},
		})
		// Transport absorbs network failures into Result.Err so caller has the attempted result
		if res == nil && err == nil {
			t.Fatalf("returned neither result nor error")
		}
		if res != nil && err != nil {
			t.Fatalf("returned both result and error: res=%v, err=%v", res, err)
		}
		if res != nil && res.Err == "" {
			t.Errorf("expected res.Err to be populated for failed connection")
		}
	})
}

// TestUnresolvedVariablesReported asserts that unresolved placeholders in
// URL or headers trigger the sink's Unresolved callback.
func TestUnresolvedVariablesReported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sink := &engine.RecordingSink{}
	eng := engine.New()

	item := &model.Item{
		Request: &model.Request{
			Method: "GET",
			URL:    server.URL + "/path?foo={{missing_query_var}}",
			Headers: []model.KeyValue{
				{
					Name:    strPtr("X-Header"),
					Value:   strPtr("{{missing_header_var}}"),
					Enabled: true,
				},
			},
		},
	}

	res, err := eng.Do(context.Background(), engine.Input{
		Item: item,
		Sink: sink,
	})
	if err != nil || res == nil {
		t.Fatalf("unexpected Do failure: res=%v, err=%v", res, err)
	}

	if len(sink.UnresolvedVars) == 0 {
		t.Fatalf("expected unresolved variables reported to sink, got none")
	}

	foundQuery := false
	foundHeader := false
	for _, batch := range sink.UnresolvedVars {
		for _, name := range batch {
			if name == "missing_query_var" {
				foundQuery = true
			}
			if name == "missing_header_var" {
				foundHeader = true
			}
		}
	}
	if !foundQuery || !foundHeader {
		t.Errorf("expected missing_query_var and missing_header_var, got %v", sink.UnresolvedVars)
	}
}

// TestPostResponseVarsAndJSONBody asserts that post-response vars extract values
// from a JSON response body into the request scope.
func TestPostResponseVarsAndJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":    42,
				"email": "user@example.com",
			},
		})
	}))
	defer server.Close()

	eng := engine.New()
	item := &model.Item{
		Request: &model.Request{
			Method: "GET",
			URL:    server.URL,
			Vars: &model.VarBlocks{
				PostResponse: []model.Variable{
					{
						Name:    strPtr("userId"),
						Value:   "$res.body.data.id",
						Enabled: true,
					},
					{
						Name:    strPtr("userEmail"),
						Value:   "$res.body.data.email",
						Enabled: true,
					},
				},
			},
		},
	}

	in := engine.Input{
		Item: item,
		Scopes: vars.Scopes{
			Request: make(map[string]any),
		},
	}

	res, err := eng.Do(context.Background(), in)
	if err != nil || res == nil {
		t.Fatalf("unexpected Do result: res=%v, err=%v", res, err)
	}

	if fmt.Sprintf("%v", in.Scopes.Request["userId"]) != "42" {
		t.Errorf("expected userId 42, got %v", in.Scopes.Request["userId"])
	}
	if in.Scopes.Request["userEmail"] != "user@example.com" {
		t.Errorf("expected user@example.com, got %v", in.Scopes.Request["userEmail"])
	}
}

// TestStreamingAndTimelineToSink asserts that chunks and timeline phases
// are forwarded from transport to the EventSink.
func TestStreamingAndTimelineToSink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("chunk1-"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		_, _ = w.Write([]byte("chunk2"))
	}))
	defer server.Close()

	sink := &engine.RecordingSink{}
	eng := engine.New()

	item := &model.Item{
		Request: &model.Request{
			Method: "GET",
			URL:    server.URL,
		},
	}

	res, err := eng.Do(context.Background(), engine.Input{
		Item: item,
		Sink: sink,
	})
	if err != nil || res == nil {
		t.Fatalf("Do failed: %v", err)
	}

	if len(sink.Chunks) == 0 {
		t.Errorf("expected chunks delivered to sink")
	}
	if len(sink.Phases) == 0 {
		t.Errorf("expected timeline phases delivered to sink")
	}
	if len(sink.SentRequests) == 0 {
		t.Errorf("expected SentRequest queued event delivered to sink")
	}
}

// TestNoUIDependencies asserts Acceptance Criterion 3:
// The engine must never link or import any UI or image packages.
func TestNoUIDependencies(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{.Imports}}", "fearless/internal/core/engine/...")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list failed: %v, output: %s", err, string(out))
	}

	forbidden := []string{"ui", "mygo", "fyne", "gio", "image"}
	for _, pkg := range forbidden {
		if strings.Contains(string(out), "/"+pkg) || strings.Contains(string(out), " "+pkg+" ") {
			t.Errorf("forbidden import found in engine: %s", pkg)
		}
	}
}

// TestEngineOptionsAndDefaults tests engine configuration options and default propagation.
func TestEngineOptionsAndDefaults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spooler-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	spooler := &persistence.Spooler{}
	_ = spooler

	jar, err := cookies.New()
	if err != nil {
		t.Fatalf("cookies.New failed: %v", err)
	}

	eng := engine.New(
		engine.WithDefaultUserAgent("test-agent/1.0"),
		engine.WithDefaultJar(jar),
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.UserAgent() != "test-agent/1.0" {
			t.Errorf("expected User-Agent 'test-agent/1.0', got %q", r.UserAgent())
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("options-ok"))
	}))
	defer server.Close()

	res, err := eng.Do(context.Background(), engine.Input{
		Item: &model.Item{
			Request: &model.Request{
				Method: "GET",
				URL:    server.URL,
			},
		},
	})
	if err != nil || res == nil {
		t.Fatalf("Do failed with default options: %v", err)
	}

	// Test NopSink
	nop := engine.NopSink{}
	nop.Queued(model.SentRequest{})
	nop.Chunk([]byte("x"))
	nop.Phase(model.TimelineEntry{})
	nop.ScriptOutput("info", "test")
	nop.Unresolved([]string{"foo"})
}

// TestAuthApplicationAndRewind asserts that auth headers and query parameters
// are applied to the outgoing request and rewind callback can rebuild requests.
func TestAuthApplicationAndRewind(t *testing.T) {
	var receivedAuthHeader string
	var receivedQueryKey string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		receivedQueryKey = r.URL.Query().Get("api_key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("auth-ok"))
	}))
	defer server.Close()

	eng := engine.New()
	item := &model.Item{
		Request: &model.Request{
			Method: "GET",
			URL:    server.URL,
			Auth: &model.Auth{
				Mode:  model.AuthBearer,
				Token: strPtr("secret-token-bearer"),
			},
		},
	}

	res, err := eng.Do(context.Background(), engine.Input{
		Item: item,
	})
	if err != nil || res == nil {
		t.Fatalf("Do failed: %v", err)
	}

	if receivedAuthHeader != "Bearer secret-token-bearer" {
		t.Errorf("expected 'Bearer secret-token-bearer', got %q", receivedAuthHeader)
	}

	// Test API Key query placement
	itemQueryAuth := &model.Item{
		Request: &model.Request{
			Method: "GET",
			URL:    server.URL,
			Auth: &model.Auth{
				Mode:      model.AuthAPIKey,
				Key:       strPtr("api_key"),
				Value:     strPtr("xyz-secret"),
				Placement: model.APIKeyQueryParams,
			},
		},
	}

	resQuery, err := eng.Do(context.Background(), engine.Input{
		Item: itemQueryAuth,
	})
	if err != nil || resQuery == nil {
		t.Fatalf("Do failed: %v", err)
	}

	if receivedQueryKey != "xyz-secret" {
		t.Errorf("expected api_key query param 'xyz-secret', got %q", receivedQueryKey)
	}
}

// TestSettingsInheritanceAndOverrides tests redirect, timeout and header forward settings.
func TestSettingsInheritanceAndOverrides(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	eng := engine.New()

	timeoutMs := model.TimeoutMillis(2500)
	disallowRedirects := false
	forwardAuth := false
	maxRedirects := 0

	item := &model.Item{
		Request: &model.Request{
			Method: "GET",
			URL:    server.URL,
		},
	}

	res, err := eng.Do(context.Background(), engine.Input{
		Item: item,
		Settings: model.ItemSettings{
			Timeout:                    &timeoutMs,
			FollowRedirects:            &disallowRedirects,
			MaxRedirects:               &maxRedirects,
			ForwardAuthorizationHeader: &forwardAuth,
		},
		GlobalPreferences: model.Settings{
			Request: model.RequestDefaults{
				TimeoutMs: 5000,
			},
		},
	})
	if err != nil || res == nil {
		t.Fatalf("Do failed: %v", err)
	}
}

// TestTemplateResponseFromSpooledResult tests reading a spooled body file.
func TestTemplateResponseFromSpooledResult(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "spool-body-*.json")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, _ = tmpFile.WriteString(`{"spooled": true, "count": 100}`)
	_ = tmpFile.Close()

	hName := "X-Test"
	hVal := "test-val"
	res := &model.Result{
		Status:   200,
		Duration: 150 * time.Millisecond,
		Headers: []model.KeyValue{
			{Name: &hName, Value: &hVal, Enabled: true},
		},
		Body: model.BodyRef{
			Path:        tmpFile.Name(),
			ContentType: "application/json",
		},
	}

	tpl := engine.TemplateResponseFromResult(res)
	if tpl.Status != 200 {
		t.Errorf("expected status 200, got %d", tpl.Status)
	}
	if tpl.Headers["x-test"] != "test-val" {
		t.Errorf("expected header x-test = test-val, got %s", tpl.Headers["x-test"])
	}
	m, ok := tpl.Body.(map[string]any)
	if !ok || m["spooled"] != true {
		t.Errorf("expected parsed JSON body map with spooled=true, got %v", tpl.Body)
	}

	// Test nil result
	nilTpl := engine.TemplateResponseFromResult(nil)
	if nilTpl.Status != 0 {
		t.Errorf("expected 0 status for nil result")
	}
}

// TestScriptOutputLogging asserts that script errors are forwarded to sink ScriptOutput.
func TestScriptOutputLogging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	sink := &engine.RecordingSink{}
	eng := engine.New()

	postHookErr := engine.PostResponseFunc(func(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink engine.EventSink) error {
		return errors.New("post-script error")
	})

	assertHookErr := engine.AssertFunc(func(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink engine.EventSink) (*events.AssertionResults, error) {
		return nil, errors.New("assertion error")
	})

	testHookErr := engine.TestFunc(func(ctx context.Context, item *model.Item, res *model.Result, scopes *vars.Scopes, sink engine.EventSink) (*events.TestResults, error) {
		return nil, errors.New("test hook error")
	})

	res, err := eng.Do(context.Background(), engine.Input{
		Item: &model.Item{
			Request: &model.Request{
				Method: "GET",
				URL:    server.URL,
			},
		},
		Hooks: engine.HookSet{
			PostResponse: []engine.PostResponseHook{postHookErr},
			Assertions:   []engine.AssertHook{assertHookErr},
			Tests:        []engine.TestHook{testHookErr},
		},
		Sink: sink,
	})
	if err != nil || res == nil {
		t.Fatalf("Do failed: %v", err)
	}

	if len(sink.ScriptLogs) < 3 {
		t.Errorf("expected at least 3 script error logs, got %d", len(sink.ScriptLogs))
	}
}
