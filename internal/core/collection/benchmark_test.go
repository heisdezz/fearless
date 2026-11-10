package collection_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fearless/internal/core/collection"
	"fearless/internal/core/model"
)

// create1000RequestCollection generates a directory with 1 000 request files across 10 folders.
func create1000RequestCollection(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()

	brunoJSON := `{
  "version": "1",
  "name": "large-benchmark-collection",
  "type": "collection"
}`
	if err := os.WriteFile(filepath.Join(dir, "bruno.json"), []byte(brunoJSON), 0o644); err != nil {
		t.Fatalf("failed to write bruno.json: %v", err)
	}

	// 10 folders, 100 requests each = 1,000 requests
	for f := 1; f <= 10; f++ {
		folderPath := filepath.Join(dir, fmt.Sprintf("folder_%02d", f))
		if err := os.MkdirAll(folderPath, 0o755); err != nil {
			t.Fatalf("failed to mkdir: %v", err)
		}
		folderMeta := fmt.Sprintf("meta {\n  name: Folder %02d\n  seq: %d\n}\n", f, f)
		if err := os.WriteFile(filepath.Join(folderPath, "folder.bru"), []byte(folderMeta), 0o644); err != nil {
			t.Fatalf("failed to write folder.bru: %v", err)
		}

		for r := 1; r <= 100; r++ {
			reqBru := fmt.Sprintf(`meta {
  name: Request %02d-%03d
  type: http
  seq: %d
}

get {
  url: {{host}}/api/v1/resource/%d
  body: none
  auth: inherit
}
`, f, r, r, r)
			reqPath := filepath.Join(folderPath, fmt.Sprintf("req_%03d.bru", r))
			if err := os.WriteFile(reqPath, []byte(reqBru), 0o644); err != nil {
				t.Fatalf("failed to write request file: %v", err)
			}
		}
	}

	return dir
}

// Test1000RequestCollectionLoadTime asserts that a 1,000-request collection loads in under 2 seconds.
func Test1000RequestCollectionLoadTime(t *testing.T) {
	dir := create1000RequestCollection(t)

	start := time.Now()
	col, err := collection.Load(dir)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Load 1000-request collection failed: %v", err)
	}

	totalRequests := 0
	_ = col.Walk(func(it *model.Item) error {
		if !it.IsFolder() {
			totalRequests++
		}
		return nil
	})

	if totalRequests != 1000 {
		t.Fatalf("expected 1000 requests, got %d", totalRequests)
	}

	t.Logf("Loaded 1,000-request collection in %v (limit: 2s)", elapsed)

	if elapsed >= 2*time.Second {
		t.Fatalf("collection load exceeded 2-second SLA: took %v", elapsed)
	}
}

// BenchmarkLoad1000Requests benchmarks loading a 1,000-request collection.
func BenchmarkLoad1000Requests(b *testing.B) {
	dir := create1000RequestCollection(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		col, err := collection.Load(dir)
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
		_ = col
	}
}
