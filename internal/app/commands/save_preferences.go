package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"fearless/internal/core/model"
	"fearless/internal/core/persistence"
)

// SavePreferences persists updated application settings to disk and updates the workspace state.
// Writes are performed atomically using persistence.WriteAtomic.
func (s *PreferencesService) SavePreferences(ctx context.Context, cmd SavePreferencesCmd) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	targetPath := cmd.ConfigPath
	if targetPath == "" {
		targetPath = s.ConfigPath()
	}
	if targetPath == "" {
		return fmt.Errorf("%w: config path is not set", ErrInvalidPreferences)
	}

	// Sanitize and validate settings defaults
	settings := cmd.Settings
	if settings.SchemaVersion <= 0 {
		settings.SchemaVersion = model.CurrentSettingsSchemaVersion
	}
	if settings.Appearance.Zoom <= 0 {
		settings.Appearance.Zoom = 100
	}
	if settings.Appearance.Theme == "" {
		settings.Appearance.Theme = "system"
	}
	if settings.Request.MaxRedirects < 0 {
		settings.Request.MaxRedirects = model.DefaultMaxRedirects
	}
	if settings.Request.MaxResponseBytes <= 0 {
		settings.Request.MaxResponseBytes = 512 << 20
	}
	if settings.Request.InlineBodyBytes <= 0 {
		settings.Request.InlineBodyBytes = 32 << 20
	}
	if settings.Editor.FontSize <= 0 {
		settings.Editor.FontSize = 13
	}
	if settings.Editor.TabSize <= 0 {
		settings.Editor.TabSize = 2
	}
	if settings.General.Language == "" {
		settings.General.Language = "en"
	}
	if settings.Privacy.HistoryRetainDays < 0 {
		settings.Privacy.HistoryRetainDays = 30
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize preferences: %w", err)
	}

	// Atomic write ensures readers never observe partial files even across crashes
	if err := persistence.WriteAtomic(targetPath, data, 0o644); err != nil {
		return fmt.Errorf("write preferences to %s: %w", targetPath, err)
	}

	// Update workspace state
	if s.store != nil {
		s.store.Apply(func(state *model.WorkspaceState) {
			state.Settings = settings
		})
	}

	return nil
}
