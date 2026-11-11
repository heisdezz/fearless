// Package platform isolates every operating-system concern behind an interface.
//
// The rule this package enforces is simple and load-bearing: nothing above it
// may import the GUI toolkit. The GUI layer supplies the real implementations;
// tests and the headless CLI runner use the fakes in [fake], which have no
// dependency on MyGo at all.
//
// Each sub-concern lives in its own interface so a caller depends only on what
// it uses. The five are:
//
//   - Dialogs: file pickers and save prompts, for import/export and body-from-file
//   - Secrets: the OS credential store, for tokens and client secrets
//   - Browser: opening a URL, for the OAuth2 authorization-code flow
//   - Watcher: file-system change notifications, for external edits to a collection
//   - WindowState: remembering window geometry between launches
//
// A GUI implementation of each lives behind a build tag in this package's
// subpackages; the interfaces here are toolkit-free by construction, so the
// dependency check (`scripts/check-deps.sh`) covers them by ordinary means.
//
// Source: mygo v0.8.4 — Secrets, WatchFiles (see docs/research/dependencies.md)
package platform

import (
	"context"
	"errors"
	"time"
)

// Common errors.
var (
	// ErrUnsupported reports an operation the platform cannot perform. MyGo
	// itself returns errors satisfying errors.Is(err, errors.ErrUnsupported) when
	// there is no credential store, so that sentinel is reused rather than
	// inventing a parallel one.
	ErrUnsupported = errors.ErrUnsupported

	// ErrCancelled reports a dialog the user dismissed.
	ErrCancelled = errors.New("platform: cancelled")

	// ErrNotFound reports a secret that is not stored.
	ErrNotFound = errors.New("platform: not found")
)

// --- Dialogs -------------------------------------------------------------

// FileFilter is one entry in a picker's filter list.
type FileFilter struct {
	// Name is the label, e.g. "Bruno collection".
	Name string
	// Extensions are the allowed suffixes without a leading dot, e.g. ["bru"].
	// An empty list means "any file".
	Extensions []string
}

// OpenOptions configure a file-open prompt.
type OpenOptions struct {
	Title string
	// Directory is where the picker starts; empty uses the last used place.
	Directory string
	Filters   []FileFilter
	// Multiple allows more than one selection.
	Multiple bool
}

// SaveOptions configure a save prompt.
type SaveOptions struct {
	Title     string
	Directory string
	// SuggestedName is pre-filled in the name field.
	SuggestedName string
	Filters       []FileFilter
}

// Dialogs prompts the user for a file or folder.
//
// Every method returns ErrCancelled when the user dismisses the prompt, so a
// caller can distinguish "nothing chosen" from a failure.
type Dialogs interface {
	// OpenFile returns the chosen path, or ErrCancelled.
	OpenFile(ctx context.Context, opts OpenOptions) (string, error)
	// OpenFiles returns every chosen path, or ErrCancelled.
	OpenFiles(ctx context.Context, opts OpenOptions) ([]string, error)
	// SaveFile returns the chosen target path, or ErrCancelled.
	SaveFile(ctx context.Context, opts SaveOptions) (string, error)
	// OpenDirectory returns the chosen folder, or ErrCancelled.
	OpenDirectory(ctx context.Context, opts OpenOptions) (string, error)
}

// --- Secrets -------------------------------------------------------------

// Secrets is the OS credential store.
//
// Values are small: a token, a client secret, an environment secret. A caller
// that needs to store more should encrypt a file with a key kept here, which is
// the pattern MyGo documents for its own 2560-byte limit.
//
// Implementations that cannot reach a store return ErrUnsupported, which is the
// normal state on a Linux server or a minimal desktop without a Secret Service.
// Callers must handle it: refusing to start because a keyring is absent would
// make the CLI runner unusable in CI.
type Secrets interface {
	// Get returns the secret, or ErrNotFound.
	Get(ctx context.Context, key string) ([]byte, error)
	// Set stores the secret, replacing any previous value.
	Set(ctx context.Context, key string, value []byte) error
	// Delete removes the secret. Deleting an absent key is not an error.
	Delete(ctx context.Context, key string) error
	// Available reports whether the store can be used, so a caller can warn once
	// rather than on every operation.
	Available(ctx context.Context) bool
}

// --- Browser -------------------------------------------------------------

// Browser opens URLs outside the application.
type Browser interface {
	// Open asks the system to open the URL and returns once the request has been
	// handed over; it does not wait for the browser to finish.
	Open(ctx context.Context, url string) error
}

// Loopback is a one-shot local HTTP listener, which is how the OAuth2
// authorization-code flow receives its callback.
//
// It is separate from Browser because the flow needs both: open the
// authorization URL, then receive the redirect on a port the server will accept.
// Redirect URIs must be registered with the provider, so the port is usually
// fixed rather than random.
type Loopback interface {
	// Listen binds to addr (host:port) and returns the full redirect URI to send
	// to the provider.
	Listen(ctx context.Context, addr string) (redirectURI string, err error)
	// Wait blocks until the provider redirects, returning the query parameters.
	Wait(ctx context.Context) (Query, error)
	// Close stops the listener. It is safe to call twice.
	Close() error
}

// Query is a URL query string as received by the loopback listener.
type Query map[string][]string

// Get returns the first value for a key.
func (q Query) Get(key string) string {
	if values, ok := q[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

// --- Watcher -------------------------------------------------------------

// Op is a file-system change kind.
type Op string

const (
	OpCreate Op = "create"
	OpWrite  Op = "write"
	OpRemove Op = "remove"
	OpRename Op = "rename"
	// OpHint says only that something under the path changed, which is what a
	// platform reports when it cannot be more specific.
	OpHint Op = "hint"
)

// Event is one change.
type Event struct {
	Op Op
	// Path is relative to the watched root, slash-separated. "." is the root.
	Path string
	// OldPath is set for a rename.
	OldPath string
	IsDir   bool
}

// WatchOptions configure a watch.
type WatchOptions struct {
	// Recursive watches subdirectories, which a collection needs.
	Recursive bool
	// Debounce coalesces bursts. Zero uses the implementation's default.
	Debounce time.Duration
}

// Watcher reports changes under one path.
//
// Events are hints: a file saved twice may be reported once, and a caller should
// re-read rather than assume. That contract is the platform's, not this
// package's — MyGo documents exactly this behaviour.
type Watcher interface {
	// Next blocks until events are available. It returns io.EOF when the watch
	// has ended because the root was removed or renamed.
	Next(ctx context.Context) ([]Event, error)
	// Close stops the watch.
	Close() error
}

// WatcherFactory opens watches.
type WatcherFactory interface {
	// Watch starts watching path.
	Watch(ctx context.Context, path string, opts WatchOptions) (Watcher, error)
}

// --- WindowState ---------------------------------------------------------

// WindowState remembers window geometry between launches.
//
// Saving on every resize would write constantly, so the implementation is
// expected to be cheap and idempotent; a caller may call Save as often as it
// likes.
type WindowState interface {
	// Load returns the stored geometry, or (Geometry{}, false) when none is
	// stored.
	Load(ctx context.Context) (Geometry, bool, error)
	// Save stores the geometry.
	Save(ctx context.Context, g Geometry) error
}

// Geometry is a window's position and size in screen points.
type Geometry struct {
	X      int
	Y      int
	Width  int
	Height int
	// Maximized records the maximized state separately, because a maximized
	// window's Width and Height are the restored size.
	Maximized bool
}

// Valid reports whether the geometry is plausible.
//
// A stored value can be corrupt, or come from a monitor that is no longer
// attached, so a caller should check before applying it. A non-positive size is
// the common failure and is what this rejects.
func (g Geometry) Valid() bool {
	return g.Width > 0 && g.Height > 0
}

// --- Aggregate -----------------------------------------------------------

// Services bundles the platform implementations.
//
// Passing one value keeps constructors short. A field may be nil when the
// caller does not need that concern; every accessor returns a no-op
// implementation rather than nil, so a missing service degrades instead of
// panicking.
type Services struct {
	Dialogs     Dialogs
	Secrets     Secrets
	Browser     Browser
	Loopback    func() (Loopback, error)
	Watchers    WatcherFactory
	WindowState WindowState
}

// DialogsOr returns the dialogs, or cancellable no-ops.
func (s Services) DialogsOr() Dialogs {
	if s.Dialogs != nil {
		return s.Dialogs
	}
	return NoDialogs{}
}

// SecretsOr returns the secrets store, or one that is never available.
func (s Services) SecretsOr() Secrets {
	if s.Secrets != nil {
		return s.Secrets
	}
	return UnavailableSecrets{}
}

// BrowserOr returns the browser, or one that reports ErrUnsupported.
func (s Services) BrowserOr() Browser {
	if s.Browser != nil {
		return s.Browser
	}
	return UnsupportedBrowser{}
}

// LoopbackOr returns a loopback factory, or one that reports ErrUnsupported.
func (s Services) LoopbackOr() func() (Loopback, error) {
	if s.Loopback != nil {
		return s.Loopback
	}
	return func() (Loopback, error) { return nil, ErrUnsupported }
}

// WatchersOr returns the watcher factory, or one that reports ErrUnsupported.
func (s Services) WatchersOr() WatcherFactory {
	if s.Watchers != nil {
		return s.Watchers
	}
	return UnsupportedWatchers{}
}

// WindowStateOr returns the window state store, or a memory-only one.
func (s Services) WindowStateOr() WindowState {
	if s.WindowState != nil {
		return s.WindowState
	}
	return &MemoryWindowState{}
}

// --- Degrading defaults --------------------------------------------------

// NoDialogs cancels every prompt.
//
// This is the right default for the CLI runner: there is no window to attach a
// picker to, and a caller that reaches one has made a mistake that should
// surface as a cancellation rather than a hang.
type NoDialogs struct{}

// OpenFile implements Dialogs.
func (NoDialogs) OpenFile(context.Context, OpenOptions) (string, error) {
	return "", ErrCancelled
}

// OpenFiles implements Dialogs.
func (NoDialogs) OpenFiles(context.Context, OpenOptions) ([]string, error) {
	return nil, ErrCancelled
}

// SaveFile implements Dialogs.
func (NoDialogs) SaveFile(context.Context, SaveOptions) (string, error) {
	return "", ErrCancelled
}

// OpenDirectory implements Dialogs.
func (NoDialogs) OpenDirectory(context.Context, OpenOptions) (string, error) {
	return "", ErrCancelled
}

// UnavailableSecrets reports an absent credential store.
type UnavailableSecrets struct{}

// Get implements Secrets.
func (UnavailableSecrets) Get(context.Context, string) ([]byte, error) {
	return nil, ErrUnsupported
}

// Set implements Secrets.
func (UnavailableSecrets) Set(context.Context, string, []byte) error { return ErrUnsupported }

// Delete implements Secrets.
func (UnavailableSecrets) Delete(context.Context, string) error { return ErrUnsupported }

// Available implements Secrets.
func (UnavailableSecrets) Available(context.Context) bool { return false }

// UnsupportedBrowser reports that handing a URL to the system is unavailable.
type UnsupportedBrowser struct{}

// Open implements Browser.
func (UnsupportedBrowser) Open(context.Context, string) error { return ErrUnsupported }

// UnsupportedWatchers reports that file watching is unavailable.
type UnsupportedWatchers struct{}

// Watch implements WatcherFactory.
func (UnsupportedWatchers) Watch(context.Context, string, WatchOptions) (Watcher, error) {
	return nil, ErrUnsupported
}

// MemoryWindowState keeps geometry for the session only.
//
// It is the correct default when nothing persists it: the window still opens at
// the size the user last chose within a session, rather than jumping to the
// default on every restart of the view.
type MemoryWindowState struct {
	geometry Geometry
	set      bool
}

// Load implements WindowState.
func (m *MemoryWindowState) Load(context.Context) (Geometry, bool, error) {
	return m.geometry, m.set, nil
}

// Save implements WindowState.
func (m *MemoryWindowState) Save(_ context.Context, g Geometry) error {
	m.geometry = g
	m.set = true
	return nil
}
