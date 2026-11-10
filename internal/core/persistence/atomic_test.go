package persistence_test

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"fearless/internal/core/persistence"
)

func TestWriteAtomic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sub", "test.txt")
	content := []byte("hello fearless atomic write")

	if err := persistence.WriteAtomic(target, content, 0o600); err != nil {
		t.Fatalf("WriteAtomic failed: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("expected %q, got %q", content, got)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected perm 0600, got %o", perm)
	}
}

func TestWriteAtomicStream(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "stream.bin")
	content := bytes.Repeat([]byte("stream-atomic-"), 1000)

	if err := persistence.WriteAtomicStream(target, bytes.NewReader(content), 0o644); err != nil {
		t.Fatalf("WriteAtomicStream failed: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch")
	}
}

func TestWriteAtomicNoTornReads(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "concurrent.bin")

	// Pre-create initial version
	initData := bytes.Repeat([]byte("A"), 64*1024)
	if err := persistence.WriteAtomic(target, initData, 0o644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// 5 readers continuously reading target
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					data, err := os.ReadFile(target)
					if err != nil {
						t.Errorf("read error during concurrent write: %v", err)
						return
					}
					// Every read must be complete: either all A's or all B's, never partial length
					if len(data) != 64*1024 {
						t.Errorf("torn read observed! length=%d", len(data))
						return
					}
					first := data[0]
					if first != 'A' && first != 'B' && first != 'C' {
						t.Errorf("unexpected content byte: %c", first)
						return
					}
					for _, b := range data {
						if b != first {
							t.Errorf("torn content observed! Mixed bytes in single read")
							return
						}
					}
				}
			}
		}()
	}

	// Writer overwriting target with new atomic versions
	for round := 0; round < 30; round++ {
		char := byte('B')
		if round%2 == 0 {
			char = 'C'
		}
		newData := bytes.Repeat([]byte{char}, 64*1024)
		if err := persistence.WriteAtomic(target, newData, 0o644); err != nil {
			t.Fatalf("WriteAtomic error: %v", err)
		}
	}

	close(stop)
	wg.Wait()
}

func TestQuarantineCorrupt(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "preferences.json")
	corruptContent := []byte(`{"broken": json`)

	if err := os.WriteFile(target, corruptContent, 0o644); err != nil {
		t.Fatal(err)
	}

	quarantined, err := persistence.QuarantineCorrupt(target)
	if err != nil {
		t.Fatalf("QuarantineCorrupt failed: %v", err)
	}

	// Original file must not exist
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("original file should have been moved away")
	}

	// Quarantined file must exist with identical content
	got, err := os.ReadFile(quarantined)
	if err != nil {
		t.Fatalf("failed to read quarantined file: %v", err)
	}
	if !bytes.Equal(got, corruptContent) {
		t.Fatalf("quarantined content mismatch")
	}
}
