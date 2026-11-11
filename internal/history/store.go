// Package history records executed requests and the responses they produced.
//
// The store is an append-only JSONL file per collection plus an in-memory index.
// SQLite would be the obvious choice and is deliberately avoided: it needs a
// new driver dependency, and the access pattern here is append-mostly with
// occasional prefix deletion, which a file handles without a query planner.
//
// Retention follows D-011: rows older than the configured age are pruned, then
// the oldest rows are dropped until the store is under the configured size. Both
// bounds are settings, so a user who wants everything can disable pruning.
package history

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"fearless/internal/core/model"
)

// Defaults from D-011.
const (
	// DefaultRetainDays is the age limit for a row.
	DefaultRetainDays = 30
	// DefaultMaxBytes is the total size limit across history files.
	DefaultMaxBytes int64 = 200 << 20
)

// ErrClosed reports use of a closed store.
var ErrClosed = errors.New("history: store is closed")

// Entry is one recorded execution.
//
// It carries the metadata row D-011 requires: what was called, when, with which
// environment, and what came back. Bodies are not stored here; BodyRef points at
// the spool, which owns the bytes.
type Entry struct {
	// ID is unique within the store.
	ID string `json:"id"`
	// ExecutedAt is when the request was sent.
	ExecutedAt time.Time `json:"executedAt"`
	// CollectionUID and ItemUID locate the request.
	CollectionUID string `json:"collectionUid"`
	ItemUID       string `json:"itemUid"`
	// ItemName and ItemPath are the human-readable location, kept so the row
	// survives the request being renamed or deleted.
	ItemName string `json:"itemName,omitempty"`
	ItemPath string `json:"itemPath,omitempty"`
	// Environment is the active environment's name, or empty.
	Environment string `json:"environment,omitempty"`

	// Method and URL are the request as sent.
	Method string `json:"method"`
	URL    string `json:"url"`

	// Status and StatusText are the response status, or 0 when it never came.
	Status     int    `json:"status"`
	StatusText string `json:"statusText,omitempty"`
	// DurationMs is the total time in milliseconds.
	DurationMs int64 `json:"durationMs"`
	// Size is the body length in bytes.
	Size int64 `json:"size"`
	// Error is the failure reason; Status may still be set.
	Error string `json:"error,omitempty"`

	// BodyRef points at a retained response body. D-011 keeps bodies for failed
	// requests and explicit saves only, so this is usually nil on success.
	BodyRef *model.BodyRef `json:"bodyRef,omitempty"`
	// RequestHeaders and ResponseHeaders support the detail view.
	RequestHeaders  map[string][]string `json:"requestHeaders,omitempty"`
	ResponseHeaders map[string][]string `json:"responseHeaders,omitempty"`
	// AssertionSummary counts outcomes, when the request had assertions.
	AssertionsPassed int `json:"assertionsPassed,omitempty"`
	AssertionsFailed int `json:"assertionsFailed,omitempty"`
	// Tags are copied from the request so a saved filter keeps working after the
	// request is retagged or deleted.
	Tags []string `json:"tags,omitempty"`
}

// Failed reports whether the row records a failure.
//
// A transport error, a 4xx or a 5xx all count. A 3xx does not: the transport
// follows redirects, so a final 3xx means it stopped early for another reason and
// the user's attention belongs on the error field, which is set separately.
func (e *Entry) Failed() bool {
	if e == nil {
		return false
	}
	return e.Error != "" || e.Status >= 400
}

// Query filters and paginates history.
type Query struct {
	// CollectionUID restricts to one collection; empty means all.
	CollectionUID string
	// ItemUID restricts to one request; empty means any.
	ItemUID string
	// Since and Until bound ExecutedAt; zero means unbounded.
	Since time.Time
	Until time.Time
	// StatusClass restricts to a 1-5 class (2 for 2xx); 0 means any.
	StatusClass int
	// FailedOnly restricts to failed rows.
	FailedOnly bool
	// Search is a case-insensitive substring match against method, URL and item
	// name, which is what the history search box needs.
	Search string
	// Limit caps the returned rows; 0 means no cap.
	Limit int
}

// Summary aggregates a query without returning the rows.
type Summary struct {
	Total  int
	Passed int
	Failed int
}

// Options configure a store.
type Options struct {
	// Dir holds one JSONL file per collection.
	Dir string
	// RetainDays prunes rows older than this; negative disables age pruning.
	RetainDays int
	// MaxBytes caps the total size of the history files; negative disables size
	// pruning. Zero uses the default.
	MaxBytes int64
	// Now is injectable for deterministic retention tests.
	Now func() time.Time
}

// Store is a history store.
//
// Appends are serialised behind a mutex and only ever extend a file, which keeps
// the write path cheap enough not to stall a request completion. Reads take an
// index snapshot under a read lock.
type Store struct {
	opts Options

	mu      sync.RWMutex
	files   map[string]*os.File
	entries []*Entry // ascending by ExecutedAt
	closed  bool
	nextSeq uint64
	size    int64
}

// Open creates a store, loading any existing history.
func Open(opts Options) (*Store, error) {
	if opts.Dir == "" {
		return nil, errors.New("history: Dir is required")
	}
	if opts.RetainDays == 0 {
		opts.RetainDays = DefaultRetainDays
	}
	if opts.MaxBytes == 0 {
		opts.MaxBytes = DefaultMaxBytes
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("history: create %s: %w", opts.Dir, err)
	}

	s := &Store{opts: opts, files: make(map[string]*os.File)}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// load reads every history file into the index.
//
// A malformed line is skipped rather than failing the load. History is a log: a
// line truncated by a crash mid-append must not make the whole log unreadable,
// and refusing to start the application is a far worse outcome than losing that
// one row.
func (s *Store) load() error {
	paths, err := filepath.Glob(filepath.Join(s.opts.Dir, "*.jsonl"))
	if err != nil {
		return fmt.Errorf("history: list %s: %w", s.opts.Dir, err)
	}
	for _, path := range paths {
		if err := s.loadFile(path); err != nil {
			return err
		}
	}

	sort.SliceStable(s.entries, func(i, j int) bool {
		return s.entries[i].ExecutedAt.Before(s.entries[j].ExecutedAt)
	})
	return nil
}

// loadFile appends one file's entries to the index.
func (s *Store) loadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("history: open %s: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("history: stat %s: %w", path, err)
	}
	s.size += info.Size()

	scanner := bufio.NewScanner(file)
	// A row is small; the larger ceiling guards a pathological header set
	// without allocating a megabyte for every line.
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.ID == "" {
			continue
		}
		s.entries = append(s.entries, &entry)
	}
	return nil
}

// Record appends one entry and returns it with its assigned ID and timestamp.
func (s *Store) Record(entry Entry) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, ErrClosed
	}

	recorded := entry
	if recorded.ExecutedAt.IsZero() {
		recorded.ExecutedAt = s.opts.Now().UTC()
	}
	s.nextSeq++
	recorded.ID = fmt.Sprintf("%s-%06d", recorded.ExecutedAt.UTC().Format("20060102T150405"), s.nextSeq)

	encoded, err := json.Marshal(recorded)
	if err != nil {
		return nil, fmt.Errorf("history: encode entry: %w", err)
	}
	encoded = append(encoded, '\n')

	file, err := s.fileFor(recorded.CollectionUID)
	if err != nil {
		return nil, err
	}
	if _, err := file.Write(encoded); err != nil {
		return nil, fmt.Errorf("history: append: %w", err)
	}
	s.size += int64(len(encoded))

	stored := &recorded
	s.entries = append(s.entries, stored)

	// A failed prune must not fail the write: the row is already durable, and
	// reporting an error would invite a retry that duplicates it.
	_ = s.pruneLocked()
	return stored, nil
}

// fileFor returns the append handle for a collection, opening it if needed.
func (s *Store) fileFor(collectionUID string) (*os.File, error) {
	key := collectionKey(collectionUID)
	if file, ok := s.files[key]; ok {
		return file, nil
	}

	path := s.pathFor(key)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("history: open %s: %w", path, err)
	}
	s.files[key] = file
	return file, nil
}

// pathFor returns a collection's file path.
func (s *Store) pathFor(key string) string {
	return filepath.Join(s.opts.Dir, safeFilename(key)+".jsonl")
}

// Query returns matching entries, newest first.
func (s *Store) Query(q Query) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, ErrClosed
	}

	out := make([]Entry, 0, 16)
	// Newest first, so a limit keeps the most recent rows.
	for i := len(s.entries) - 1; i >= 0; i-- {
		entry := s.entries[i]
		if !matches(entry, q) {
			continue
		}
		out = append(out, *entry)
		if q.Limit > 0 && len(out) >= q.Limit {
			break
		}
	}
	return out, nil
}

// Summarize counts a query's matches without returning them.
func (s *Store) Summarize(q Query) (Summary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return Summary{}, ErrClosed
	}

	var summary Summary
	for _, entry := range s.entries {
		if !matches(entry, q) {
			continue
		}
		summary.Total++
		if entry.Failed() {
			summary.Failed++
			continue
		}
		summary.Passed++
	}
	return summary, nil
}

// Delete removes one entry by ID.
//
// The row is rewritten out of its file rather than tombstoned, so the on-disk
// size reflects what remains. That costs a rewrite of one collection's file,
// which is fine for an explicit delete and never happens on the write path.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrClosed
	}

	for i, entry := range s.entries {
		if entry.ID != id {
			continue
		}
		collection := collectionKey(entry.CollectionUID)
		s.entries = append(s.entries[:i], s.entries[i+1:]...)
		return s.rewriteLocked(collection)
	}
	return nil
}

// Clear removes every entry, for one collection or all of them.
func (s *Store) Clear(collectionUID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrClosed
	}

	if collectionUID != "" {
		kept := make([]*Entry, 0, len(s.entries))
		for _, entry := range s.entries {
			if entry.CollectionUID != collectionUID {
				kept = append(kept, entry)
			}
		}
		s.entries = kept
		return s.rewriteLocked(collectionKey(collectionUID))
	}

	for _, file := range s.files {
		_ = file.Close()
	}
	s.files = make(map[string]*os.File)
	s.entries = nil

	paths, err := filepath.Glob(filepath.Join(s.opts.Dir, "*.jsonl"))
	if err != nil {
		return fmt.Errorf("history: list: %w", err)
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("history: remove %s: %w", path, err)
		}
	}
	s.size = 0
	return nil
}

// Prune applies the retention policy and reports how many rows were dropped.
func (s *Store) Prune() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, ErrClosed
	}

	before := len(s.entries)
	if err := s.pruneLocked(); err != nil {
		return 0, err
	}
	return before - len(s.entries), nil
}

// pruneLocked enforces both bounds from D-011.
//
// Age goes first: a row past the age limit is gone regardless of size. Then size,
// oldest first, because the newest rows are the ones a user is looking at.
func (s *Store) pruneLocked() error {
	changed := make(map[string]bool)

	if s.opts.RetainDays >= 0 {
		cutoff := s.opts.Now().UTC().AddDate(0, 0, -s.opts.RetainDays)
		kept := s.entries[:0]
		for _, entry := range s.entries {
			if entry.ExecutedAt.Before(cutoff) {
				changed[collectionKey(entry.CollectionUID)] = true
				continue
			}
			kept = append(kept, entry)
		}
		s.entries = kept
	}

	if s.opts.MaxBytes > 0 {
		// The size figure is exact here because it is only decremented by the
		// encoded length that Record added, and corrected after every rewrite.
		for s.size > s.opts.MaxBytes && len(s.entries) > 0 {
			dropped := s.entries[0]
			changed[collectionKey(dropped.CollectionUID)] = true
			if encoded, err := json.Marshal(*dropped); err == nil {
				s.size -= int64(len(encoded)) + 1
			}
			s.entries = s.entries[1:]
		}
		if s.size < 0 {
			s.size = 0
		}
	}

	// Rewrite in a stable order so a failure partway leaves a predictable state.
	names := make([]string, 0, len(changed))
	for name := range changed {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := s.rewriteLocked(name); err != nil {
			return err
		}
	}
	return nil
}

// rewriteLocked rewrites one collection's file from the index.
//
// It writes a temporary file and renames it into place, so a crash mid-rewrite
// leaves the old file intact rather than a truncated one. The size is re-derived
// afterwards, which is cheaper than tracking every byte through three paths.
//
// The argument is a collection key, already normalised by collectionKey, so a raw
// UID and `_unknown` are both handled without a second lookup.
func (s *Store) rewriteLocked(key string) error {
	if file, ok := s.files[key]; ok {
		_ = file.Close()
		delete(s.files, key)
	}

	path := s.pathFor(key)

	var kept []*Entry
	for _, entry := range s.entries {
		if collectionKey(entry.CollectionUID) == key {
			kept = append(kept, entry)
		}
	}

	if len(kept) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("history: remove %s: %w", path, err)
		}
		s.recomputeSizeLocked()
		return nil
	}

	temp := path + ".tmp"
	out, err := os.Create(temp)
	if err != nil {
		return fmt.Errorf("history: create %s: %w", temp, err)
	}

	writer := bufio.NewWriter(out)
	for _, entry := range kept {
		encoded, err := json.Marshal(*entry)
		if err != nil {
			out.Close()
			os.Remove(temp)
			return fmt.Errorf("history: encode entry: %w", err)
		}
		encoded = append(encoded, '\n')
		if _, err := writer.Write(encoded); err != nil {
			out.Close()
			os.Remove(temp)
			return fmt.Errorf("history: write %s: %w", temp, err)
		}
	}
	if err := writer.Flush(); err != nil {
		out.Close()
		os.Remove(temp)
		return fmt.Errorf("history: flush %s: %w", temp, err)
	}
	if err := out.Close(); err != nil {
		os.Remove(temp)
		return fmt.Errorf("history: close %s: %w", temp, err)
	}
	if err := os.Rename(temp, path); err != nil {
		os.Remove(temp)
		return fmt.Errorf("history: replace %s: %w", path, err)
	}

	s.recomputeSizeLocked()
	return nil
}

// recomputeSizeLocked re-derives the on-disk total.
func (s *Store) recomputeSizeLocked() {
	paths, err := filepath.Glob(filepath.Join(s.opts.Dir, "*.jsonl"))
	if err != nil {
		return
	}
	var total int64
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			total += info.Size()
		}
	}
	s.size = total
}

// Close flushes and closes the store.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	var firstErr error
	for _, file := range s.files {
		if err := file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.files = nil
	return firstErr
}

// Size returns the on-disk total in bytes, for the settings display.
func (s *Store) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.size
}

// Count returns the number of entries held.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// matches reports whether an entry satisfies a query.
func matches(entry *Entry, q Query) bool {
	if q.CollectionUID != "" && entry.CollectionUID != q.CollectionUID {
		return false
	}
	if q.ItemUID != "" && entry.ItemUID != q.ItemUID {
		return false
	}
	if !q.Since.IsZero() && entry.ExecutedAt.Before(q.Since) {
		return false
	}
	if !q.Until.IsZero() && entry.ExecutedAt.After(q.Until) {
		return false
	}
	if q.StatusClass != 0 && entry.Status/100 != q.StatusClass {
		return false
	}
	if q.FailedOnly && !entry.Failed() {
		return false
	}
	if q.Search != "" && !entryContains(entry, q.Search) {
		return false
	}
	return true
}

// entryContains reports a case-insensitive substring match.
//
// Only method, URL, item name and path are searched. Headers and bodies are
// deliberately excluded: a token in a query string is already in the URL, and
// matching against a response body would turn a search box into a memory sink.
func entryContains(entry *Entry, needle string) bool {
	lower := strings.ToLower(needle)
	for _, field := range []string{entry.Method, entry.URL, entry.ItemName, entry.ItemPath} {
		if strings.Contains(strings.ToLower(field), lower) {
			return true
		}
	}
	return false
}

// collectionKey normalises a collection UID for use as a file key.
func collectionKey(uid string) string {
	if uid == "" {
		return "_unknown"
	}
	return uid
}

// safeFilename reduces a collection UID to a legal file stem.
//
// A UID is normally generated, but it comes from a file on disk and may be
// anything, so path separators are replaced rather than trusted.
func safeFilename(uid string) string {
	out := make([]rune, 0, len(uid))
	for _, r := range uid {
		switch r {
		case '/', '\\', ':', 0:
			out = append(out, '-')
		default:
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "_unknown"
	}
	return string(out)
}
