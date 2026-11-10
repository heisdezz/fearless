package persistence

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// WriteAtomic writes data to path atomically using a temporary file in the same
// directory followed by an fsync and os.Rename. This ensures that readers never
// see partial/torn writes even under crashes, power loss, or cancellations.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent dir %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
	}

	cleanup = false
	return nil
}

// WriteAtomicStream streams content from r to path atomically.
func WriteAtomicStream(path string, r io.Reader, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent dir %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmpFile, r); err != nil {
		return fmt.Errorf("copy stream to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
	}

	cleanup = false
	return nil
}

// QuarantineCorrupt moves a corrupted file at path to path.corrupt-<timestamp>
// instead of silently deleting it. This preserves the corrupt file for debugging
// and post-mortem analysis.
func QuarantineCorrupt(path string) (string, error) {
	ts := time.Now().UTC().Format("20060102T150405Z")
	target := fmt.Sprintf("%s.corrupt-%s", path, ts)
	if err := os.Rename(path, target); err != nil {
		return "", fmt.Errorf("quarantine %s to %s: %w", path, target, err)
	}
	return target, nil
}
