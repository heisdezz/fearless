package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fearless/internal/core/model"
)

func opts() Options {
	return Options{
		FollowRedirects:   true,
		MaxRedirects:      5,
		ForwardAuthHeader: true,
	}
}

// get builds a GET request to url.
func get(t *testing.T, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func TestSimpleGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	res, err := Execute(context.Background(), get(t, server.URL), opts())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Err != "" {
		t.Fatalf("result error: %s", res.Err)
	}
	if res.Status != 200 {
		t.Errorf("status = %d", res.Status)
	}
	if !strings.Contains(string(res.Body.Inline), `"ok":true`) {
		t.Errorf("body = %q", res.Body.Inline)
	}
	if res.Body.ContentType != "application/json" {
		t.Errorf("content type = %q", res.Body.ContentType)
	}
	if res.Duration <= 0 {
		t.Error("duration not measured")
	}
}

// TestRedirectMatrix covers RFC 7231 method rewriting, which is the part most
// implementations get wrong.
func TestRedirectMatrix(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		method     string
		wantMethod string
		wantBody   bool
	}{
		{"301 GET stays GET", 301, http.MethodGet, http.MethodGet, false},
		{"301 POST becomes GET", 301, http.MethodPost, http.MethodGet, false},
		{"302 POST becomes GET", 302, http.MethodPost, http.MethodGet, false},
		{"303 POST becomes GET", 303, http.MethodPost, http.MethodGet, false},
		{"303 PUT becomes GET", 303, http.MethodPut, http.MethodGet, false},
		{"307 POST preserved", 307, http.MethodPost, http.MethodPost, true},
		{"308 POST preserved", 308, http.MethodPost, http.MethodPost, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var finalMethod string
			var finalBody string
			var calls int

			mux := http.NewServeMux()
			mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
				finalMethod = r.Method
				b, _ := io.ReadAll(r.Body)
				finalBody = string(b)
				w.WriteHeader(http.StatusOK)
			})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/final" {
					mux.ServeHTTP(w, r)
					return
				}
				calls++
				w.Header().Set("Location", "/final")
				w.WriteHeader(tc.status)
			}))
			defer server.Close()

			var body io.Reader
			if tc.method == http.MethodPost || tc.method == http.MethodPut {
				body = strings.NewReader("payload")
			}
			req, err := http.NewRequest(tc.method, server.URL+"/start", body)
			if err != nil {
				t.Fatal(err)
			}
			if body != nil {
				req.Body = &replayReader{data: []byte("payload")}
			}

			res, err := Execute(context.Background(), req, opts())
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if res.Err != "" {
				t.Fatalf("result error: %s", res.Err)
			}
			if finalMethod != tc.wantMethod {
				t.Errorf("final method = %s, want %s", finalMethod, tc.wantMethod)
			}
			if tc.wantBody && finalBody != "payload" {
				t.Errorf("body not preserved: %q", finalBody)
			}
			if !tc.wantBody && finalBody != "" {
				t.Errorf("body should have been dropped, got %q", finalBody)
			}
			if len(res.Redirects) != 1 {
				t.Fatalf("recorded %d redirects, want 1", len(res.Redirects))
			}
			if res.Redirects[0].Status != tc.status {
				t.Errorf("recorded status = %d", res.Redirects[0].Status)
			}
		})
	}
}

// replayReader is a body that can be reset for a redirect.
type replayReader struct {
	data []byte
	pos  int
}

func (r *replayReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *replayReader) Close() error { return nil }

// Reset implements the replay contract redirectRequest looks for.
func (r *replayReader) Reset() (io.ReadCloser, error) {
	return &replayReader{data: r.data}, nil
}

func TestRedirectLimit(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Location", "/again")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	o := opts()
	o.MaxRedirects = 3

	res, err := Execute(context.Background(), get(t, server.URL), o)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	// The chain stops at the limit and the last response is reported.
	if res.Status != http.StatusFound {
		t.Errorf("final status = %d, want the last 302", res.Status)
	}
	if len(res.Redirects) != 3 {
		t.Errorf("followed %d redirects, want 3", len(res.Redirects))
	}
}

func TestRedirectsDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/elsewhere")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	o := opts()
	o.FollowRedirects = false

	res, err := Execute(context.Background(), get(t, server.URL), o)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Status != http.StatusFound {
		t.Errorf("status = %d, want 302 returned to the caller", res.Status)
	}
	if len(res.Redirects) != 0 {
		t.Errorf("recorded %d redirects, want none", len(res.Redirects))
	}
}

func TestRedirectWithoutLocationIsNotFollowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	res, _ := Execute(context.Background(), get(t, server.URL), opts())
	if res.Status != http.StatusFound {
		t.Errorf("status = %d", res.Status)
	}
	if len(res.Redirects) != 0 {
		t.Error("a redirect without Location must not be recorded")
	}
}

func TestRelativeRedirectResolution(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/a/b/start" {
			w.Header().Set("Location", "../target")
			w.WriteHeader(http.StatusFound)
			return
		}
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	res, _ := Execute(context.Background(), get(t, server.URL+"/a/b/start"), opts())
	if res.Err != "" {
		t.Fatalf("error: %s", res.Err)
	}
	if gotPath != "/a/target" {
		t.Errorf("resolved path = %q, want /a/target", gotPath)
	}
}

// TestAuthHeaderDroppedOnCrossHostRedirect is a security property: credentials
// must not follow a redirect to a different host.
func TestAuthHeaderDroppedOnCrossHostRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("Authorization leaked to a different host")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
	}))
	defer source.Close()

	req := get(t, source.URL)
	req.Header.Set("Authorization", "Bearer secret")

	o := opts()
	o.ForwardAuthHeader = false
	res, _ := Execute(context.Background(), req, o)
	if res.Err != "" {
		t.Fatalf("error: %s", res.Err)
	}
}

// TestAuthHeaderKeptWhenForwarding checks the opposite when the setting allows it.
func TestAuthHeaderKeptWhenForwarding(t *testing.T) {
	var seen string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
	}))
	defer source.Close()

	req := get(t, source.URL)
	req.Header.Set("Authorization", "Bearer secret")

	o := opts()
	o.ForwardAuthHeader = true
	_, _ = Execute(context.Background(), req, o)
	if seen != "Bearer secret" {
		t.Errorf("Authorization = %q, want it forwarded", seen)
	}
}

func TestCancellation(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	req := get(t, server.URL).WithContext(ctx)

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	res, err := Execute(ctx, req, opts())
	if err != nil {
		t.Fatalf("execute returned a programming error: %v", err)
	}
	if res.Err == "" {
		t.Fatal("expected the result to record an error")
	}
	if !strings.Contains(strings.ToLower(res.Err), "cancel") {
		t.Errorf("error = %q, want it to mention cancellation", res.Err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("cancellation took %v", elapsed)
	}
}

func TestTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)

	o := opts()
	o.TimeoutMs = 30

	res, err := Execute(context.Background(), get(t, server.URL), o)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Err == "" {
		t.Fatal("expected a timeout error")
	}
	if !strings.Contains(strings.ToLower(res.Err), "timeout") &&
		!strings.Contains(strings.ToLower(res.Err), "deadline") {
		t.Errorf("error = %q, want it to mention the timeout", res.Err)
	}
}

// TestStreamingChunks checks that chunks are delivered incrementally rather than
// after the whole body, which is what makes SSE work.
func TestStreamingChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for _, part := range []string{"data: one\n\n", "data: two\n\n", "data: three\n\n"} {
			_, _ = w.Write([]byte(part))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	var sink recordingSink
	o := opts()
	o.Sink = &sink

	res, err := Execute(context.Background(), get(t, server.URL), o)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Err != "" {
		t.Fatalf("error: %s", res.Err)
	}
	if len(sink.chunks) < 2 {
		t.Errorf("got %d chunks, want several", len(sink.chunks))
	}
	total := strings.Join(sink.chunks, "")
	if !strings.Contains(total, "data: three") {
		t.Errorf("chunks lost content: %q", total)
	}
}

// TestTimelineHasPhases checks the timeline records at least the phases that do
// not need a real network hop.
func TestTimelineHasPhases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	var sink recordingSink
	o := opts()
	o.Sink = &sink

	res, _ := Execute(context.Background(), get(t, server.URL), o)
	if len(res.Timeline) == 0 {
		t.Fatal("timeline is empty")
	}
	seen := map[model.TimelinePhase]bool{}
	for _, e := range res.Timeline {
		seen[e.Phase] = true
	}
	for _, want := range []model.TimelinePhase{model.PhaseRequestSent, model.PhaseDownload} {
		if !seen[want] {
			t.Errorf("timeline missing phase %s (have %v)", want, seen)
		}
	}
	if len(sink.phases) == 0 {
		t.Error("sink received no phase events")
	}
}

func TestNilRequestIsAProgrammingError(t *testing.T) {
	if _, err := Execute(context.Background(), nil, opts()); err == nil {
		t.Fatal("expected an error for a nil request")
	}
}

func TestTopLevelErrorPath(t *testing.T) {
	// A port nothing listens on: the transport must report, not panic.
	res, err := Execute(context.Background(), get(t, "http://127.0.0.1:1/"), opts())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Err == "" {
		t.Fatal("expected a connection error")
	}
}

// recordingSink captures chunks and phases.
type recordingSink struct {
	chunks []string
	phases []model.TimelineEntry
}

func (r *recordingSink) Chunk(data []byte) { r.chunks = append(r.chunks, string(data)) }
func (r *recordingSink) Phase(e model.TimelineEntry) {
	r.phases = append(r.phases, e)
}

func TestOptionsString(t *testing.T) {
	s := opts().String()
	if !strings.Contains(s, "redirects:true") {
		t.Errorf("String = %q", s)
	}
}

func TestSentRequestCaptured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req := get(t, server.URL)
	req.Header.Set("X-Test", "1")

	res, _ := Execute(context.Background(), req, opts())
	if res.Sent.Method != http.MethodGet {
		t.Errorf("sent method = %q", res.Sent.Method)
	}
	if !strings.HasPrefix(res.Sent.URL, server.URL) {
		t.Errorf("sent url = %q", res.Sent.URL)
	}
	if v, ok := res.Sent.Header("X-Test"); !ok || v != "1" {
		t.Errorf("sent header missing: %v", res.Sent.Headers)
	}
}
