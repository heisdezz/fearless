package persistence_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fearless/internal/core/model"
	"fearless/internal/core/persistence"
)

func TestSpoolInlineWhenSmall(t *testing.T) {
	s, err := persistence.NewSpooler(persistence.SpoolOptions{
		InlineThreshold: persistence.DefaultInlineThreshold, // 1 MB
		CleanupOnClose:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// 500 KB payload <= 1 MB
	payload := bytes.Repeat([]byte("inline-data-1234"), 32*1024) // 512 KB
	expectedHash := sha256.Sum256(payload)
	expectedID := hex.EncodeToString(expectedHash[:])

	ref, err := s.SpoolBytes(payload, "application/json", "gzip")
	if err != nil {
		t.Fatalf("SpoolBytes failed: %v", err)
	}

	if ref.Spilled() {
		t.Fatalf("expected body to be inline, but it spilled to %s", ref.Path)
	}
	if ref.Path != "" {
		t.Fatalf("expected empty Path, got %s", ref.Path)
	}
	if ref.Size != int64(len(payload)) {
		t.Fatalf("expected Size %d, got %d", len(payload), ref.Size)
	}
	if ref.ID != expectedID {
		t.Fatalf("expected ID %s, got %s", expectedID, ref.ID)
	}
	if ref.ContentType != "application/json" {
		t.Fatalf("expected ContentType application/json, got %s", ref.ContentType)
	}
	if ref.Encoding != "gzip" {
		t.Fatalf("expected Encoding gzip, got %s", ref.Encoding)
	}
	if !bytes.Equal(ref.Inline, payload) {
		t.Fatalf("inline data mismatch")
	}

	// Read through ReadBytes
	readBack, err := s.ReadBytes(ref)
	if err != nil {
		t.Fatalf("ReadBytes failed: %v", err)
	}
	if !bytes.Equal(readBack, payload) {
		t.Fatalf("ReadBytes mismatch")
	}
}

func TestSpoolSpillWhenLarge(t *testing.T) {
	s, err := persistence.NewSpooler(persistence.SpoolOptions{
		InlineThreshold: persistence.DefaultInlineThreshold, // 1 MB
		CleanupOnClose:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// 1.5 MB payload > 1 MB
	payload := bytes.Repeat([]byte("large-spool-payload-"), 75*1024) // ~1.5 MB
	expectedHash := sha256.Sum256(payload)
	expectedID := hex.EncodeToString(expectedHash[:])

	ref, err := s.SpoolBytes(payload, "application/octet-stream", "")
	if err != nil {
		t.Fatalf("SpoolBytes failed: %v", err)
	}

	if !ref.Spilled() {
		t.Fatalf("expected payload > 1 MB to spill to disk")
	}
	if ref.Inline != nil {
		t.Fatalf("expected ref.Inline to be nil for spilled body")
	}
	if ref.Path == "" {
		t.Fatalf("expected non-empty Path")
	}
	if ref.ID != expectedID {
		t.Fatalf("expected ID %s, got %s", expectedID, ref.ID)
	}
	if ref.Size != int64(len(payload)) {
		t.Fatalf("expected Size %d, got %d", len(payload), ref.Size)
	}

	// Verify file on disk
	diskContent, err := os.ReadFile(ref.Path)
	if err != nil {
		t.Fatalf("failed to read disk file %s: %v", ref.Path, err)
	}
	if !bytes.Equal(diskContent, payload) {
		t.Fatalf("disk content mismatch")
	}

	// Read via Open
	rc, err := s.Open(ref)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer rc.Close()
	all, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(all, payload) {
		t.Fatalf("streamed content mismatch")
	}
}

func TestDeduplication(t *testing.T) {
	s, err := persistence.NewSpooler(persistence.SpoolOptions{
		InlineThreshold: 1024, // 1 KB
		CleanupOnClose:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	payload := bytes.Repeat([]byte("dedup-payload-content"), 200) // ~4.2 KB
	ref1, err := s.SpoolBytes(payload, "text/plain", "")
	if err != nil {
		t.Fatal(err)
	}
	ref2, err := s.SpoolBytes(payload, "text/plain", "")
	if err != nil {
		t.Fatal(err)
	}

	if ref1.ID != ref2.ID {
		t.Fatalf("expected same ID: %s vs %s", ref1.ID, ref2.ID)
	}
	if ref1.Path != ref2.Path {
		t.Fatalf("expected deduplicated same path: %s vs %s", ref1.Path, ref2.Path)
	}
}

type bodyRefEntry struct {
	ref   model.BodyRef
	index int
}

func TestEvictionObeysMaxCacheBytes(t *testing.T) {
	// Acceptance criterion:
	// "2. Eviction policy obeys max_cache_bytes (tested with synthetic flood)."
	maxCache := int64(30 * 1024) // 30 KB max cache limit
	inlineLimit := int64(1024)   // 1 KB threshold so 5 KB items spill

	s, err := persistence.NewSpooler(persistence.SpoolOptions{
		InlineThreshold: inlineLimit,
		MaxCacheBytes:   maxCache,
		CleanupOnClose:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Synthetic flood: 15 items of 5 KB each = 75 KB total (exceeds 30 KB limit)
	var refs []bodyRefEntry
	itemSize := 5 * 1024
	for i := 0; i < 15; i++ {
		// Unique content to avoid deduplication
		uniqueData := bytes.Repeat([]byte(fmt.Sprintf("flood-%02d-", i)), itemSize/9)
		ref, err := s.SpoolBytes(uniqueData, "application/octet-stream", "")
		if err != nil {
			t.Fatalf("spool item %d: %v", i, err)
		}
		if !ref.Spilled() {
			t.Fatalf("item %d should have spilled", i)
		}
		refs = append(refs, bodyRefEntry{ref: ref, index: i})
		// Tiny sleep to guarantee distinct ModTimes for deterministic eviction order
		time.Sleep(5 * time.Millisecond)
	}

	// Verify total cache on disk does not exceed maxCache
	entries, err := os.ReadDir(s.Dir())
	if err != nil {
		t.Fatal(err)
	}
	var totalOnDisk int64
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		totalOnDisk += info.Size()
	}

	if totalOnDisk > maxCache {
		t.Fatalf("cache size on disk (%d bytes) exceeds maxCacheBytes (%d bytes)!", totalOnDisk, maxCache)
	}

	// The newest items must be alive, oldest items should have been evicted
	lastItem := refs[len(refs)-1]
	if _, err := os.Stat(lastItem.ref.Path); err != nil {
		t.Fatalf("newest spooled item should be retained on disk: %v", err)
	}
}

func TestSizeCapAndTruncation(t *testing.T) {
	s, err := persistence.NewSpooler(persistence.SpoolOptions{
		InlineThreshold: 512,
		CleanupOnClose:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	payload := bytes.Repeat([]byte("0123456789"), 1000) // 10,000 bytes
	sizeCap := int64(2000)                              // Cap at 2,000 bytes

	ref, err := s.Spool(bytes.NewReader(payload), "text/plain", "", sizeCap)
	if err != nil {
		t.Fatalf("Spool failed: %v", err)
	}

	if !ref.Truncated {
		t.Fatalf("expected ref.Truncated to be true")
	}
	if ref.Size != sizeCap {
		t.Fatalf("expected Size %d, got %d", sizeCap, ref.Size)
	}

	data, err := s.ReadBytes(ref)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(data)) != sizeCap {
		t.Fatalf("read data length %d != sizeCap %d", len(data), sizeCap)
	}
}

func TestCleanupOnClose(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "persist-test-cleanup-*")
	if err != nil {
		t.Fatal(err)
	}

	s, err := persistence.NewSpooler(persistence.SpoolOptions{
		Dir:             tempDir,
		InlineThreshold: 10,
		CleanupOnClose:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.SpoolBytes([]byte("larger than 10 bytes content"), "text/plain", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	// After close with CleanupOnClose, directory must be removed
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Fatalf("expected dir %s to be removed on Close()", tempDir)
	}
}

func TestConcurrentSpooling(t *testing.T) {
	s, err := persistence.NewSpooler(persistence.SpoolOptions{
		InlineThreshold: 1024,
		MaxCacheBytes:   10 * 1024 * 1024,
		CleanupOnClose:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var wg sync.WaitGroup
	for worker := 0; worker < 10; worker++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				data := bytes.Repeat([]byte(fmt.Sprintf("worker-%d-item-%d-", w, i)), 200)
				ref, err := s.SpoolBytes(data, "text/plain", "")
				if err != nil {
					t.Errorf("worker %d spool error: %v", w, err)
					return
				}
				read, err := s.ReadBytes(ref)
				if err != nil {
					t.Errorf("worker %d read error: %v", w, err)
					return
				}
				if !bytes.Equal(read, data) {
					t.Errorf("worker %d data mismatch", w)
					return
				}
			}
		}(worker)
	}
	wg.Wait()
}

func TestSpoolerEdgeCases(t *testing.T) {
	dir := t.TempDir()
	spoolDir := filepath.Join(dir, "custom-spool")

	s, err := persistence.NewSpooler(persistence.SpoolOptions{
		Dir:             spoolDir,
		InlineThreshold: 100,
		MaxCacheBytes:   1024 * 1024,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if s.Dir() != spoolDir {
		t.Fatalf("expected Dir %s, got %s", spoolDir, s.Dir())
	}

	// Evict on empty directory
	reclaimed, err := s.Evict()
	if err != nil {
		t.Fatalf("Evict on empty failed: %v", err)
	}
	if reclaimed != 0 {
		t.Fatalf("expected 0 reclaimed on empty dir")
	}

	// Open nonexistent spilled file
	fakeRef := model.BodyRef{
		Path: filepath.Join(spoolDir, "nonexistent-hash"),
	}
	_, err = s.Open(fakeRef)
	if !errors.Is(err, persistence.ErrSpoolFileNotFound) {
		t.Fatalf("expected ErrSpoolFileNotFound, got %v", err)
	}
	_, err = s.ReadBytes(fakeRef)
	if !errors.Is(err, persistence.ErrSpoolFileNotFound) {
		t.Fatalf("expected ErrSpoolFileNotFound on ReadBytes, got %v", err)
	}

	// Close spooler and verify operations after close
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	// Double close is a no-op
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = s.SpoolBytes([]byte("after close"), "text/plain", "")
	if !errors.Is(err, persistence.ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
	_, err = s.Evict()
	if !errors.Is(err, persistence.ErrClosed) {
		t.Fatalf("expected ErrClosed on Evict, got %v", err)
	}
}
