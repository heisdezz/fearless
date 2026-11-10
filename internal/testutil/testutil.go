package testutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temporary directory for tests.
func TempDir(t testing.TB) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "fearless-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// WriteFile writes content to path, creating parent directories as needed.
func WriteFile(t testing.TB, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// ReadFile reads a file or fails the test.
func ReadFile(t testing.TB, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", path, err)
	}
	return b
}

// Golden compares got against the contents of goldenPath. When update is true,
// it writes got to goldenPath instead of comparing.
func Golden(t testing.TB, goldenPath string, got []byte, update bool) {
	t.Helper()
	if update {
		WriteFile(t, goldenPath, got)
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v", goldenPath, err)
	}
	if string(want) != string(got) {
		t.Fatalf("golden mismatch for %s\nwant (%d bytes):\n%s\n\ngot (%d bytes):\n%s", goldenPath, len(want), string(want), len(got), string(got))
	}
}

// NewHTTPTestServer starts an httptest.Server and registers cleanup.
func NewHTTPTestServer(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// NewHTTPSTestServer starts an httptest TLS server and registers cleanup.
func NewHTTPSTestServer(t testing.TB, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	return srv
}
