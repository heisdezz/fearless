package platform

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// TestDefaultsDegradeNotPanic is the acceptance criterion's core: with no
// services supplied, every accessor returns something usable.
func TestDefaultsDegradeNotPanic(t *testing.T) {
	var services Services
	ctx := context.Background()

	// Dialogs cancel rather than returning an empty path, so a caller cannot
	// mistake "no window" for "the user chose nothing" in a way it might act on.
	if _, err := services.DialogsOr().OpenFile(ctx, OpenOptions{}); !errors.Is(err, ErrCancelled) {
		t.Errorf("OpenFile err = %v, want ErrCancelled", err)
	}
	if _, err := services.DialogsOr().OpenFiles(ctx, OpenOptions{}); !errors.Is(err, ErrCancelled) {
		t.Errorf("OpenFiles err = %v, want ErrCancelled", err)
	}
	if _, err := services.DialogsOr().SaveFile(ctx, SaveOptions{}); !errors.Is(err, ErrCancelled) {
		t.Errorf("SaveFile err = %v, want ErrCancelled", err)
	}
	if _, err := services.DialogsOr().OpenDirectory(ctx, OpenOptions{}); !errors.Is(err, ErrCancelled) {
		t.Errorf("OpenDirectory err = %v, want ErrCancelled", err)
	}

	// Secrets report the platform as unsupported and as unavailable.
	secrets := services.SecretsOr()
	if secrets.Available(ctx) {
		t.Error("the default secrets store must report unavailable")
	}
	if _, err := secrets.Get(ctx, "k"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Get err = %v", err)
	}
	if err := secrets.Set(ctx, "k", []byte("v")); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Set err = %v", err)
	}
	if err := secrets.Delete(ctx, "k"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Delete err = %v", err)
	}

	// Browser and loopback report unsupported rather than failing silently.
	if err := services.BrowserOr().Open(ctx, "https://example.com"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Browser err = %v", err)
	}
	if _, err := services.LoopbackOr()(); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Loopback err = %v", err)
	}

	// Watchers report unsupported.
	if _, err := services.WatchersOr().Watch(ctx, "/tmp", WatchOptions{}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Watch err = %v", err)
	}

	// Window state keeps geometry in memory, so a window still reopens at its
	// last size within a session.
	state := services.WindowStateOr()
	if _, ok, err := state.Load(ctx); err != nil || ok {
		t.Errorf("initial load = %v, %v", ok, err)
	}
	want := Geometry{X: 10, Y: 20, Width: 1200, Height: 800}
	if err := state.Save(ctx, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, ok, err := state.Load(ctx)
	if err != nil || !ok {
		t.Fatalf("load = %v, %v", ok, err)
	}
	if got != want {
		t.Errorf("geometry = %+v, want %+v", got, want)
	}
}

// TestUnsupportedIsErrorsUnsupported lets callers match MyGo's own sentinel.
//
// MyGo returns errors.Is(err, errors.ErrUnsupported) when there is no credential
// store; aliasing that here means one check works for both.
func TestUnsupportedIsErrorsUnsupported(t *testing.T) {
	if !errors.Is(ErrUnsupported, errors.ErrUnsupported) {
		t.Fatal("ErrUnsupported must satisfy errors.Is(err, errors.ErrUnsupported)")
	}
}

// TestGeometryValidity covers the guard against stored garbage.
func TestGeometryValidity(t *testing.T) {
	cases := []struct {
		name string
		g    Geometry
		want bool
	}{
		{"normal", Geometry{Width: 800, Height: 600}, true},
		{"maximized", Geometry{Width: 800, Height: 600, Maximized: true}, true},
		{"zero size", Geometry{}, false},
		{"negative size", Geometry{Width: -1, Height: 600}, false},
		{"missing height", Geometry{Width: 800}, false},
	}
	for _, tc := range cases {
		if got := tc.g.Valid(); got != tc.want {
			t.Errorf("%s: Valid() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestQueryGet covers the loopback query accessor.
func TestQueryGet(t *testing.T) {
	q := Query{"code": {"abc"}, "state": {"s1", "s2"}, "empty": {}}

	if got := q.Get("code"); got != "abc" {
		t.Errorf("code = %q", got)
	}
	// Multiple values return the first, which is what OAuth providers send.
	if got := q.Get("state"); got != "s1" {
		t.Errorf("state = %q", got)
	}
	if got := q.Get("empty"); got != "" {
		t.Errorf("empty = %q", got)
	}
	if got := q.Get("missing"); got != "" {
		t.Errorf("missing = %q", got)
	}
	// A nil Query must not panic.
	var nilQuery Query
	if got := nilQuery.Get("x"); got != "" {
		t.Errorf("nil query = %q", got)
	}
}

// --- test doubles --------------------------------------------------------

// stubDialogs records the requests it received.
type stubDialogs struct {
	mu       sync.Mutex
	opened   []OpenOptions
	saved    []SaveOptions
	files    []string
	fileErr  error
	savePath string
}

func (s *stubDialogs) OpenFile(_ context.Context, opts OpenOptions) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = append(s.opened, opts)
	if s.fileErr != nil {
		return "", s.fileErr
	}
	if len(s.files) == 0 {
		return "", ErrCancelled
	}
	return s.files[0], nil
}

func (s *stubDialogs) OpenFiles(_ context.Context, opts OpenOptions) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = append(s.opened, opts)
	if s.fileErr != nil {
		return nil, s.fileErr
	}
	return s.files, nil
}

func (s *stubDialogs) SaveFile(_ context.Context, opts SaveOptions) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = append(s.saved, opts)
	if s.fileErr != nil {
		return "", s.fileErr
	}
	return s.savePath, nil
}

func (s *stubDialogs) OpenDirectory(_ context.Context, opts OpenOptions) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = append(s.opened, opts)
	if s.fileErr != nil {
		return "", s.fileErr
	}
	if len(s.files) == 0 {
		return "", ErrCancelled
	}
	return s.files[0], nil
}

// TestInjectedServicesAreUsed proves the seam works: a fake supplied through
// Services is what the caller reaches.
func TestInjectedServicesAreUsed(t *testing.T) {
	dialogs := &stubDialogs{files: []string{"/tmp/one.json", "/tmp/two.json"}, savePath: "/tmp/out.json"}
	services := Services{Dialogs: dialogs}

	path, err := services.DialogsOr().OpenFile(context.Background(), OpenOptions{Title: "Import"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if path != "/tmp/one.json" {
		t.Errorf("path = %q", path)
	}
	// The options reached the implementation intact, which is what a real picker
	// needs in order to apply the title and filters.
	if len(dialogs.opened) != 1 || dialogs.opened[0].Title != "Import" {
		t.Errorf("options = %+v", dialogs.opened)
	}

	if _, err := services.DialogsOr().SaveFile(context.Background(), SaveOptions{SuggestedName: "x.json"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if dialogs.saved[0].SuggestedName != "x.json" {
		t.Errorf("save options = %+v", dialogs.saved[0])
	}
}

// TestDialogCancellationIsDistinct keeps "user dismissed" separate from failure.
func TestDialogCancellationIsDistinct(t *testing.T) {
	dialogs := &stubDialogs{}
	_, err := dialogs.OpenFile(context.Background(), OpenOptions{})
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("err = %v", err)
	}

	// A real failure is not reported as a cancellation.
	failing := &stubDialogs{fileErr: errors.New("permission denied")}
	_, err = failing.OpenFile(context.Background(), OpenOptions{})
	if errors.Is(err, ErrCancelled) {
		t.Error("a failure must not be reported as a cancellation")
	}
}

// TestNoDialogsAlwaysCancels pins the headless dialog behaviour.
func TestNoDialogsAlwaysCancels(t *testing.T) {
	var d NoDialogs
	ctx := context.Background()

	if _, err := d.OpenFile(ctx, OpenOptions{}); !errors.Is(err, ErrCancelled) {
		t.Errorf("OpenFile = %v", err)
	}
	if _, err := d.OpenFiles(ctx, OpenOptions{}); !errors.Is(err, ErrCancelled) {
		t.Errorf("OpenFiles = %v", err)
	}
	if _, err := d.SaveFile(ctx, SaveOptions{}); !errors.Is(err, ErrCancelled) {
		t.Errorf("SaveFile = %v", err)
	}
	if _, err := d.OpenDirectory(ctx, OpenOptions{}); !errors.Is(err, ErrCancelled) {
		t.Errorf("OpenDirectory = %v", err)
	}
}
