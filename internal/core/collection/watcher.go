package collection

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fearless/internal/core/model"

	"github.com/fsnotify/fsnotify"
)

var (
	suppressMu  sync.RWMutex
	suppressMap = make(map[string]time.Time)
)

// SuppressPath suppresses reload notifications for self-writes to the specified file path
// within a default cooldown window of 500 milliseconds.
func SuppressPath(path string) {
	SuppressPathFor(path, 500*time.Millisecond)
}

// SuppressPathFor suppresses reload notifications for path for the specified duration.
func SuppressPathFor(path string, duration time.Duration) {
	cleaned := filepath.Clean(path)
	suppressMu.Lock()
	defer suppressMu.Unlock()
	expiry := time.Now().Add(duration)
	suppressMap[cleaned] = expiry
	if realPath, err := filepath.EvalSymlinks(cleaned); err == nil {
		suppressMap[realPath] = expiry
	}
}

// IsSuppressed reports whether a filesystem event for path should be suppressed because
// it originated from Fearless itself (self-write).
func IsSuppressed(path string) bool {
	cleaned := filepath.Clean(path)
	suppressMu.Lock()
	defer suppressMu.Unlock()

	check := func(p string) bool {
		expires, ok := suppressMap[p]
		if !ok {
			return false
		}
		if time.Now().Before(expires) {
			return true
		}
		delete(suppressMap, p)
		return false
	}

	if check(cleaned) {
		return true
	}
	if realPath, err := filepath.EvalSymlinks(cleaned); err == nil {
		if check(realPath) {
			return true
		}
	}
	return false
}

// WatcherOptions configures filesystem watching and debouncing.
type WatcherOptions struct {
	DebounceDelay time.Duration
	OnChange      func(col *model.Collection, err error)
}

// Watcher monitors a collection directory tree for external modifications.
type Watcher struct {
	collection *model.Collection
	fsWatcher  *fsnotify.Watcher
	options    WatcherOptions
	closeChan  chan struct{}
	closeOnce  sync.Once
	mu         sync.Mutex
}

// NewWatcher creates and initializes a file watcher for the collection directory.
func NewWatcher(col *model.Collection, opts WatcherOptions) (*Watcher, error) {
	if col == nil || col.Path == "" {
		return nil, fmt.Errorf("collection: invalid collection for watcher")
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("collection: create fsnotify watcher: %w", err)
	}

	if opts.DebounceDelay <= 0 {
		opts.DebounceDelay = 50 * time.Millisecond
	}

	w := &Watcher{
		collection: col,
		fsWatcher:  fsw,
		options:    opts,
		closeChan:  make(chan struct{}),
	}

	// Recursively watch collection directories
	if err := w.watchRecursive(col.Path); err != nil {
		_ = fsw.Close()
		return nil, err
	}

	go w.loop()

	return w, nil
}

// watchRecursive registers the directory and all existing non-ignored subdirectories.
func (w *Watcher) watchRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if name == ".git" || name == "node_modules" || strings.Contains(name, ".tmp-") {
			return filepath.SkipDir
		}
		return w.fsWatcher.Add(path)
	})
}

// loop runs the background watcher event handling and debouncing loop.
func (w *Watcher) loop() {
	var (
		debounceTimer *time.Timer
		timerChan     <-chan time.Time
	)

	for {
		select {
		case <-w.closeChan:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Skip temporary and editor backup files
			base := filepath.Base(event.Name)
			if strings.Contains(base, ".tmp-") || strings.HasPrefix(base, ".tmp") || strings.HasSuffix(base, "~") {
				continue
			}

			// Check self-write suppression
			if IsSuppressed(event.Name) {
				continue
			}

			// If a new directory was created, add it to watcher
			if event.Has(fsnotify.Create) {
				_ = w.watchRecursive(event.Name)
			}

			// Reset debounce timer
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.NewTimer(w.options.DebounceDelay)
			timerChan = debounceTimer.C

		case <-timerChan:
			// Debounce expired: perform reload and reconcile
			reloaded, err := w.Reload()
			if w.options.OnChange != nil {
				w.options.OnChange(reloaded, err)
			}

		case <-w.fsWatcher.Errors:
			// Ignore transient filesystem watch errors
		}
	}
}

// Reload loads the latest collection state from disk and reconciles it with active drafts.
func (w *Watcher) Reload() (*model.Collection, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	fresh, err := LoadCollection(w.collection.Path)
	if err != nil {
		return w.collection, err
	}

	reconciled := Reconcile(w.collection, fresh)
	w.collection = reconciled
	return reconciled, nil
}

// Current returns the current state of the watched collection.
func (w *Watcher) Current() *model.Collection {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.collection
}

// Close terminates file watching.
func (w *Watcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.closeChan)
		err = w.fsWatcher.Close()
	})
	return err
}
