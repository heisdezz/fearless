package presenters

import (
	"context"
	"errors"
	"testing"

	"fearless/internal/core/model"
)

type fakePreferencesSaver struct {
	savedSettings model.Settings
	err           error
}

func (f *fakePreferencesSaver) SavePreferences(ctx context.Context, settings model.Settings) error {
	if f.err != nil {
		return f.err
	}
	f.savedSettings = settings
	return nil
}

type fakePreferencesLoader struct {
	loadedSettings *model.Settings
	err            error
}

func (f *fakePreferencesLoader) LoadPreferences(ctx context.Context) (*model.Settings, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.loadedSettings, nil
}

type fakeOAuthCleaner struct {
	clearedID string
	err       error
}

func (f *fakeOAuthCleaner) ClearOAuth2Cache(ctx context.Context, credentialsID string) error {
	if f.err != nil {
		return f.err
	}
	f.clearedID = credentialsID
	return nil
}

func TestPreferencesPresenter_BasicsAndTabs(t *testing.T) {
	initial := model.DefaultSettings()
	saver := &fakePreferencesSaver{}
	loader := &fakePreferencesLoader{}
	cleaner := &fakeOAuthCleaner{}

	p := NewPreferencesPresenter(initial, saver, loader, cleaner)

	changed := 0
	p.OnChanged(func() {
		changed++
	})

	if p.ActiveTab() != PrefTabGeneral {
		t.Errorf("expected default tab general, got %s", p.ActiveTab())
	}
	if p.IsDirty() {
		t.Error("expected not dirty initially")
	}
	if p.Settings().General.Language != "en" {
		t.Errorf("expected language 'en', got %s", p.Settings().General.Language)
	}

	for _, tab := range AllPreferencesTabs {
		p.SelectTab(tab)
		if p.ActiveTab() != tab {
			t.Errorf("expected tab %s, got %s", tab, p.ActiveTab())
		}
	}
	if changed != len(AllPreferencesTabs) {
		t.Errorf("expected %d notifications, got %d", len(AllPreferencesTabs), changed)
	}
}

func TestPreferencesPresenter_MutationsAndRevert(t *testing.T) {
	initial := model.DefaultSettings()
	saver := &fakePreferencesSaver{}
	p := NewPreferencesPresenter(initial, saver, nil, nil)

	// General
	p.SetGeneral(model.GeneralSettings{Language: "fr", AutosaveIntervalSec: 10})
	if !p.IsDirty() {
		t.Error("expected dirty after SetGeneral")
	}
	if p.Settings().General.Language != "fr" {
		t.Errorf("expected 'fr', got %s", p.Settings().General.Language)
	}

	// Appearance
	p.SetAppearance(model.AppearanceSettings{Theme: "dark", Zoom: 120})
	if p.Settings().Appearance.Theme != "dark" {
		t.Errorf("expected 'dark', got %s", p.Settings().Appearance.Theme)
	}

	// Request
	p.SetRequest(model.RequestDefaults{TimeoutMs: 5000})
	if p.Settings().Request.TimeoutMs != 5000 {
		t.Errorf("expected timeout 5000, got %d", p.Settings().Request.TimeoutMs)
	}

	// Proxy
	p.SetProxy(model.ProxySettings{Mode: model.ProxyManual, Hostname: "127.0.0.1", Port: "8080"})
	if p.Settings().Proxy.Hostname != "127.0.0.1" {
		t.Errorf("expected proxy host 127.0.0.1, got %s", p.Settings().Proxy.Hostname)
	}

	// TLS
	p.SetTLS(model.TLSSettings{VerifyServerCertificates: false})
	if p.Settings().TLS.VerifyServerCertificates {
		t.Error("expected verify false")
	}

	// Editor
	p.SetEditor(model.EditorSettings{FontSize: 16})
	if p.Settings().Editor.FontSize != 16 {
		t.Errorf("expected font size 16, got %d", p.Settings().Editor.FontSize)
	}

	// Privacy
	p.SetPrivacy(model.PrivacySettings{TelemetryEnabled: true})
	if !p.Settings().Privacy.TelemetryEnabled {
		t.Error("expected telemetry true")
	}

	// Revert
	p.Revert()
	if p.IsDirty() {
		t.Error("expected not dirty after revert")
	}
	if p.Settings().General.Language != "en" {
		t.Errorf("expected reverted language 'en', got %s", p.Settings().General.Language)
	}
}

func TestPreferencesPresenter_SaveAndReset(t *testing.T) {
	initial := model.DefaultSettings()
	saver := &fakePreferencesSaver{}
	p := NewPreferencesPresenter(initial, saver, nil, nil)

	p.SetGeneral(model.GeneralSettings{Language: "de"})
	if err := p.Save(context.Background()); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if p.IsDirty() {
		t.Error("expected clean after save")
	}
	if saver.savedSettings.General.Language != "de" {
		t.Errorf("saver got language %s", saver.savedSettings.General.Language)
	}

	// Save when not dirty is a no-op
	if err := p.Save(context.Background()); err != nil {
		t.Errorf("save clean no-op failed: %v", err)
	}

	// ResetDefaults
	p.ResetDefaults()
	if !p.IsDirty() {
		t.Error("expected dirty after ResetDefaults")
	}
	if p.Settings().General.Language != "en" {
		t.Errorf("expected default language 'en', got %s", p.Settings().General.Language)
	}
}

func TestPreferencesPresenter_LoadAndOAuthCleaner(t *testing.T) {
	initial := model.DefaultSettings()
	loaded := model.DefaultSettings()
	loaded.General.Language = "ja"

	loader := &fakePreferencesLoader{loadedSettings: &loaded}
	cleaner := &fakeOAuthCleaner{}

	p := NewPreferencesPresenter(initial, nil, loader, cleaner)

	// Load
	if err := p.Load(context.Background()); err != nil {
		t.Fatalf("load error: %v", err)
	}
	if p.Settings().General.Language != "ja" {
		t.Errorf("expected loaded language 'ja', got %s", p.Settings().General.Language)
	}

	// Load error
	loader.err = errors.New("load failed")
	if err := p.Load(context.Background()); err == nil {
		t.Error("expected error from loader")
	}

	// ClearOAuthCache
	if err := p.ClearOAuthCache(context.Background(), "cred-123"); err != nil {
		t.Fatalf("oauth clean error: %v", err)
	}
	if cleaner.clearedID != "cred-123" {
		t.Errorf("expected clearedID 'cred-123', got %s", cleaner.clearedID)
	}

	cleaner.err = errors.New("purge failed")
	if err := p.ClearOAuthCache(context.Background(), "cred-123"); err == nil {
		t.Error("expected error from cleaner")
	}
}
