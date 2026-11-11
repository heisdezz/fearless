package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// setupMockCollection creates a temporary collection directory with bruno.json,
// environments, and request items.
func setupMockCollection(t *testing.T, serverURL string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cli-test-col-*")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	// bruno.json
	brunoJSON := `{
  "version": "1",
  "name": "CLI Mock Collection",
  "type": "collection",
  "ignore": [".git", "node_modules"]
}`
	if err := os.WriteFile(filepath.Join(dir, "bruno.json"), []byte(brunoJSON), 0o644); err != nil {
		t.Fatalf("write bruno.json failed: %v", err)
	}

	// environments/Local.bru
	envDir := filepath.Join(dir, "environments")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("mkdir environments failed: %v", err)
	}
	localEnv := fmt.Sprintf(`vars {
  host: %s
  endpoint: /test-endpoint
  param_var: from-env-file
}
`, serverURL)
	if err := os.WriteFile(filepath.Join(envDir, "Local.bru"), []byte(localEnv), 0o644); err != nil {
		t.Fatalf("write Local.bru failed: %v", err)
	}

	// request-1.bru
	req1 := `meta {
  name: Request 1
  type: http
  seq: 1
}

get {
  url: {{host}}{{endpoint}}
}
`
	if err := os.WriteFile(filepath.Join(dir, "request-1.bru"), []byte(req1), 0o644); err != nil {
		t.Fatalf("write request-1.bru failed: %v", err)
	}

	// subfolder/request-2.bru
	subDir := filepath.Join(dir, "subfolder")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir subfolder failed: %v", err)
	}
	req2 := `meta {
  name: Request 2
  type: http
  seq: 2
}

get {
  url: {{host}}/subfolder-endpoint?q={{param_var}}
}
`
	if err := os.WriteFile(filepath.Join(subDir, "request-2.bru"), []byte(req2), 0o644); err != nil {
		t.Fatalf("write request-2.bru failed: %v", err)
	}

	return dir
}

// TestCLIRunSuccess asserts that running a collection against a successful server
// succeeds with exit code 0.
func TestCLIRunSuccess(t *testing.T) {
	var reqCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	colDir := setupMockCollection(t, server.URL)

	var stdout, stderr bytes.Buffer
	code := run([]string{"run", colDir, "-e", "Local"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr: %s, stdout: %s", code, stderr.String(), stdout.String())
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, "Ran 2 requests (2 passed, 0 failed)") {
		t.Errorf("expected 2 passed requests, got output:\n%s", outStr)
	}
	if reqCount.Load() != 2 {
		t.Errorf("expected 2 server requests, got %d", reqCount.Load())
	}
}

// TestCLIRunFailureNonZeroExit asserts Acceptance Criterion 3:
// Non-zero exit on failure.
func TestCLIRunFailureNonZeroExit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	colDir := setupMockCollection(t, server.URL)

	var stdout, stderr bytes.Buffer
	code := run([]string{"run", colDir, "-e", "Local"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected non-zero exit code 1, got %d", code)
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, "failed") {
		t.Errorf("expected failure in output, got:\n%s", outStr)
	}
}

// TestCLIGlobalEnvVarOverride asserts Acceptance Criterion 2:
// Environment selection flags match cli-global-env-var-override behaviour.
// `--env-var key=value` overrides environment file variable.
func TestCLIGlobalEnvVarOverride(t *testing.T) {
	var seenQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/subfolder-endpoint" {
			seenQuery = r.URL.Query().Get("q")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	colDir := setupMockCollection(t, server.URL)

	var stdout, stderr bytes.Buffer
	// Override param_var from "from-env-file" to "cli-override-value"
	code := run([]string{
		"run", colDir,
		"-e", "Local",
		"--env-var", "param_var=cli-override-value",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run failed with code %d, stderr: %s", code, stderr.String())
	}

	if seenQuery != "cli-override-value" {
		t.Errorf("expected query param overridden to 'cli-override-value', got %q", seenQuery)
	}
}

// TestCLIRunSingleFile asserts running a single .bru file.
func TestCLIRunSingleFile(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	colDir := setupMockCollection(t, server.URL)
	singleFile := filepath.Join(colDir, "request-1.bru")

	var stdout, stderr bytes.Buffer
	code := run([]string{
		"run", singleFile,
		"--env-var", fmt.Sprintf("host=%s", server.URL),
		"--env-var", "endpoint=/single-hit",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run single file failed: %d, stderr: %s", code, stderr.String())
	}
	if requestedPath != "/single-hit" {
		t.Errorf("expected /single-hit, got %q", requestedPath)
	}
}

// TestCLIVersionAndHelp asserts version and help outputs.
func TestCLIVersionAndHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"version"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "fearless-cli version") {
		t.Errorf("version output failed: %s", stdout.String())
	}

	stdout.Reset()
	code = run([]string{"help"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("help output failed: %s", stdout.String())
	}
}

// TestCLIExportAndImport asserts collection export and import commands.
func TestCLIExportAndImport(t *testing.T) {
	colDir := setupMockCollection(t, "http://localhost:8080")

	// Export to stdout
	var stdout, stderr bytes.Buffer
	code := run([]string{"export", colDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("export stdout failed: %d, stderr: %s", code, stderr.String())
	}

	var parsed map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("export did not produce valid JSON: %v", err)
	}
	if parsed["name"] != "CLI Mock Collection" {
		t.Errorf("expected collection name 'CLI Mock Collection', got %v", parsed["name"])
	}

	// Export to file
	tmpExportFile := filepath.Join(colDir, "exported.json")
	code = run([]string{"export", colDir, "-o", tmpExportFile}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("export to file failed: %d", code)
	}
	if _, err := os.Stat(tmpExportFile); err != nil {
		t.Fatalf("exported file not created: %v", err)
	}

	// Import command
	stdout.Reset()
	code = run([]string{"import", colDir}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "Successfully verified and imported") {
		t.Fatalf("import failed: %d, output: %s", code, stdout.String())
	}
}

// TestNoUIDependencies asserts Acceptance Criterion 1:
// Runs a collection end-to-end with no GUI packages linked (verified with go list -deps).
func TestNoUIDependencies(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "fearless/cmd/fearless-cli")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v, output: %s", err, string(out))
	}

	forbidden := []string{"ui", "mygo", "fyne", "gio", "image"}
	for _, pkg := range forbidden {
		if strings.Contains(string(out), "/"+pkg) || strings.Contains(string(out), "\n"+pkg+"\n") {
			t.Errorf("forbidden GUI/image package linked in fearless-cli: %s", pkg)
		}
	}
}

// TestCLIEdgeCasesAndCoverage tests CLI edge cases, JSON output formatting, and bail option.
func TestCLIEdgeCasesAndCoverage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	colDir := setupMockCollection(t, server.URL)

	// No arguments prints usage
	var stdout, stderr bytes.Buffer
	code := run([]string{}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("empty args expected code 0, got %d", code)
	}

	// Unknown flag or command
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--invalid-flag"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("invalid command expected code 1, got %d", code)
	}

	// JSON format output
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"run", colDir, "-e", "Local", "--format", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run json format failed: %d, stderr: %s", code, stderr.String())
	}
	var summary RunSummary
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatalf("failed to unmarshal JSON run output: %v", err)
	}
	if summary.Total != 2 || summary.Passed != 2 {
		t.Errorf("expected summary 2 passed, got %+v", summary)
	}

	// Bail option on failure
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	failColDir := setupMockCollection(t, failServer.URL)
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"run", failColDir, "-e", "Local", "--bail"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected code 1 on failure, got %d", code)
	}

	// Import missing arg
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"import"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected code 1 for import without args, got %d", code)
	}

	// Export missing/invalid path
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"export", "/nonexistent/collection/path"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected code 1 for invalid export path, got %d", code)
	}

	// Load target non-existent path
	_, _, err := loadTarget("/nonexistent/file/path", true)
	if err == nil {
		t.Errorf("expected error for non-existent target path, got nil")
	}

	// Test envVarFlags Set without value
	flags := make(envVarFlags)
	_ = flags.Set("KEY_ONLY")
	if flags["KEY_ONLY"] != "" {
		t.Errorf("expected empty value for KEY_ONLY, got %v", flags["KEY_ONLY"])
	}
	_ = flags.String()
}
