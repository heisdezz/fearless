package platform

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// LoopbackListener is the standard OAuth2 redirect receiver.
//
// The authorization-code flow needs an HTTPS redirect whose result the
// application can read. A desktop app cannot claim a public HTTPS endpoint, so
// providers were given the option of a loopback address on a fixed port; the
// browser redirects to `http://127.0.0.1:<port>/callback` and this listener picks
// the query up.
//
// Two properties matter and are why this is implemented rather than stubbed:
//
//   - **The port must be fixed** where the provider requires it. A registered
//     redirect URI that changes every run would have to be re-registered.
//   - **Only one request is accepted.** The listener answers the browser with a
//     short page and then refuses further requests, so a second tab cannot
//     replace the authorization code being awaited.
type LoopbackListener struct {
	listener net.Listener
	server   *http.Server

	mu    sync.Mutex
	query Query
	err   error
	ready chan struct{}
	done  chan struct{}

	// Path is the callback path. It defaults to "/callback".
	Path string
	// StopAfterFirst ends the listener once a callback arrives, which is what a
	// one-shot flow wants.
	StopAfterFirst bool
}

// DefaultCallbackPath is the path the listener answers on.
const DefaultCallbackPath = "/callback"

// NewLoopback binds addr and starts serving.
//
// addr is a host:port pair. Port "0" picks a free port and is used in tests; a
// real flow normally passes the port registered with the provider.
//
// The callback path is fixed at DefaultCallbackPath by construction
// (NewLoopbackAt for a different one): the route is registered when the server
// starts, so changing Path afterwards would leave the handler unreachable.
func NewLoopback(addr string) (*LoopbackListener, error) {
	return NewLoopbackAt(addr, DefaultCallbackPath)
}

// NewLoopbackAt is NewLoopback with a chosen callback path, for a provider that
// registers a different one.
func NewLoopbackAt(addr, path string) (*LoopbackListener, error) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	if path == "" {
		path = DefaultCallbackPath
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("platform: listen %s: %w", addr, err)
	}

	l := &LoopbackListener{
		listener:       listener,
		ready:          make(chan struct{}),
		done:           make(chan struct{}),
		Path:           path,
		StopAfterFirst: true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(l.Path, l.handle)
	l.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		err := l.server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.finish(nil, fmt.Errorf("platform: loopback serve: %w", err))
		}
	}()
	return l, nil
}

// RedirectURI returns the URI to register with the provider.
//
// The loopback address is used literally: `localhost` can resolve to IPv6 first
// and fail to connect on a machine where the listener is IPv4, and providers
// compare the URI as a string, so it must match what was registered.
func (l *LoopbackListener) RedirectURI() string {
	host := l.listener.Addr().String()
	_, port, err := net.SplitHostPort(host)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%s%s", port, l.Path)
}

// Listen implements Loopback.
func (l *LoopbackListener) Listen(context.Context, string) (string, error) {
	return l.RedirectURI(), nil
}

// Wait implements Loopback, returning the callback's query parameters.
//
// It returns when a callback arrives or the context ends, whichever is first.
// Ending on the context is what lets a user cancel an authorization they
// abandoned in the browser: without it, the flow would wait forever.
func (l *LoopbackListener) Wait(ctx context.Context) (Query, error) {
	select {
	case <-l.ready:
		l.mu.Lock()
		defer l.mu.Unlock()
		return l.query, l.err
	case <-l.done:
		// Serve ended without a callback.
		l.mu.Lock()
		defer l.mu.Unlock()
		if l.err != nil {
			return nil, l.err
		}
		return nil, errors.New("platform: callback listener closed")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close implements Loopback.
func (l *LoopbackListener) Close() error {
	l.mu.Lock()
	select {
	case <-l.done:
		l.mu.Unlock()
		return nil
	default:
	}
	l.mu.Unlock()

	err := l.server.Close()
	l.finish(nil, nil)
	return err
}

// handle serves one callback.
func (l *LoopbackListener) handle(w http.ResponseWriter, r *http.Request) {
	query := Query(r.URL.Query())

	l.mu.Lock()
	already := l.query != nil
	if !already {
		l.query = query
	}
	stopAfter := l.StopAfterFirst
	l.mu.Unlock()

	// A second request finds the code already taken. Answering it with success
	// would tell a user their second attempt worked when it did not.
	if already {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("This authorization request has already been completed.\n"))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Authorization received. You can close this window.\n"))

	close(l.ready)

	if stopAfter {
		// Shut down after the response flushes, so the browser sees the page.
		go func() {
			l.server.Close()
			l.finish(nil, nil)
		}()
	}
}

// finish records the terminal state of the listener.
func (l *LoopbackListener) finish(q Query, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if q != nil && l.query == nil {
		l.query = q
	}
	if err != nil && l.err == nil {
		l.err = err
	}
	select {
	case <-l.done:
	default:
		close(l.done)
	}
}

// LoopbackFactory returns a function that starts a listener on a fixed address.
//
// It is the shape Services.Loopback expects.
func LoopbackFactory(addr string) func() (Loopback, error) {
	return func() (Loopback, error) {
		return NewLoopback(addr)
	}
}
