package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fearless/internal/core/model"
)

// fixedNow returns a clock pinned to a test-controlled instant.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock(at time.Time) *clock { return &clock{now: at} }

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// openTemp creates a store in a temporary directory.
func openTemp(t *testing.T, opts Options) *Store {
	t.Helper()
	if opts.Dir == "" {
		opts.Dir = t.TempDir()
	}
	store, err := Open(opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// TestRecordAssignsIdentity covers the ID and timestamp assignment.
func TestRecordAssignsIdentity(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	store := openTemp(t, Options{Dir: t.TempDir(), Now: func() time.Time { return at }})

	first, err := store.Record(Entry{CollectionUID: "c1", ItemUID: "i1", Method: "GET", URL: "https://x/"})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if first.ID != "20260102T030405-000001" {
		t.Errorf("id = %q", first.ID)
	}
	if !first.ExecutedAt.Equal(at) {
		t.Errorf("executedAt = %v", first.ExecutedAt)
	}

	second, err := store.Record(Entry{CollectionUID: "c1"})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	// Two rows in the same second must still be distinguishable.
	if second.ID == first.ID {
		t.Fatalf("duplicate id %q", first.ID)
	}
}

// TestRecordPreservesCallerTimestamp keeps an explicitly timed row.
func TestRecordPreservesCallerTimestamp(t *testing.T) {
	explicit := time.Date(2020, 5, 5, 5, 5, 5, 0, time.UTC)
	store := openTemp(t, Options{Dir: t.TempDir()})

	entry, err := store.Record(Entry{CollectionUID: "c", ExecutedAt: explicit})
	if err != nil {
		t.Fatal(err)
	}
	if !entry.ExecutedAt.Equal(explicit) {
		t.Errorf("executedAt = %v, want %v", entry.ExecutedAt, explicit)
	}
}

// TestQueryIsNewestFirst covers ordering, which the UI depends on.
func TestQueryIsNewestFirst(t *testing.T) {
	clk := newClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := openTemp(t, Options{Dir: t.TempDir(), Now: clk.Now})

	for i := 0; i < 3; i++ {
		if _, err := store.Record(Entry{CollectionUID: "c", URL: "u"}); err != nil {
			t.Fatal(err)
		}
		clk.Advance(time.Minute)
	}

	rows, err := store.Query(Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d", len(rows))
	}
	for i := 1; i < len(rows); i++ {
		if rows[i].ExecutedAt.After(rows[i-1].ExecutedAt) {
			t.Fatalf("rows are not newest first at %d", i)
		}
	}
}

// TestQueryFilters covers every filter field.
func TestQueryFilters(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := newClock(base)
	store := openTemp(t, Options{Dir: t.TempDir(), Now: clk.Now})

	rows := []Entry{
		{CollectionUID: "c1", ItemUID: "i1", Method: "GET", URL: "https://api/ok", Status: 200, ItemName: "get ok"},
		{CollectionUID: "c1", ItemUID: "i2", Method: "POST", URL: "https://api/fail", Status: 500},
		{CollectionUID: "c2", ItemUID: "i3", Method: "GET", URL: "https://other/", Status: 404},
		{CollectionUID: "c2", ItemUID: "i4", Method: "GET", URL: "https://other/err", Error: "dial tcp: refused"},
	}
	for _, row := range rows {
		if _, err := store.Record(row); err != nil {
			t.Fatal(err)
		}
		clk.Advance(time.Second)
	}

	cases := []struct {
		name string
		q    Query
		want int
	}{
		{"all", Query{}, 4},
		{"by collection", Query{CollectionUID: "c1"}, 2},
		{"by item", Query{ItemUID: "i2"}, 1},
		{"5xx", Query{StatusClass: 5}, 1},
		{"4xx", Query{StatusClass: 4}, 1},
		{"failed only", Query{FailedOnly: true}, 3},
		// Search is case-insensitive across method, URL and name.
		{"search method", Query{Search: "post"}, 1},
		{"search url", Query{Search: "api/"}, 2},
		{"search name", Query{Search: "GET OK"}, 1},
		{"search miss", Query{Search: "nothing"}, 0},
		{"limit", Query{Limit: 2}, 2},
		{"since", Query{Since: base.Add(2 * time.Second)}, 2},
		{"until", Query{Until: base.Add(time.Second)}, 2},
	}

	for _, tc := range cases {
		got, err := store.Query(tc.q)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(got) != tc.want {
			t.Errorf("%s: got %d rows, want %d", tc.name, len(got), tc.want)
		}
	}
}

// TestLimitKeepsNewest pins that a limit truncates the tail, not the head.
func TestLimitKeepsNewest(t *testing.T) {
	clk := newClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := openTemp(t, Options{Dir: t.TempDir(), Now: clk.Now})

	for i := 0; i < 5; i++ {
		if _, err := store.Record(Entry{CollectionUID: "c", URL: string(rune('a' + i))}); err != nil {
			t.Fatal(err)
		}
		clk.Advance(time.Minute)
	}

	rows, err := store.Query(Query{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	if rows[0].URL != "e" || rows[1].URL != "d" {
		t.Errorf("kept %q, %q; want the two newest", rows[0].URL, rows[1].URL)
	}
}

// TestFailedClassification covers what counts as a failure.
func TestFailedClassification(t *testing.T) {
	cases := []struct {
		name string
		row  Entry
		want bool
	}{
		{"200", Entry{Status: 200}, false},
		{"204", Entry{Status: 204}, false},
		{"301", Entry{Status: 301}, false},
		{"400", Entry{Status: 400}, true},
		{"404", Entry{Status: 404}, true},
		{"500", Entry{Status: 500}, true},
		{"transport error with no status", Entry{Error: "refused"}, true},
		{"transport error with a status", Entry{Status: 200, Error: "reset"}, true},
		{"zero status", Entry{}, false},
	}
	for _, tc := range cases {
		if got := tc.row.Failed(); got != tc.want {
			t.Errorf("%s: failed = %v, want %v", tc.name, got, tc.want)
		}
	}

	// A nil entry is a programming error, not a failure.
	var nilEntry *Entry
	if nilEntry.Failed() {
		t.Error("nil entry reported as failed")
	}
}

// TestSummarize covers the aggregate counts.
func TestSummarize(t *testing.T) {
	store := openTemp(t, Options{Dir: t.TempDir()})
	for _, status := range []int{200, 201, 404, 500} {
		if _, err := store.Record(Entry{CollectionUID: "c", Status: status}); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := store.Summarize(Query{CollectionUID: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 4 || summary.Passed != 2 || summary.Failed != 2 {
		t.Errorf("summary = %+v", summary)
	}

	// The same filters apply.
	summary, err = store.Summarize(Query{FailedOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 2 {
		t.Errorf("failed summary = %+v", summary)
	}
}

// TestPersistenceAcrossReopen is the important durability property: a restart
// must not lose history.
func TestPersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)

	store, err := Open(Options{Dir: dir, Now: func() time.Time { return at }})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := store.Record(Entry{
			CollectionUID: "c1",
			Method:        "GET",
			URL:           "https://example.com/" + string(rune('a'+i)),
			Status:        200,
			BodyRef:       &model.BodyRef{ID: "h", Size: 3},
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	if got := reopened.Count(); got != 3 {
		t.Fatalf("count after reopen = %d", got)
	}
	rows, err := reopened.Query(Query{})
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].BodyRef == nil || rows[0].BodyRef.Size != 3 {
		t.Errorf("body ref lost across reopen: %+v", rows[0].BodyRef)
	}
}

// TestMalformedLineIsSkipped keeps a truncated row from making the log
// unreadable.
func TestMalformedLineIsSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c1.jsonl")

	good, _ := json.Marshal(Entry{ID: "a-1", CollectionUID: "c1", URL: "https://good/"})
	content := string(good) + "\n" + `{"id":"truncated"` + "\n" + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	if got := store.Count(); got != 1 {
		t.Fatalf("count = %d, want the one readable row", got)
	}
	rows, _ := store.Query(Query{})
	if rows[0].URL != "https://good/" {
		t.Errorf("row = %+v", rows[0])
	}
}

// TestAgePruning covers the age bound from D-011.
func TestAgePruning(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	clk := newClock(base)
	dir := t.TempDir()

	// Seed the file first, then open a store with a short retention so the stale
	// row is present at load time and only the explicit prune removes it.
	seed := openTemp(t, Options{Dir: dir, RetainDays: -1, MaxBytes: -1, Now: clk.Now})
	if _, err := seed.Record(Entry{CollectionUID: "c", URL: "old", ExecutedAt: base.AddDate(0, 0, -40)}); err != nil {
		t.Fatal(err)
	}
	if _, err := seed.Record(Entry{CollectionUID: "c", URL: "new"}); err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}

	store := openTemp(t, Options{Dir: dir, RetainDays: 30, Now: clk.Now})
	if got := store.Count(); got != 2 {
		t.Fatalf("count after load = %d, want both rows", got)
	}

	dropped, err := store.Prune()
	if err != nil {
		t.Fatal(err)
	}
	if dropped != 1 {
		t.Errorf("dropped = %d, want 1", dropped)
	}

	rows, _ := store.Query(Query{})
	if len(rows) != 1 || rows[0].URL != "new" {
		t.Fatalf("remaining rows = %+v", rows)
	}

	// The file must reflect the prune, not just the index: reopening proves it.
	reopened := openTemp(t, Options{Dir: dir, RetainDays: 30, Now: clk.Now})
	if got := reopened.Count(); got != 1 {
		t.Errorf("count after reopen = %d, want 1", got)
	}
}

// TestRecordPrunesInline keeps retention from waiting for an explicit prune.
func TestRecordPrunesInline(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	store := openTemp(t, Options{Dir: t.TempDir(), RetainDays: 1, Now: func() time.Time { return base }})

	if _, err := store.Record(Entry{CollectionUID: "c", URL: "stale", ExecutedAt: base.AddDate(0, 0, -5)}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record(Entry{CollectionUID: "c", URL: "fresh"}); err != nil {
		t.Fatal(err)
	}

	rows, _ := store.Query(Query{})
	for _, row := range rows {
		if row.URL == "stale" {
			t.Error("a stale row survived an inline prune")
		}
	}
}

// TestAgePruningDisabled keeps a negative setting meaning "never".
func TestAgePruningDisabled(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	store := openTemp(t, Options{Dir: t.TempDir(), RetainDays: -1, MaxBytes: -1, Now: func() time.Time { return base }})

	if _, err := store.Record(Entry{CollectionUID: "c", ExecutedAt: base.AddDate(0, 0, -4000)}); err != nil {
		t.Fatal(err)
	}
	if got := store.Count(); got != 1 {
		t.Errorf("count = %d, a disabled age limit must keep the row", got)
	}
}

// TestSizePruningDropsOldest covers the size bound.
func TestSizePruningDropsOldest(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	clk := newClock(base)

	// Measure one row so the cap can be set to hold roughly three.
	probe := openTemp(t, Options{Dir: t.TempDir(), RetainDays: -1, MaxBytes: -1, Now: clk.Now})
	row, err := probe.Record(Entry{CollectionUID: "c", Method: "GET", URL: "https://example.com/path", Status: 200})
	if err != nil {
		t.Fatal(err)
	}
	single := int64(len(mustJSON(t, row))) + 1

	dir := t.TempDir()
	store := openTemp(t, Options{Dir: dir, RetainDays: -1, MaxBytes: single * 3, Now: clk.Now})

	for i := 0; i < 6; i++ {
		if _, err := store.Record(Entry{CollectionUID: "c", URL: string(rune('a' + i))}); err != nil {
			t.Fatal(err)
		}
		clk.Advance(time.Minute)
	}

	if store.Size() > single*3+single {
		t.Errorf("size = %d, cap = %d", store.Size(), single*3)
	}
	rows, _ := store.Query(Query{})
	if len(rows) == 0 {
		t.Fatal("size pruning removed everything")
	}
	// The newest row must survive, because pruning drops oldest first.
	if rows[0].URL != "f" {
		t.Errorf("newest row = %q, want f", rows[0].URL)
	}
	// And the oldest must be gone.
	for _, r := range rows {
		if r.URL == "a" {
			t.Error("the oldest row survived size pruning")
		}
	}
}

// TestDeleteRemovesOneRow covers explicit deletion.
func TestDeleteRemovesOneRow(t *testing.T) {
	dir := t.TempDir()
	store := openTemp(t, Options{Dir: dir})

	first, _ := store.Record(Entry{CollectionUID: "c", URL: "a"})
	if _, err := store.Record(Entry{CollectionUID: "c", URL: "b"}); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(first.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	rows, _ := store.Query(Query{})
	if len(rows) != 1 || rows[0].URL != "b" {
		t.Fatalf("rows = %+v", rows)
	}

	// The deletion is durable.
	reopened := openTemp(t, Options{Dir: dir})
	if got := reopened.Count(); got != 1 {
		t.Errorf("count after reopen = %d", got)
	}

	// Deleting an unknown ID is a no-op, not an error.
	if err := store.Delete("nonexistent"); err != nil {
		t.Errorf("err = %v", err)
	}
}

// TestClear covers both scopes.
func TestClear(t *testing.T) {
	dir := t.TempDir()
	store := openTemp(t, Options{Dir: dir})

	if _, err := store.Record(Entry{CollectionUID: "c1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record(Entry{CollectionUID: "c2"}); err != nil {
		t.Fatal(err)
	}

	if err := store.Clear("c1"); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := store.Count(); got != 1 {
		t.Fatalf("count = %d", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "c1.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Error("the cleared collection's file should be gone")
	}

	if err := store.Clear(""); err != nil {
		t.Fatalf("clear all: %v", err)
	}
	if got := store.Count(); got != 0 {
		t.Errorf("count = %d", got)
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if len(paths) != 0 {
		t.Errorf("files remain: %v", paths)
	}
}

// TestConcurrentWritesDoNotCorrupt is the -race property and the "write never
// blocks the UI" requirement: many goroutines recording at once must produce a
// consistent log.
func TestConcurrentWritesDoNotCorrupt(t *testing.T) {
	dir := t.TempDir()
	store := openTemp(t, Options{Dir: dir, RetainDays: -1, MaxBytes: -1})

	const workers = 8
	const perWorker = 25
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				if _, err := store.Record(Entry{
					CollectionUID: "shared",
					ItemUID:       string(rune('a' + n)),
					URL:           "https://example.com/",
				}); err != nil {
					t.Errorf("record: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	want := workers * perWorker
	if got := store.Count(); got != want {
		t.Fatalf("count = %d, want %d", got, want)
	}

	// Every row must be readable from disk, with a unique ID.
	reopened := openTemp(t, Options{Dir: dir})
	rows, err := reopened.Query(Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != want {
		t.Fatalf("rows on disk = %d, want %d", len(rows), want)
	}
	ids := make(map[string]bool, len(rows))
	for _, row := range rows {
		if ids[row.ID] {
			t.Fatalf("duplicate id %q", row.ID)
		}
		ids[row.ID] = true
	}
}

// TestReadsDuringWritesDoNotRace exercises the read lock against a writer.
func TestReadsDuringWritesDoNotRace(t *testing.T) {
	store := openTemp(t, Options{Dir: t.TempDir(), RetainDays: -1, MaxBytes: -1})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			_, _ = store.Record(Entry{CollectionUID: "c", Status: 200})
		}
	}()

	for {
		select {
		case <-done:
			return
		default:
			if _, err := store.Query(Query{Limit: 5}); err != nil {
				t.Errorf("query: %v", err)
				return
			}
			if _, err := store.Summarize(Query{}); err != nil {
				t.Errorf("summarize: %v", err)
				return
			}
			_ = store.Size()
			_ = store.Count()
		}
	}
}

// TestClosedStoreReports covers the lifecycle guard.
func TestClosedStoreReports(t *testing.T) {
	store, err := Open(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record(Entry{CollectionUID: "c"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	// Closing twice is not an error, so a deferred close cannot mask a real one.
	if err := store.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}

	if _, err := store.Record(Entry{}); !errors.Is(err, ErrClosed) {
		t.Errorf("record after close: %v", err)
	}
	if _, err := store.Query(Query{}); !errors.Is(err, ErrClosed) {
		t.Errorf("query after close: %v", err)
	}
	if _, err := store.Summarize(Query{}); !errors.Is(err, ErrClosed) {
		t.Errorf("summarize after close: %v", err)
	}
	if err := store.Delete("x"); !errors.Is(err, ErrClosed) {
		t.Errorf("delete after close: %v", err)
	}
	if err := store.Clear(""); !errors.Is(err, ErrClosed) {
		t.Errorf("clear after close: %v", err)
	}
	if _, err := store.Prune(); !errors.Is(err, ErrClosed) {
		t.Errorf("prune after close: %v", err)
	}
}

// TestOpenRequiresDir is a programming-error guard.
func TestOpenRequiresDir(t *testing.T) {
	if _, err := Open(Options{}); err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}

// TestUnsafeCollectionUIDStaysInDirectory stops a crafted UID escaping.
func TestUnsafeCollectionUIDStaysInDirectory(t *testing.T) {
	dir := t.TempDir()
	store := openTemp(t, Options{Dir: dir})

	if _, err := store.Record(Entry{CollectionUID: "../../evil"}); err != nil {
		t.Fatal(err)
	}

	// No file may exist above the store directory.
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "evil.jsonl")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a traversal UID wrote outside the store directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("files in store = %d, want 1", len(entries))
	}
}

// TestUnknownCollectionIsRecorded covers rows with no collection, which happens
// when a request is executed outside a collection.
func TestUnknownCollectionIsRecorded(t *testing.T) {
	dir := t.TempDir()
	store := openTemp(t, Options{Dir: dir})

	if _, err := store.Record(Entry{Method: "GET", URL: "https://x/"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "_unknown.jsonl")); err != nil {
		t.Errorf("unknown-collection file missing: %v", err)
	}
}

// TestDefaultsAreApplied covers the option defaults.
func TestDefaultsAreApplied(t *testing.T) {
	store, err := Open(Options{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if store.opts.RetainDays != DefaultRetainDays {
		t.Errorf("retain days = %d", store.opts.RetainDays)
	}
	if store.opts.MaxBytes != DefaultMaxBytes {
		t.Errorf("max bytes = %d", store.opts.MaxBytes)
	}
}

// mustJSON encodes a value for size arithmetic in tests.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	encoded, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
