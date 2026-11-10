package persistence

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"fearless/internal/core/model"
)

const (
	// DefaultInlineThreshold is 1 MB. Payloads <= 1 MB remain inline in memory;
	// larger payloads are spooled to disk.
	DefaultInlineThreshold int64 = 1024 * 1024

	// DefaultMaxCacheBytes is 500 MB maximum total spool cache size on disk.
	DefaultMaxCacheBytes int64 = 500 * 1024 * 1024
)

var (
	// ErrClosed is returned when operating on a closed Spooler.
	ErrClosed = errors.New("spooler is closed")

	// ErrSpoolFileNotFound is returned when a referenced spooled file cannot be found.
	ErrSpoolFileNotFound = errors.New("spooled body file not found")
)

// SpoolOptions configures a Spooler.
type SpoolOptions struct {
	// Dir is the directory where spooled body files are stored.
	// If empty, a per-session temporary directory is created automatically.
	Dir string

	// InlineThreshold defines the maximum payload size (in bytes) kept in memory.
	// Payloads exceeding this threshold are written to disk.
	// Defaults to DefaultInlineThreshold (1 MB).
	InlineThreshold int64

	// MaxCacheBytes defines the maximum total disk space used by the spool cache.
	// When exceeded, the oldest spooled files are evicted.
	// Defaults to DefaultMaxCacheBytes (500 MB).
	MaxCacheBytes int64

	// CleanupOnClose removes the spool directory when Close is called.
	CleanupOnClose bool
}

// Spooler manages response body streaming, in-memory buffering, and disk spooling
// backing model.BodyRef.
type Spooler struct {
	mu             sync.Mutex
	dir            string
	inlineThresh   int64
	maxCacheBytes  int64
	cleanupOnClose bool
	closed         bool
}

// NewSpooler creates a new Spooler with the provided options.
func NewSpooler(opts SpoolOptions) (*Spooler, error) {
	dir := opts.Dir
	cleanup := opts.CleanupOnClose
	if dir == "" {
		tempDir, err := os.MkdirTemp("", "fearless-spool-*")
		if err != nil {
			return nil, fmt.Errorf("create spool temp dir: %w", err)
		}
		dir = tempDir
		cleanup = true
	} else {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("create spool dir %s: %w", dir, err)
		}
	}

	inlineThresh := opts.InlineThreshold
	if inlineThresh == 0 {
		inlineThresh = DefaultInlineThreshold
	}

	maxCache := opts.MaxCacheBytes
	if maxCache <= 0 {
		maxCache = DefaultMaxCacheBytes
	}

	return &Spooler{
		dir:            dir,
		inlineThresh:   inlineThresh,
		maxCacheBytes:  maxCache,
		cleanupOnClose: cleanup,
	}, nil
}

// Dir returns the directory where spool files are placed.
func (s *Spooler) Dir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dir
}

// SpoolBytes inlines data if len(data) <= InlineThreshold, or spools it to disk.
func (s *Spooler) SpoolBytes(data []byte, contentType, encoding string) (model.BodyRef, error) {
	return s.Spool(bytes.NewReader(data), contentType, encoding, 0)
}

// Spool streams data from r, computing sha256. If total bytes <= inlineThreshold,
// the body is inlined. If it exceeds inlineThreshold, it is spooled atomically
// to disk. If sizeCap > 0, reading is cut off at sizeCap and Truncated is set.
func (s *Spooler) Spool(r io.Reader, contentType, encoding string, sizeCap int64) (model.BodyRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return model.BodyRef{}, ErrClosed
	}

	hasher := sha256.New()
	buffered := make([]byte, 0, min(s.inlineThresh, 64*1024))
	bufReader := io.TeeReader(r, hasher)

	// Read up to inlineThresh + 1 into memory buffer
	chunk := make([]byte, 32*1024)
	spilled := false
	var totalRead int64
	var tmpFile *os.File
	var tmpPath string

	defer func() {
		if tmpFile != nil {
			_ = tmpFile.Close()
			if tmpPath != "" {
				_ = os.Remove(tmpPath)
			}
		}
	}()

	truncated := false

	for {
		// Respect sizeCap if set
		toRead := int64(len(chunk))
		if sizeCap > 0 {
			remaining := sizeCap - totalRead
			if remaining <= 0 {
				truncated = true
				break
			}
			if remaining < toRead {
				toRead = remaining
			}
		}

		n, err := bufReader.Read(chunk[:toRead])
		if n > 0 {
			totalRead += int64(n)

			if !spilled {
				if totalRead <= s.inlineThresh {
					buffered = append(buffered, chunk[:n]...)
				} else {
					// Exceeded inline threshold; transition to disk spooling
					spilled = true
					f, err := os.CreateTemp(s.dir, "spool-upload-*.tmp")
					if err != nil {
						return model.BodyRef{}, fmt.Errorf("create temp spool file: %w", err)
					}
					tmpFile = f
					tmpPath = f.Name()

					// Write buffered initial bytes
					if _, err := tmpFile.Write(buffered); err != nil {
						return model.BodyRef{}, fmt.Errorf("write buffer to temp file: %w", err)
					}
					// Write current chunk
					if _, err := tmpFile.Write(chunk[:n]); err != nil {
						return model.BodyRef{}, fmt.Errorf("write chunk to temp file: %w", err)
					}
					// Free in-memory buffer
					buffered = nil
				}
			} else {
				// Already spilled; write directly to temp file
				if _, err := tmpFile.Write(chunk[:n]); err != nil {
					return model.BodyRef{}, fmt.Errorf("write stream to temp file: %w", err)
				}
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return model.BodyRef{}, fmt.Errorf("read stream: %w", err)
		}
	}

	id := hex.EncodeToString(hasher.Sum(nil))

	if !spilled {
		return model.BodyRef{
			ID:          id,
			Size:        totalRead,
			ContentType: contentType,
			Encoding:    encoding,
			Inline:      buffered,
			Path:        "",
			Truncated:   truncated,
		}, nil
	}

	// Finalize disk file
	if err := tmpFile.Sync(); err != nil {
		return model.BodyRef{}, fmt.Errorf("sync temp spool file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return model.BodyRef{}, fmt.Errorf("close temp spool file: %w", err)
	}

	destPath := filepath.Join(s.dir, id)

	// Deduplication: if a file with this hash already exists, remove temp and reuse
	if _, err := os.Stat(destPath); err == nil {
		_ = os.Remove(tmpPath)
		tmpPath = ""
		tmpFile = nil
	} else {
		if err := os.Rename(tmpPath, destPath); err != nil {
			return model.BodyRef{}, fmt.Errorf("commit spool file %s: %w", destPath, err)
		}
		tmpPath = ""
		tmpFile = nil
	}

	// Enforce maxCacheBytes eviction
	_, _ = s.evictLocked()

	return model.BodyRef{
		ID:          id,
		Size:        totalRead,
		ContentType: contentType,
		Encoding:    encoding,
		Inline:      nil,
		Path:        destPath,
		Truncated:   truncated,
	}, nil
}

// Open returns an io.ReadCloser for the body content referenced by ref.
func (s *Spooler) Open(ref model.BodyRef) (io.ReadCloser, error) {
	if !ref.Spilled() {
		return io.NopCloser(bytes.NewReader(ref.Inline)), nil
	}

	f, err := os.Open(ref.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSpoolFileNotFound
		}
		return nil, fmt.Errorf("open spool file %s: %w", ref.Path, err)
	}
	return f, nil
}

// ReadBytes returns the entire body bytes (either from memory or from disk).
func (s *Spooler) ReadBytes(ref model.BodyRef) ([]byte, error) {
	if !ref.Spilled() {
		return ref.Inline, nil
	}
	rc, err := s.Open(ref)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

type fileEntry struct {
	path    string
	size    int64
	modTime time.Time
}

// Evict enforces MaxCacheBytes across all files in the spool directory,
// removing the oldest files first until the total size is within the limit.
// Returns the number of bytes reclaimed.
func (s *Spooler) Evict() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, ErrClosed
	}
	return s.evictLocked()
}

func (s *Spooler) evictLocked() (int64, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0, fmt.Errorf("read spool dir: %w", err)
	}

	var files []fileEntry
	var totalSize int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, fileEntry{
			path:    filepath.Join(s.dir, entry.Name()),
			size:    info.Size(),
			modTime: info.ModTime(),
		})
		totalSize += info.Size()
	}

	if totalSize <= s.maxCacheBytes {
		return 0, nil
	}

	// Sort oldest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	var reclaimed int64
	for _, f := range files {
		if totalSize-reclaimed <= s.maxCacheBytes {
			break
		}
		if err := os.Remove(f.path); err == nil {
			reclaimed += f.size
		}
	}

	return reclaimed, nil
}

// Close closes the spooler and, if CleanupOnClose was set, removes all spool files.
func (s *Spooler) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	if s.cleanupOnClose && s.dir != "" {
		return os.RemoveAll(s.dir)
	}
	return nil
}
