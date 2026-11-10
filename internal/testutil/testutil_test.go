package testutil

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestTempDir(t *testing.T) {
	d := TempDir(t)
	if _, err := os.Stat(d); err != nil {
		t.Fatalf("temp dir missing: %v", err)
	}
}

func TestWriteReadAndGolden(t *testing.T) {
	d := TempDir(t)
	p := filepath.Join(d, "a", "b.txt")
	WriteFile(t, p, []byte("hello"))
	got := ReadFile(t, p)
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
	g := filepath.Join(d, "golden.txt")
	Golden(t, g, []byte("golden"), true)
	Golden(t, g, []byte("golden"), false)
}

func TestHTTPTestServer(t *testing.T) {
	s := NewHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	resp, err := http.Get(s.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
