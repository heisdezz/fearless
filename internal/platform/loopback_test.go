package platform

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestLoopbackReceivesCallback covers the happy path: a provider redirect lands
// on the listener and its query becomes available.
func TestLoopbackReceivesCallback(t *testing.T) {
	l, err := NewLoopback("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()

	uri := l.RedirectURI()
	if !strings.HasPrefix(uri, "http://127.0.0.1:") || !strings.HasSuffix(uri, DefaultCallbackPath) {
		t.Fatalf("redirect URI = %q", uri)
	}

	// The redirect URI must be usable as-is, so the client uses it verbatim.
	go func() {
		resp, err := http.Get(uri + "?code=abc123&state=xyz")
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query, err := l.Wait(ctx)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if query.Get("code") != "abc123" {
		t.Errorf("code = %q", query.Get("code"))
	}
	if query.Get("state") != "xyz" {
		t.Errorf("state = %q", query.Get("state"))
	}
}

// TestLoopbackReportsProviderError keeps an error redirect readable.
func TestLoopbackReportsProviderError(t *testing.T) {
	l, err := NewLoopback("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	go func() {
		resp, err := http.Get(l.RedirectURI() + "?error=access_denied&error_description=User+denied")
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query, err := l.Wait(ctx)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	// An error redirect is a successful callback carrying bad news; the caller
	// inspects the query rather than treating it as a transport failure.
	if query.Get("error") != "access_denied" {
		t.Errorf("error = %q", query.Get("error"))
	}
}

// TestLoopbackSecondRequestIsRefused stops a stray tab replacing the code.
func TestLoopbackSecondRequestIsRefused(t *testing.T) {
	l, err := NewLoopback("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	l.StopAfterFirst = false // keep serving so the second request can arrive
	defer l.Close()

	first, err := http.Get(l.RedirectURI() + "?code=first")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, first.Body)
	first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first status = %d", first.StatusCode)
	}

	second, err := http.Get(l.RedirectURI() + "?code=second")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, second.Body)
	second.Body.Close()
	if second.StatusCode != http.StatusConflict {
		t.Errorf("second status = %d, want 409", second.StatusCode)
	}

	// The first code is what the caller sees, not the second.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	query, err := l.Wait(ctx)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if query.Get("code") != "first" {
		t.Errorf("code = %q, want the first", query.Get("code"))
	}
}

// TestLoopbackCancelUnblocksWait frees a caller whose user abandoned the browser.
func TestLoopbackCancelUnblocksWait(t *testing.T) {
	l, err := NewLoopback("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	_, err = l.Wait(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// TestLoopbackCloseUnblocksWait covers shutdown while waiting.
func TestLoopbackCloseUnblocksWait(t *testing.T) {
	l, err := NewLoopback("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := l.Wait(context.Background())
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	if err := l.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	select {
	case err := <-done:
		if err == nil {
			t.Error("wait should report that the listener closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not unblock Wait")
	}

	// Closing twice is safe, so a deferred close cannot mask a real one.
	if err := l.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
}

// TestLoopbackFixedPort covers the registered-redirect case.
//
// Providers compare the redirect URI as a string, so a fixed port must be
// honoured exactly rather than replaced with a free one.
func TestLoopbackFixedPort(t *testing.T) {
	first, err := NewLoopback("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	uri := first.RedirectURI()
	first.Close()

	// Reuse the port the first listener had.
	_, port, err := splitHostPort(uri)
	if err != nil {
		t.Fatal(err)
	}

	second, err := NewLoopback("127.0.0.1:" + port)
	if err != nil {
		// The port may be in TIME_WAIT on a busy machine; skip rather than flake.
		t.Skipf("could not rebind %s: %v", port, err)
	}
	defer second.Close()

	if got := second.RedirectURI(); got != uri {
		t.Errorf("redirect URI = %q, want %q", got, uri)
	}
}

// TestLoopbackCustomPath covers a provider that registers a different callback.
func TestLoopbackCustomPath(t *testing.T) {
	l, err := NewLoopbackAt("127.0.0.1:0", "/oauth/cb")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if !strings.HasSuffix(l.RedirectURI(), "/oauth/cb") {
		t.Fatalf("redirect URI = %q", l.RedirectURI())
	}

	go func() {
		resp, err := http.Get(l.RedirectURI() + "?code=z")
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query, err := l.Wait(ctx)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if query.Get("code") != "z" {
		t.Errorf("code = %q", query.Get("code"))
	}
}

// TestLoopbackListenReturnsURI covers the Loopback interface method.
func TestLoopbackListenReturnsURI(t *testing.T) {
	l, err := NewLoopback("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	uri, err := l.Listen(context.Background(), "")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	if uri != l.RedirectURI() {
		t.Errorf("uri = %q, want %q", uri, l.RedirectURI())
	}
}

// TestLoopbackFactorySatisfiesServices proves the factory plugs into Services.
func TestLoopbackFactorySatisfiesServices(t *testing.T) {
	services := Services{Loopback: LoopbackFactory("127.0.0.1:0")}

	lb, err := services.LoopbackOr()()
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	defer lb.Close()

	uri, err := lb.Listen(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(uri, "127.0.0.1") {
		t.Errorf("uri = %q", uri)
	}
}

// TestLoopbackBindFailureIsReported covers a port already in use.
func TestLoopbackBindFailureIsReported(t *testing.T) {
	first, err := NewLoopback("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	_, port, err := splitHostPort(first.RedirectURI())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewLoopback("127.0.0.1:" + port); err == nil {
		t.Error("binding an occupied port should fail")
	}
}

// splitHostPort extracts the port from an http URI.
func splitHostPort(uri string) (string, string, error) {
	trimmed := strings.TrimPrefix(uri, "http://")
	hostPort := strings.TrimSuffix(trimmed, DefaultCallbackPath)
	idx := strings.LastIndex(hostPort, ":")
	if idx < 0 {
		return "", "", errors.New("no port in URI")
	}
	return hostPort[:idx], hostPort[idx+1:], nil
}
