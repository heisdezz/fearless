package transport

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

// TestProxyAttemptsFallsThroughOnDialFailure is the B-020 regression test.
//
// PAC returns a ranked list: "PROXY a; PROXY b; DIRECT" means try a, then b,
// then connect directly. A dead first proxy must not fail the request when a
// later attempt can still satisfy it.
func TestProxyAttemptsFallsThroughOnDialFailure(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("reached"))
	}))
	defer target.Close()

	// A proxy that was alive once and is gone now: bind, then close.
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL, err := url.Parse(dead.URL)
	if err != nil {
		t.Fatal(err)
	}
	dead.Close()

	o := opts()
	o.TimeoutMs = 5000
	o.ProxyAttempts = []*url.URL{deadURL, nil}

	res, err := Execute(context.Background(), get(t, target.URL), o)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Err != "" {
		t.Fatalf("request failed instead of falling through to DIRECT: %s", res.Err)
	}
	if res.Status != http.StatusOK {
		t.Errorf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if got := string(res.Body.Inline); got != "reached" {
		t.Errorf("body = %q, want %q", got, "reached")
	}
}

// TestProxyAttemptsUsesSecondProxy checks fallback between two live proxies, so
// the test does not depend on the DIRECT path happening to work.
func TestProxyAttemptsUsesSecondProxy(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("via proxy"))
	}))
	defer target.Close()

	var secondHit bool
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondHit = true
		// Minimal forward proxy: dial the absolute URL the client asked for.
		out, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		out.Header = r.Header.Clone()
		resp, err := http.DefaultTransport.RoundTrip(out)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer second.Close()

	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL, err := url.Parse(dead.URL)
	if err != nil {
		t.Fatal(err)
	}
	dead.Close()

	secondURL, err := url.Parse(second.URL)
	if err != nil {
		t.Fatal(err)
	}

	o := opts()
	o.TimeoutMs = 5000
	o.ProxyAttempts = []*url.URL{deadURL, secondURL}

	res, err := Execute(context.Background(), get(t, target.URL), o)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Err != "" {
		t.Fatalf("request failed instead of using the second proxy: %s", res.Err)
	}
	if !secondHit {
		t.Error("second proxy was never reached")
	}
}

// TestProxyAttemptsStopsAtFirstSuccess pins that a healthy first attempt is not
// retried against the rest of the chain.
func TestProxyAttemptsStopsAtFirstSuccess(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer target.Close()

	var secondHit bool
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondHit = true
		http.Error(w, "should not be reached", http.StatusInternalServerError)
	}))
	defer second.Close()

	secondURL, err := url.Parse(second.URL)
	if err != nil {
		t.Fatal(err)
	}

	o := opts()
	o.ProxyAttempts = []*url.URL{nil, secondURL}

	res, err := Execute(context.Background(), get(t, target.URL), o)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Err != "" {
		t.Fatalf("request failed: %s", res.Err)
	}
	if secondHit {
		t.Error("a successful DIRECT attempt still tried the next proxy")
	}
}

// TestRetryableAcrossAttempts pins which failures may fall through. A cancelled
// or timed-out request must not silently retry against another proxy.
func TestRetryableAcrossAttempts(t *testing.T) {
	connErr := &net.OpError{Op: "dial", Net: "tcp", Err: os.ErrDeadlineExceeded}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"dial failure", connErr, true},
		{"plain error", errString("connection refused"), true},
		{"cancelled", context.Canceled, false},
		{"deadline", context.DeadlineExceeded, false},
		{"wrapped cancel", wrappedErr{context.Canceled}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := retryableAcrossAttempts(tc.err); got != tc.want {
				t.Errorf("retryableAcrossAttempts(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

type errString string

func (e errString) Error() string { return string(e) }

type wrappedErr struct{ inner error }

func (w wrappedErr) Error() string { return "wrapped: " + w.inner.Error() }

func (w wrappedErr) Unwrap() error { return w.inner }
