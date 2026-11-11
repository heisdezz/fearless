package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"fearless/internal/app/bus"
	"fearless/internal/core/engine/auth"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
	"fearless/internal/core/persistence"
)

var (
	// ErrInvalidPreferences is returned when preferences data fails validation.
	ErrInvalidPreferences = errors.New("invalid preferences")
)

// Consumer interfaces declared per interface-design.md §2 for UI presenters.

// PreferencesSaver persists application preferences and updates workspace state.
type PreferencesSaver interface {
	SavePreferences(ctx context.Context, cmd SavePreferencesCmd) error
}

// PreferencesLoader loads application preferences from disk into workspace state.
type PreferencesLoader interface {
	LoadPreferences(ctx context.Context, cmd LoadPreferencesCmd) (*model.Settings, error)
}

// OAuthCacheCleaner purges cached OAuth2 tokens from memory and disk.
type OAuthCacheCleaner interface {
	ClearOAuth2Cache(ctx context.Context, cmd ClearOAuthCacheCmd) error
}

// Command payloads.

// SavePreferencesCmd defines preferences to be saved.
type SavePreferencesCmd struct {
	Settings   model.Settings
	ConfigPath string // Optional explicit path; defaults to service configPath if empty
}

// LoadPreferencesCmd defines preferences to be loaded.
type LoadPreferencesCmd struct {
	ConfigPath string // Optional explicit path; defaults to service configPath if empty
}

// ClearOAuthCacheCmd specifies tokens to clear.
// If CredentialsID is empty, all cached tokens are cleared.
type ClearOAuthCacheCmd struct {
	CredentialsID string
}

// PreferencesService coordinates application settings persistence, schema migration,
// state updates, and OAuth2 token cache management.
type PreferencesService struct {
	mu         sync.RWMutex
	store      *bus.Store
	bus        *bus.Bus
	configPath string
	oauthCache OAuthTokenCache
}

// NewPreferencesService creates a new PreferencesService.
func NewPreferencesService(store *bus.Store, b *bus.Bus, configPath string, cache OAuthTokenCache) *PreferencesService {
	if cache == nil {
		cache = NewMemoryOAuthTokenCache()
	}
	if configPath == "" {
		if p, err := DefaultPreferencesPath(); err == nil {
			configPath = p
		}
	}
	return &PreferencesService{
		store:      store,
		bus:        b,
		configPath: configPath,
		oauthCache: cache,
	}
}

// ConfigPath returns the current target configuration file path.
func (s *PreferencesService) ConfigPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configPath
}

// SetConfigPath updates the target configuration file path.
func (s *PreferencesService) SetConfigPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configPath = path
}

// OAuthCache returns the underlying OAuthTokenCache.
func (s *PreferencesService) OAuthCache() OAuthTokenCache {
	return s.oauthCache
}

// TokenStore returns a token lookup function satisfying auth.Deps.TokenStore.
func (s *PreferencesService) TokenStore() func(credentialsID string) (auth.Token, bool) {
	if s.oauthCache == nil {
		return func(string) (auth.Token, bool) { return auth.Token{}, false }
	}
	return s.oauthCache.Get
}

// DefaultPreferencesPath determines the platform-appropriate preferences file location.
func DefaultPreferencesPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("unable to determine user config directory: %w", err)
		}
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "fearless", "preferences.json"), nil
}

// MigrateSettings upgrades raw JSON settings from older schemas to CurrentSettingsSchemaVersion.
// It returns the migrated settings and whether an upgrade occurred.
func MigrateSettings(data []byte) (*model.Settings, bool, error) {
	// First unmarshal into a generic map to detect raw version and fields.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false, fmt.Errorf("unmarshal raw settings: %w", err)
	}

	defaults := model.DefaultSettings()

	// Parse into concrete model.Settings
	var parsed model.Settings
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, false, fmt.Errorf("unmarshal structured settings: %w", err)
	}

	migrated := false
	ver := parsed.SchemaVersion

	// Version 0 or missing: migrate legacy structures
	if ver < model.CurrentSettingsSchemaVersion {
		migrated = true
		parsed.SchemaVersion = model.CurrentSettingsSchemaVersion

		// Legacy proxy format migration: { proxy: { enabled: true/false } }
		if proxyMap, ok := raw["proxy"].(map[string]any); ok {
			if enabled, hasEnabled := proxyMap["enabled"].(bool); hasEnabled && parsed.Proxy.Mode == "" {
				if enabled {
					parsed.Proxy.Mode = model.ProxyManual
				} else {
					parsed.Proxy.Mode = model.ProxyOff
				}
			}
		}

		// Ensure sensible defaults for unpopulated fields in legacy files
		if parsed.Request.MaxRedirects == 0 {
			parsed.Request.MaxRedirects = defaults.Request.MaxRedirects
		}
		if parsed.Request.MaxResponseBytes == 0 {
			parsed.Request.MaxResponseBytes = defaults.Request.MaxResponseBytes
		}
		if parsed.Request.InlineBodyBytes == 0 {
			parsed.Request.InlineBodyBytes = defaults.Request.InlineBodyBytes
		}
		if parsed.Appearance.Zoom <= 0 {
			parsed.Appearance.Zoom = defaults.Appearance.Zoom
		}
		if parsed.Appearance.Theme == "" {
			parsed.Appearance.Theme = defaults.Appearance.Theme
		}
		if parsed.Editor.FontSize <= 0 {
			parsed.Editor.FontSize = defaults.Editor.FontSize
		}
		if parsed.Editor.TabSize <= 0 {
			parsed.Editor.TabSize = defaults.Editor.TabSize
		}
		if parsed.General.Language == "" {
			parsed.General.Language = defaults.General.Language
		}
		if parsed.Privacy.HistoryRetainDays == 0 {
			parsed.Privacy.HistoryRetainDays = defaults.Privacy.HistoryRetainDays
		}
	}

	return &parsed, migrated, nil
}

// LoadPreferences loads application settings from disk into workspace state.
// If the settings file does not exist, it initializes with DefaultSettings.
// If corrupted, the corrupted file is quarantined and DefaultSettings are restored.
func (s *PreferencesService) LoadPreferences(ctx context.Context, cmd LoadPreferencesCmd) (*model.Settings, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	targetPath := cmd.ConfigPath
	if targetPath == "" {
		targetPath = s.ConfigPath()
	}
	if targetPath == "" {
		return nil, fmt.Errorf("%w: config path is not set", ErrInvalidPreferences)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			defaults := model.DefaultSettings()
			if s.store != nil {
				s.store.Apply(func(state *model.WorkspaceState) {
					state.Settings = defaults
				})
			}
			return &defaults, nil
		}
		return nil, fmt.Errorf("read preferences %s: %w", targetPath, err)
	}

	settings, migrated, err := MigrateSettings(data)
	if err != nil {
		// File is corrupted. Quarantine it to preserve for diagnosis and restore defaults.
		quarantined, qErr := persistence.QuarantineCorrupt(targetPath)
		errMsg := fmt.Sprintf("corrupt preferences file at %s quarantined", targetPath)
		if qErr == nil {
			errMsg = fmt.Sprintf("corrupt preferences file quarantined to %s", quarantined)
		}
		if s.bus != nil {
			s.bus.Publish(events.ErrorOccurred{
				Scope:   events.ScopeSettings,
				Class:   events.ErrorValidation,
				Msg:     errMsg,
				Details: err.Error(),
				Err:     err,
			})
		}

		defaults := model.DefaultSettings()
		// Save clean defaults to disk
		_ = s.SavePreferences(ctx, SavePreferencesCmd{
			Settings:   defaults,
			ConfigPath: targetPath,
		})
		return &defaults, nil
	}

	// If migration took place, write back once atomically
	if migrated {
		_ = s.SavePreferences(ctx, SavePreferencesCmd{
			Settings:   *settings,
			ConfigPath: targetPath,
		})
	} else if s.store != nil {
		s.store.Apply(func(state *model.WorkspaceState) {
			state.Settings = *settings
		})
	}

	return settings, nil
}

// SavePreferencesCmdHandler adapts PreferencesService for Execute-based dispatch.
type SavePreferencesCmdHandler struct{ s *PreferencesService }

// Execute runs SavePreferences.
func (h *SavePreferencesCmdHandler) Execute(ctx context.Context, cmd SavePreferencesCmd) error {
	return h.s.SavePreferences(ctx, cmd)
}

// LoadPreferencesCmdHandler adapts PreferencesService for Execute-based dispatch.
type LoadPreferencesCmdHandler struct{ s *PreferencesService }

// Execute runs LoadPreferences.
func (h *LoadPreferencesCmdHandler) Execute(ctx context.Context, cmd LoadPreferencesCmd) (*model.Settings, error) {
	return h.s.LoadPreferences(ctx, cmd)
}

// ClearOAuthCacheCmdHandler adapts PreferencesService for Execute-based dispatch.
type ClearOAuthCacheCmdHandler struct{ s *PreferencesService }

// Execute runs ClearOAuth2Cache.
func (h *ClearOAuthCacheCmdHandler) Execute(ctx context.Context, cmd ClearOAuthCacheCmd) error {
	return h.s.ClearOAuth2Cache(ctx, cmd)
}

// Saver returns an Execute-compatible handler for SavePreferencesCmd.
func (s *PreferencesService) Saver() *SavePreferencesCmdHandler {
	return &SavePreferencesCmdHandler{s: s}
}

// Loader returns an Execute-compatible handler for LoadPreferencesCmd.
func (s *PreferencesService) Loader() *LoadPreferencesCmdHandler {
	return &LoadPreferencesCmdHandler{s: s}
}

// Cleaner returns an Execute-compatible handler for ClearOAuthCacheCmd.
func (s *PreferencesService) Cleaner() *ClearOAuthCacheCmdHandler {
	return &ClearOAuthCacheCmdHandler{s: s}
}
