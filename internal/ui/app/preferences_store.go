package app

import (
	"context"

	"fearless/internal/app/commands"
	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

// preferencesStore adapts the app's PreferencesService to the presenter's
// loader and saver ports.
//
// It exists so UI-APP owns the wiring and the presenter keeps importing nothing
// from commands (its own layering test forbids it). A nil service is valid: the
// presenter then reads the defaults and Save/Load are no-ops, which is the state
// a bare App has in tests.
type preferencesStore struct {
	svc *commands.PreferencesService
}

func (s preferencesStore) LoadPreferences(ctx context.Context) (*model.Settings, error) {
	if s.svc == nil {
		return nil, nil
	}
	return s.svc.LoadPreferences(ctx, commands.LoadPreferencesCmd{})
}

func (s preferencesStore) SavePreferences(ctx context.Context, settings model.Settings) error {
	if s.svc == nil {
		return nil
	}
	return s.svc.SavePreferences(ctx, commands.SavePreferencesCmd{Settings: settings})
}

var (
	_ presenters.PreferencesLoaderPort = preferencesStore{}
	_ presenters.PreferencesSaverPort  = preferencesStore{}
)
