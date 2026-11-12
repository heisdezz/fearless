package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// sessionState is the small file that survives a restart: which collections
// were open. Settings live in preferences.json; this is separate because it
// changes on every open/close rather than on a settings save, and losing it
// should only cost the window layout, never configuration.
type sessionState struct {
	// Collections are absolute paths, in the order they were opened.
	Collections []string `json:"collections,omitempty"`
	// Active is the path of the collection last selected. A path rather than a
	// UID because UIDs are regenerated when a collection is loaded, so they
	// cannot survive a restart.
	Active string `json:"active,omitempty"`
}

// defaultSessionPath is session.json next to preferences.json.
func defaultSessionPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("session: no config directory: %w", err)
		}
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "fearless", "session.json"), nil
}

// loadSession reads the session file. A missing file is not an error: it means
// nothing was open last time.
func loadSession(path string) sessionState {
	if path == "" {
		p, err := defaultSessionPath()
		if err != nil {
			return sessionState{}
		}
		path = p
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return sessionState{}
	}
	var s sessionState
	if err := json.Unmarshal(data, &s); err != nil {
		// A corrupt session file costs the window layout, nothing more.
		return sessionState{}
	}
	return s
}

// saveSession writes the session file atomically, so a crash mid-write cannot
// leave a truncated file that loses every collection on the next launch.
func saveSession(path string, s sessionState) error {
	if path == "" {
		p, err := defaultSessionPath()
		if err != nil {
			return err
		}
		path = p
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("session: create config directory: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("session: encode: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("session: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("session: replace %s: %w", path, err)
	}
	return nil
}
