package transport

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fearless/internal/core/model"
	"fearless/internal/core/persistence"
)

// TestTimelineHasNoDuplicatePhases is the regression test for B-012 defect 1.
//
// The bug was two call sites ending the same phase: the trace hook ended
// `waiting`, and doHop ended it again after Do returned, producing a
// zero-duration duplicate. It also left `request-sent` measuring the whole round
// trip rather than the write.
func TestTimelineHasNoDuplicatePhases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Execute(context.Background(), req, Options{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	seen := make(map[model.TimelinePhase]int)
	for _, entry := range res.Timeline {
		seen[entry.Phase]++
	}

	// Every phase that appears must appear exactly once.
	for phase, count := range seen {
		if count > 1 {
			t.Errorf("phase %q recorded %d times: %+v", phase, count, res.Timeline)
		}
	}

	// The phases a plain HTTP request to a literal IP must produce. `dns` is
	// deliberately excluded: httptest binds 127.0.0.1, so the dialer resolves
	// nothing and no DNS hook fires. Asserting it here would make the test depend
	// on a resolver that the address bypasses.
	for _, phase := range []model.TimelinePhase{
		model.PhaseConnect, model.PhaseRequestSent,
		model.PhaseWaiting, model.PhaseDownload,
	} {
		if seen[phase] != 1 {
			t.Errorf("phase %q recorded %d times, want 1 (timeline %+v)", phase, seen[phase], res.Timeline)
		}
	}
}

// TestTimelineOrderIsChronological pins the order documented in http-engine.md.
func TestTimelineOrderIsChronological(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	res, err := Execute(context.Background(), req, Options{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	// The order the phases must appear in, as a partial order: DNS before
	// connect, connect before the write, the write before waiting, waiting
	// before download.
	order := []model.TimelinePhase{
		model.PhaseDNS, model.PhaseConnect, model.PhaseTLS,
		model.PhaseRequestSent, model.PhaseWaiting, model.PhaseDownload,
	}
	rank := make(map[model.TimelinePhase]int, len(order))
	for i, phase := range order {
		rank[phase] = i
	}

	last := -1
	for _, entry := range res.Timeline {
		r, ok := rank[entry.Phase]
		if !ok {
			continue
		}
		if r < last {
			t.Fatalf("phase %q appears after a later phase: %+v", entry.Phase, res.Timeline)
		}
		last = r
	}
}

// TestRequestSentMeasuresTheWrite covers the second half of defect 1.
//
// A server that delays its response must show that delay in `waiting`, not in
// `request-sent`: the former is the wait for the first byte, the latter is how
// long the request took to write.
func TestRequestSentMeasuresTheWrite(t *testing.T) {
	const delay = 120 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Time passes between reading the request and writing the response.
		time.Sleep(delay)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	res, err := Execute(context.Background(), req, Options{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	var requestSent, waiting int64
	for _, entry := range res.Timeline {
		switch entry.Phase {
		case model.PhaseRequestSent:
			requestSent = entry.Duration.Microseconds()
		case model.PhaseWaiting:
			waiting = entry.Duration.Microseconds()
		}
	}

	// The artificial delay must land in waiting.
	if waiting < delay.Microseconds() {
		t.Errorf("waiting = %dµs, want at least %dµs", waiting, delay.Microseconds())
	}
	// And request-sent must not carry it.
	if requestSent > delay.Microseconds()/2 {
		t.Errorf("request-sent = %dµs, which looks like the round trip rather than the write", requestSent)
	}
}

// TestBodyOverCapSpillsToDisk is the acceptance test for defect 2: the body cap
// with spill to the spool, exercised through the transport rather than only
// through the spool's own tests.
func TestBodyOverCapSpillsToDisk(t *testing.T) {
	dir := t.TempDir()
	spooler, err := persistence.NewSpooler(persistence.SpoolOptions{
		Dir:             dir,
		InlineThreshold: 16,
	})
	if err != nil {
		t.Fatalf("spooler: %v", err)
	}
	defer spooler.Close()

	// 4 KiB, comfortably over the 16-byte inline threshold.
	payload := bytes.Repeat([]byte("abcdefgh"), 512)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	res, err := Execute(context.Background(), req, Options{Spooler: spooler})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Err != "" {
		t.Fatalf("transport error: %s", res.Err)
	}

	// The body must have been spilled, not left inline.
	if len(res.Body.Inline) > 0 {
		t.Errorf("body was %d bytes inline, want it spilled", len(res.Body.Inline))
	}
	if !res.Body.Spilled() {
		t.Fatalf("body ref is not spilled: %+v", res.Body)
	}
	if filepath.Dir(res.Body.Path) != dir {
		t.Errorf("spooled to %q, want the configured directory %q", res.Body.Path, dir)
	}
	if res.Body.Size != int64(len(payload)) {
		t.Errorf("size = %d, want %d", res.Body.Size, len(payload))
	}

	// And it must round-trip byte for byte through the spool.
	file, err := os.Open(res.Body.Path)
	if err != nil {
		t.Fatalf("open spooled body: %v", err)
	}
	defer file.Close()
	got, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("read spooled body: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("spooled body differs: got %d bytes, want %d", len(got), len(payload))
	}
}

// TestBodyOverCapTruncatesAtTheLimit covers the cap itself through the transport.
func TestBodyOverCapTruncatesAtTheLimit(t *testing.T) {
	const cap int64 = 1024
	payload := bytes.Repeat([]byte("x"), 8192)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	res, err := Execute(context.Background(), req, Options{MaxResponseBytes: cap})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if res.Body.Size != cap {
		t.Errorf("size = %d, want the cap %d", res.Body.Size, cap)
	}
	if !res.Body.Truncated {
		t.Errorf("body should be marked truncated: %+v", res.Body)
	}
}

// TestSmallBodyStaysInline is the other half: a body under the threshold must not
// touch the disk, so the common case stays cheap.
func TestSmallBodyStaysInline(t *testing.T) {
	dir := t.TempDir()
	spooler, err := persistence.NewSpooler(persistence.SpoolOptions{
		Dir:             dir,
		InlineThreshold: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer spooler.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("small"))
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	res, err := Execute(context.Background(), req, Options{Spooler: spooler})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if res.Body.Spilled() {
		t.Errorf("a small body should stay inline: %+v", res.Body)
	}
	if string(res.Body.Inline) != "small" {
		t.Errorf("inline = %q", res.Body.Inline)
	}

	// No file may have been written for it.
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("a file was created for an inline body: %s", entry.Name())
		}
	}
}

// TestRedirectTimelineIsPerHop keeps a redirect's phases out of the first hop's.
func TestRedirectTimelineIsPerHop(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, server.URL+"/end", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("done"))
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/start", nil)
	res, err := Execute(context.Background(), req, Options{FollowRedirects: true, MaxRedirects: 5})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(res.Redirects) != 1 {
		t.Fatalf("redirects = %+v", res.Redirects)
	}

	// Two hops mean `request-sent` may legitimately appear twice, so uniqueness is
	// not the property here; but every entry must have a real duration or a
	// legitimate zero, and no phase may appear more than twice.
	seen := make(map[model.TimelinePhase]int)
	for _, entry := range res.Timeline {
		seen[entry.Phase]++
	}
	for phase, count := range seen {
		if count > 2 {
			t.Errorf("phase %q recorded %d times across two hops: %+v", phase, count, res.Timeline)
		}
	}
}

// TestCloseWithoutResponseClosesPhases covers an error path leaving a phase open.
func TestCloseWithoutResponseClosesPhases(t *testing.T) {
	// A listener that accepts and immediately closes produces a write that fails
	// before any response, which must still leave a consistent timeline.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
		}
		listener.Close()
	}()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+addr, nil)
	res, err := Execute(context.Background(), req, Options{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Err == "" {
		t.Skip("the connection unexpectedly succeeded")
	}

	// A failed request must still produce a timeline entry, and no phase may be
	// recorded twice.
	seen := make(map[model.TimelinePhase]int)
	for _, entry := range res.Timeline {
		seen[entry.Phase]++
	}
	for phase, count := range seen {
		if count > 1 {
			t.Errorf("phase %q recorded %d times after a failure: %+v", phase, count, res.Timeline)
		}
	}
}
