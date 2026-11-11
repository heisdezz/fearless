package commands_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"fearless/internal/app/bus"
	"fearless/internal/app/commands"
	"fearless/internal/core/engine/auth"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

func TestPreferencesService_Defaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "preferences.json")

	b := bus.New()
	defer b.Close()
	store := bus.NewStore(nil, b)
	defer store.Close()

	svc := commands.NewPreferencesService(store, b, cfgPath, nil)
	if svc.ConfigPath() != cfgPath {
		t.Fatalf("expected config path %s, got %s", cfgPath, svc.ConfigPath())
	}

	loaded, err := svc.LoadPreferences(context.Background(), commands.LoadPreferencesCmd{})
	if err != nil {
		t.Fatalf("unexpected error loading preferences: %v", err)
	}

	defaults := model.DefaultSettings()
	if loaded.SchemaVersion != defaults.SchemaVersion {
		t.Errorf("expected SchemaVersion %d, got %d", defaults.SchemaVersion, loaded.SchemaVersion)
	}
	if loaded.Appearance.Theme != defaults.Appearance.Theme {
		t.Errorf("expected Theme %s, got %s", defaults.Appearance.Theme, loaded.Appearance.Theme)
	}

	snap := store.Snapshot()
	if snap.Settings.SchemaVersion != defaults.SchemaVersion {
		t.Errorf("expected store settings to have SchemaVersion %d, got %d", defaults.SchemaVersion, snap.Settings.SchemaVersion)
	}
}

func TestPreferencesService_SaveAndReload(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "preferences.json")

	b := bus.New()
	defer b.Close()
	store := bus.NewStore(nil, b)
	defer store.Close()

	svc := commands.NewPreferencesService(store, b, cfgPath, nil)

	custom := model.DefaultSettings()
	custom.Appearance.Theme = "dark"
	custom.Appearance.Zoom = 125
	custom.Request.TimeoutMs = 15000
	custom.Proxy.Mode = model.ProxyManual
	custom.Proxy.Hostname = "127.0.0.1"
	custom.Proxy.Port = "8080"

	err := svc.SavePreferences(context.Background(), commands.SavePreferencesCmd{
		Settings: custom,
	})
	if err != nil {
		t.Fatalf("unexpected error saving preferences: %v", err)
	}

	// Verify state in store
	snap := store.Snapshot()
	if snap.Settings.Appearance.Theme != "dark" {
		t.Errorf("expected store theme to be dark, got %s", snap.Settings.Appearance.Theme)
	}
	if snap.Settings.Appearance.Zoom != 125 {
		t.Errorf("expected store zoom to be 125, got %d", snap.Settings.Appearance.Zoom)
	}

	// Reload with a new service instance reading the file from disk
	newStore := bus.NewStore(nil, b)
	defer newStore.Close()
	newSvc := commands.NewPreferencesService(newStore, b, cfgPath, nil)

	reloaded, err := newSvc.LoadPreferences(context.Background(), commands.LoadPreferencesCmd{})
	if err != nil {
		t.Fatalf("unexpected error reloading preferences: %v", err)
	}

	if reloaded.Appearance.Theme != "dark" || reloaded.Appearance.Zoom != 125 || reloaded.Request.TimeoutMs != 15000 {
		t.Errorf("reloaded preferences mismatch: %+v", reloaded)
	}
	if reloaded.Proxy.Mode != model.ProxyManual || reloaded.Proxy.Hostname != "127.0.0.1" {
		t.Errorf("reloaded proxy mismatch: %+v", reloaded.Proxy)
	}
}

func TestPreferencesService_Idempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "preferences.json")

	b := bus.New()
	defer b.Close()
	store := bus.NewStore(nil, b)
	defer store.Close()

	svc := commands.NewPreferencesService(store, b, cfgPath, nil)
	settings := model.DefaultSettings()
	settings.General.Language = "fr"

	for i := 0; i < 3; i++ {
		err := svc.SavePreferences(context.Background(), commands.SavePreferencesCmd{
			Settings: settings,
		})
		if err != nil {
			t.Fatalf("save iteration %d failed: %v", i, err)
		}
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}
	if !strings.Contains(string(data), `"language": "fr"`) {
		t.Fatalf("file content does not contain language fr: %s", string(data))
	}
}

func TestPreferencesService_SanitizeBounds(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "preferences.json")

	b := bus.New()
	defer b.Close()
	store := bus.NewStore(nil, b)
	defer store.Close()

	svc := commands.NewPreferencesService(store, b, cfgPath, nil)

	bogus := model.Settings{
		SchemaVersion: -5,
		Appearance: model.AppearanceSettings{
			Zoom:  -10,
			Theme: "",
		},
		Request: model.RequestDefaults{
			MaxRedirects:     -1,
			MaxResponseBytes: -100,
			InlineBodyBytes:  0,
		},
		Editor: model.EditorSettings{
			FontSize: -2,
			TabSize:  0,
		},
		General: model.GeneralSettings{
			Language: "",
		},
		Privacy: model.PrivacySettings{
			HistoryRetainDays: -5,
		},
	}

	err := svc.SavePreferences(context.Background(), commands.SavePreferencesCmd{
		Settings: bogus,
	})
	if err != nil {
		t.Fatalf("unexpected error saving bogus settings: %v", err)
	}

	snap := store.Snapshot()
	if snap.Settings.SchemaVersion != model.CurrentSettingsSchemaVersion {
		t.Errorf("expected SchemaVersion %d, got %d", model.CurrentSettingsSchemaVersion, snap.Settings.SchemaVersion)
	}
	if snap.Settings.Appearance.Zoom != 100 {
		t.Errorf("expected Zoom clamped to 100, got %d", snap.Settings.Appearance.Zoom)
	}
	if snap.Settings.Appearance.Theme != "system" {
		t.Errorf("expected Theme defaulted to system, got %s", snap.Settings.Appearance.Theme)
	}
	if snap.Settings.Request.MaxRedirects != model.DefaultMaxRedirects {
		t.Errorf("expected MaxRedirects default %d, got %d", model.DefaultMaxRedirects, snap.Settings.Request.MaxRedirects)
	}
	if snap.Settings.Request.MaxResponseBytes != 512<<20 {
		t.Errorf("expected MaxResponseBytes default 512MiB, got %d", snap.Settings.Request.MaxResponseBytes)
	}
	if snap.Settings.Request.InlineBodyBytes != 32<<20 {
		t.Errorf("expected InlineBodyBytes default 32MiB, got %d", snap.Settings.Request.InlineBodyBytes)
	}
	if snap.Settings.Editor.FontSize != 13 {
		t.Errorf("expected FontSize default 13, got %d", snap.Settings.Editor.FontSize)
	}
	if snap.Settings.Editor.TabSize != 2 {
		t.Errorf("expected TabSize default 2, got %d", snap.Settings.Editor.TabSize)
	}
	if snap.Settings.General.Language != "en" {
		t.Errorf("expected Language default en, got %s", snap.Settings.General.Language)
	}
	if snap.Settings.Privacy.HistoryRetainDays != 30 {
		t.Errorf("expected HistoryRetainDays default 30, got %d", snap.Settings.Privacy.HistoryRetainDays)
	}
}

func TestPreferencesService_SchemaMigration(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "legacy_preferences.json")

	// Legacy JSON format (schemaVersion missing/0, legacy proxy enabled boolean)
	legacyJSON := `{
		"general": {
			"confirmOnClose": false
		},
		"proxy": {
			"enabled": true,
			"hostname": "proxy.corp.internal",
			"port": "3128"
		}
	}`

	if err := os.WriteFile(cfgPath, []byte(legacyJSON), 0o644); err != nil {
		t.Fatalf("failed to write legacy preferences: %v", err)
	}

	b := bus.New()
	defer b.Close()
	store := bus.NewStore(nil, b)
	defer store.Close()

	svc := commands.NewPreferencesService(store, b, cfgPath, nil)

	loaded, err := svc.LoadPreferences(context.Background(), commands.LoadPreferencesCmd{})
	if err != nil {
		t.Fatalf("unexpected error loading legacy preferences: %v", err)
	}

	if loaded.SchemaVersion != model.CurrentSettingsSchemaVersion {
		t.Errorf("expected migrated SchemaVersion %d, got %d", model.CurrentSettingsSchemaVersion, loaded.SchemaVersion)
	}
	if loaded.Proxy.Mode != model.ProxyManual {
		t.Errorf("expected legacy proxy.enabled: true to migrate to mode 'manual', got %q", loaded.Proxy.Mode)
	}
	if loaded.Proxy.Hostname != "proxy.corp.internal" || loaded.Proxy.Port != "3128" {
		t.Errorf("expected proxy hostname/port preserved, got %s:%s", loaded.Proxy.Hostname, loaded.Proxy.Port)
	}
	if loaded.General.ConfirmOnClose != false {
		t.Errorf("expected general.confirmOnClose false preserved")
	}
	// Check default backfills
	if loaded.Request.MaxRedirects != model.DefaultMaxRedirects {
		t.Errorf("expected default MaxRedirects %d, got %d", model.DefaultMaxRedirects, loaded.Request.MaxRedirects)
	}
	if loaded.Appearance.Theme != "system" {
		t.Errorf("expected default theme system, got %s", loaded.Appearance.Theme)
	}

	// Verify the upgraded settings were automatically written back to disk
	diskData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to re-read file: %v", err)
	}
	var reParsed model.Settings
	if err := json.Unmarshal(diskData, &reParsed); err != nil {
		t.Fatalf("failed to unmarshal re-saved file: %v", err)
	}
	if reParsed.SchemaVersion != model.CurrentSettingsSchemaVersion {
		t.Errorf("expected file on disk to have SchemaVersion %d, got %d", model.CurrentSettingsSchemaVersion, reParsed.SchemaVersion)
	}
}

func TestPreferencesService_CorruptFileQuarantine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "corrupt_preferences.json")

	// Write unparseable truncated JSON
	corruptContent := []byte(`{"general": {"confirmOnClose": tr`)
	if err := os.WriteFile(cfgPath, corruptContent, 0o644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	b := bus.New()
	defer b.Close()
	eventsCh, cancelSub := b.Subscribe()
	defer cancelSub()

	store := bus.NewStore(nil, b)
	defer store.Close()

	svc := commands.NewPreferencesService(store, b, cfgPath, nil)

	loaded, err := svc.LoadPreferences(context.Background(), commands.LoadPreferencesCmd{})
	if err != nil {
		t.Fatalf("expected fallback to defaults rather than hard error, got: %v", err)
	}

	defaults := model.DefaultSettings()
	if loaded.SchemaVersion != defaults.SchemaVersion {
		t.Errorf("expected loaded defaults SchemaVersion %d, got %d", defaults.SchemaVersion, loaded.SchemaVersion)
	}

	// Check that corrupt file was quarantined
	matches, err := filepath.Glob(filepath.Join(dir, "corrupt_preferences.json.corrupt-*"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("expected quarantined corrupt file, found none (err: %v)", err)
	}
	quarantinedData, err := os.ReadFile(matches[0])
	if err != nil || string(quarantinedData) != string(corruptContent) {
		t.Fatalf("quarantined content mismatch: %s", string(quarantinedData))
	}

	// Verify ErrorOccurred was emitted on the bus with ScopeSettings
	select {
	case ev := <-eventsCh:
		errEv, ok := ev.(events.ErrorOccurred)
		if !ok {
			t.Fatalf("expected events.ErrorOccurred, got %T", ev)
		}
		if errEv.Scope != events.ScopeSettings {
			t.Errorf("expected scope %s, got %s", events.ScopeSettings, errEv.Scope)
		}
		if !strings.Contains(errEv.Msg, "quarantined") {
			t.Errorf("expected quarantined in error message, got: %s", errEv.Msg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for quarantine ErrorOccurred event")
	}
}

func TestPreferencesService_OAuthCache_Memory(t *testing.T) {
	t.Parallel()

	b := bus.New()
	defer b.Close()
	eventsCh, cancelSub := b.Subscribe()
	defer cancelSub()

	store := bus.NewStore(nil, b)
	defer store.Close()

	cache := commands.NewMemoryOAuthTokenCache()
	svc := commands.NewPreferencesService(store, b, filepath.Join(t.TempDir(), "pref.json"), cache)

	tokenFunc := svc.TokenStore()
	if _, ok := tokenFunc("cred-1"); ok {
		t.Fatal("expected empty token for cred-1")
	}

	token1 := auth.Token{
		AccessToken:  "access-abc-123",
		RefreshToken: "refresh-xyz-789",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Unix() + 3600,
		Scope:        "read write",
	}

	// Store token
	if err := svc.SetToken("cred-1", token1); err != nil {
		t.Fatalf("set token failed: %v", err)
	}

	// Verify event emitted for set token
	select {
	case ev := <-eventsCh:
		credEv, ok := ev.(events.CredentialsUpdated)
		if !ok || credEv.CredentialsID != "cred-1" {
			t.Fatalf("expected CredentialsUpdated for cred-1, got %+v", ev)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for set token event")
	}

	// Retrieve token
	got, ok := svc.GetToken("cred-1")
	if !ok || got.AccessToken != token1.AccessToken {
		t.Fatalf("retrieved token mismatch: %+v", got)
	}

	// TokenStore func also returns it
	fnGot, fnOk := tokenFunc("cred-1")
	if !fnOk || fnGot.AccessToken != token1.AccessToken {
		t.Fatalf("tokenFunc token mismatch: %+v", fnGot)
	}

	// Add a second token
	token2 := auth.Token{
		AccessToken: "access-def-456",
		TokenType:   "Bearer",
	}
	_ = svc.SetToken("cred-2", token2)
	<-eventsCh // consume event

	keys := cache.Keys()
	if len(keys) != 2 || keys[0] != "cred-1" || keys[1] != "cred-2" {
		t.Fatalf("unexpected keys: %v", keys)
	}

	// Delete single token
	if err := svc.ClearOAuth2Cache(context.Background(), commands.ClearOAuthCacheCmd{
		CredentialsID: "cred-1",
	}); err != nil {
		t.Fatalf("clear single token failed: %v", err)
	}

	select {
	case ev := <-eventsCh:
		credEv, ok := ev.(events.CredentialsUpdated)
		if !ok || credEv.CredentialsID != "cred-1" {
			t.Fatalf("expected CredentialsUpdated for cred-1, got %+v", ev)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for single token clear event")
	}

	if _, ok := svc.GetToken("cred-1"); ok {
		t.Fatal("expected cred-1 to be deleted")
	}
	if _, ok := svc.GetToken("cred-2"); !ok {
		t.Fatal("expected cred-2 to still exist")
	}

	// Clear all tokens
	if err := svc.ClearOAuth2Cache(context.Background(), commands.ClearOAuthCacheCmd{}); err != nil {
		t.Fatalf("clear all failed: %v", err)
	}

	select {
	case ev := <-eventsCh:
		credEv, ok := ev.(events.CredentialsUpdated)
		if !ok || credEv.CredentialsID != "cred-2" {
			t.Fatalf("expected CredentialsUpdated for cred-2, got %+v", ev)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for cred-2 clear event")
	}

	if len(cache.Keys()) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(cache.Keys()))
	}

	// Idempotent: clear on empty cache succeeds
	if err := svc.ClearOAuth2Cache(context.Background(), commands.ClearOAuthCacheCmd{}); err != nil {
		t.Fatalf("clear on empty cache failed: %v", err)
	}
}

func TestPreferencesService_OAuthCache_FileBacked(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "tokens.json")

	fileCache, err := commands.NewFileOAuthTokenCache(cachePath)
	if err != nil {
		t.Fatalf("new file cache failed: %v", err)
	}

	b := bus.New()
	defer b.Close()
	store := bus.NewStore(nil, b)
	defer store.Close()

	svc := commands.NewPreferencesService(store, b, filepath.Join(dir, "pref.json"), fileCache)

	token := auth.Token{
		AccessToken: "persisted-token-xyz",
		TokenType:   "Bearer",
	}

	if err := svc.SetToken("persisted-cred", token); err != nil {
		t.Fatalf("set token failed: %v", err)
	}

	// Verify file exists on disk
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("token cache file does not exist on disk: %v", err)
	}

	// Create a new FileOAuthTokenCache instance reading from the same file
	freshCache, err := commands.NewFileOAuthTokenCache(cachePath)
	if err != nil {
		t.Fatalf("reload file cache failed: %v", err)
	}

	loadedToken, ok := freshCache.Get("persisted-cred")
	if !ok || loadedToken.AccessToken != "persisted-token-xyz" {
		t.Fatalf("loaded token mismatch: %+v", loadedToken)
	}

	// Delete from fresh cache
	if err := freshCache.Delete("persisted-cred"); err != nil {
		t.Fatalf("delete token failed: %v", err)
	}
	if _, ok := freshCache.Get("persisted-cred"); ok {
		t.Fatal("expected token to be deleted")
	}

	// Reload again to verify persistence of deletion
	reloadedCache, err := commands.NewFileOAuthTokenCache(cachePath)
	if err != nil {
		t.Fatalf("reload after delete failed: %v", err)
	}
	if len(reloadedCache.Keys()) != 0 {
		t.Fatalf("expected 0 keys after deletion, got %d", len(reloadedCache.Keys()))
	}
}

func TestPreferencesService_ContextCancellation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "preferences.json")

	svc := commands.NewPreferencesService(nil, nil, cfgPath, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := svc.SavePreferences(ctx, commands.SavePreferencesCmd{Settings: model.DefaultSettings()}); err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	if _, err := svc.LoadPreferences(ctx, commands.LoadPreferencesCmd{}); err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	if err := svc.ClearOAuth2Cache(ctx, commands.ClearOAuthCacheCmd{}); err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestPreferencesService_Adapters(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "preferences.json")

	b := bus.New()
	defer b.Close()
	store := bus.NewStore(nil, b)
	defer store.Close()

	svc := commands.NewPreferencesService(store, b, cfgPath, nil)

	var saver commands.PreferencesSaver = svc
	var loader commands.PreferencesLoader = svc
	var cleaner commands.OAuthCacheCleaner = svc

	custom := model.DefaultSettings()
	custom.General.Language = "de"

	if err := saver.SavePreferences(context.Background(), commands.SavePreferencesCmd{Settings: custom}); err != nil {
		t.Fatalf("saver failed: %v", err)
	}

	loaded, err := loader.LoadPreferences(context.Background(), commands.LoadPreferencesCmd{})
	if err != nil || loaded.General.Language != "de" {
		t.Fatalf("loader failed: %v, loaded: %+v", err, loaded)
	}

	if err := cleaner.ClearOAuth2Cache(context.Background(), commands.ClearOAuthCacheCmd{}); err != nil {
		t.Fatalf("cleaner failed: %v", err)
	}

	// Test Execute-based adapters
	saverAdapter := svc.Saver()
	loaderAdapter := svc.Loader()
	cleanerAdapter := svc.Cleaner()

	custom.General.Language = "es"
	if err := saverAdapter.Execute(context.Background(), commands.SavePreferencesCmd{Settings: custom}); err != nil {
		t.Fatalf("saverAdapter Execute failed: %v", err)
	}

	execLoaded, err := loaderAdapter.Execute(context.Background(), commands.LoadPreferencesCmd{})
	if err != nil || execLoaded.General.Language != "es" {
		t.Fatalf("loaderAdapter Execute failed: %v, loaded: %+v", err, execLoaded)
	}

	if err := cleanerAdapter.Execute(context.Background(), commands.ClearOAuthCacheCmd{}); err != nil {
		t.Fatalf("cleanerAdapter Execute failed: %v", err)
	}
}

func TestPreferencesService_Concurrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "preferences.json")

	b := bus.New()
	defer b.Close()
	store := bus.NewStore(nil, b)
	defer store.Close()

	svc := commands.NewPreferencesService(store, b, cfgPath, nil)

	var wg sync.WaitGroup
	ctx := context.Background()

	// 10 concurrent savers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			s := model.DefaultSettings()
			s.Appearance.Zoom = 100 + idx*5
			_ = svc.SavePreferences(ctx, commands.SavePreferencesCmd{Settings: s})
		}()
	}

	// 10 concurrent loaders
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.LoadPreferences(ctx, commands.LoadPreferencesCmd{})
		}()
	}

	// 10 concurrent oauth token modifiers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()
			credID := "cred"
			_ = svc.SetToken(credID, auth.Token{AccessToken: "token"})
			_, _ = svc.GetToken(credID)
			if idx%2 == 0 {
				_ = svc.ClearOAuth2Cache(ctx, commands.ClearOAuthCacheCmd{CredentialsID: credID})
			}
		}()
	}

	wg.Wait()
}
