package settings

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

type mockPrefSaver struct {
	savedSettings *model.Settings
	saveCount     int
}

func (m *mockPrefSaver) SavePreferences(ctx context.Context, settings model.Settings) error {
	m.savedSettings = &settings
	m.saveCount++
	return nil
}

type mockOAuthCleaner struct {
	clearedCredID string
	cleanCount    int
}

func (m *mockOAuthCleaner) ClearOAuth2Cache(ctx context.Context, credID string) error {
	m.clearedCredID = credID
	m.cleanCount++
	return nil
}

type mockColSaver struct {
	savedUID    string
	savedConfig model.CollectionConfig
	saveCount   int
}

func (m *mockColSaver) SaveCollectionConfig(ctx context.Context, colUID string, config model.CollectionConfig) error {
	m.savedUID = colUID
	m.savedConfig = config
	m.saveCount++
	return nil
}

type mockFolderSaver struct {
	savedColUID string
	savedFolder *model.Item
	saveCount   int
}

func (m *mockFolderSaver) SaveFolder(ctx context.Context, colUID string, folder *model.Item) error {
	m.savedColUID = colUID
	m.savedFolder = folder
	m.saveCount++
	return nil
}

func strPtr(s string) *string {
	return &s
}

func TestPreferencesScreen_AllTabsAndInteractions(t *testing.T) {
	prefSaver := &mockPrefSaver{}
	oauthCleaner := &mockOAuthCleaner{}
	initSettings := model.DefaultSettings()

	p := presenters.NewPreferencesPresenter(initSettings, prefSaver, nil, oauthCleaner)

	view := func(c *ui.Context) {
		PreferencesScreen(c, p)
	}

	tt := ui.NewTester(view, 900, 700)
	tt.Frame()

	// Title
	if !tt.HasText("Preferences") {
		t.Error("expected 'Preferences' header text")
	}

	// Verify all category tabs exist
	for _, tab := range []string{"General", "Appearance", "Request", "Proxy", "TLS", "Editor", "Privacy", "OAuth"} {
		if !tt.HasText(tab) {
			t.Errorf("expected tab %q to be visible", tab)
		}
	}

	// Switch tabs and verify content
	p.SelectTab(presenters.PrefTabGeneral)
	tt.Frame()
	if !tt.HasText("General Settings") {
		t.Error("expected General Settings section")
	}

	p.SelectTab(presenters.PrefTabAppearance)
	tt.Frame()
	if !tt.HasText("Appearance Settings") {
		t.Error("expected Appearance Settings section")
	}

	p.SelectTab(presenters.PrefTabRequest)
	tt.Frame()
	if !tt.HasText("Request Defaults") {
		t.Error("expected Request Defaults section")
	}

	p.SelectTab(presenters.PrefTabProxy)
	tt.Frame()
	if !tt.HasText("Proxy Configuration") {
		t.Error("expected Proxy Configuration section")
	}

	// Switch to Manual Proxy Mode
	proxy := p.Settings().Proxy
	proxy.Mode = model.ProxyManual
	p.SetProxy(proxy)
	tt.Frame()
	if !tt.HasText("Hostname") || !tt.HasText("Port") {
		t.Error("expected Hostname and Port inputs when manual proxy is enabled")
	}

	// Switch to PAC Proxy Mode
	proxy.Mode = model.ProxyPAC
	p.SetProxy(proxy)
	tt.Frame()
	if !tt.HasText("PAC URL") {
		t.Error("expected PAC URL input when PAC proxy is enabled")
	}

	p.SelectTab(presenters.PrefTabTLS)
	tt.Frame()
	if !tt.HasText("TLS & SSL Certificates") {
		t.Error("expected TLS & SSL Certificates section")
	}

	p.SelectTab(presenters.PrefTabEditor)
	tt.Frame()
	if !tt.HasText("Code Editor Settings") {
		t.Error("expected Code Editor Settings section")
	}

	p.SelectTab(presenters.PrefTabPrivacy)
	tt.Frame()
	if !tt.HasText("Privacy & Data Storage") {
		t.Error("expected Privacy & Data Storage section")
	}

	p.SelectTab(presenters.PrefTabOAuth)
	tt.Frame()
	if !tt.HasText("OAuth 2.0 Credentials Cache") {
		t.Error("expected OAuth 2.0 section")
	}

	// Click Clear OAuth2 Token Cache
	err := tt.Click("Clear OAuth2 Token Cache")
	if err != nil {
		t.Fatalf("failed to click Clear OAuth2 Token Cache: %v", err)
	}
	if oauthCleaner.cleanCount != 1 {
		t.Errorf("expected cleanCount to be 1, got %d", oauthCleaner.cleanCount)
	}

	// Check dirty and Save
	if !p.IsDirty() {
		t.Error("expected preferences to be dirty after modifications")
	}
	if !tt.HasText("Save *") {
		t.Error("expected 'Save *' button when dirty")
	}

	err = tt.Click("Save *")
	if err != nil {
		t.Fatalf("failed to click Save: %v", err)
	}
	tt.Frame()

	if prefSaver.saveCount != 1 {
		t.Errorf("expected prefSaver.saveCount to be 1, got %d", prefSaver.saveCount)
	}
	if p.IsDirty() {
		t.Error("expected presenter to no longer be dirty after saving")
	}

	// Mutate and Revert
	g := p.Settings().General
	g.Language = "fr"
	p.SetGeneral(g)
	if !p.IsDirty() {
		t.Fatal("expected dirty after SetGeneral")
	}
	tt.Frame()

	err = tt.Click("Revert")
	if err != nil {
		t.Fatalf("failed to click Revert: %v", err)
	}
	if p.IsDirty() {
		t.Error("expected presenter to not be dirty after revert")
	}

	// Accessibility check
	nodes := tt.AccessAll()
	if len(nodes) == 0 {
		t.Error("expected accessibility elements in PreferencesScreen")
	}

	// Nil presenter check
	ttNil := ui.NewTester(func(c *ui.Context) {
		PreferencesScreen(c, nil)
	}, 400, 300)
	ttNil.Frame()
}

func TestCollectionSettingsScreen_AllTabsAndInteractions(t *testing.T) {
	colSaver := &mockColSaver{}
	folderSaver := &mockFolderSaver{}

	col := &model.Collection{
		UID:  "col-1",
		Name: "Petstore API",
		Config: model.CollectionConfig{
			Auth: &model.Auth{
				Mode:     model.AuthBasic,
				Username: strPtr("admin"),
				Password: strPtr("secret"),
			},
			Headers: []model.KeyValue{
				{Name: strPtr("X-App"), Value: strPtr("Fearless"), Enabled: true},
			},
			Vars: []model.Variable{
				{Name: strPtr("host"), Value: "api.petstore.com", Enabled: true},
			},
			Script: &model.Scripts{
				Req: strPtr("console.log('init');"),
			},
			Docs: strPtr("# Petstore Overview"),
		},
	}

	p := presenters.NewCollectionSettingsPresenter(col, colSaver)
	st := &CollectionSettingsState{}

	view := func(c *ui.Context) {
		CollectionSettingsScreen(c, p, st)
	}

	tt := ui.NewTester(view, 900, 700)
	tt.Frame()

	if !tt.HasText("Petstore API") {
		t.Error("expected 'Petstore API' title")
	}

	// Verify all sub-tab labels
	for _, tab := range []string{"Auth", "Headers", "Vars", "Script", "Docs"} {
		if !tt.HasText(tab) {
			t.Errorf("expected sub-tab %q to be visible", tab)
		}
	}

	// Switch sub-tabs
	p.SelectSubTab(presenters.SettingsSubTabAuth)
	tt.Frame()
	if !tt.HasText("Default Authentication:") {
		t.Error("expected Default Authentication section")
	}

	// Test auth mode switching in collection
	authModes := []struct {
		mode model.AuthMode
		key  string
	}{
		{model.AuthNone, "Requests without explicit authentication"},
		{model.AuthBearer, "Bearer token"},
		{model.AuthAPIKey, "Header / Param name"},
	}
	for _, am := range authModes {
		p.SetAuth(&model.Auth{
			Mode:  am.mode,
			Token: strPtr("tok"),
			Key:   strPtr("k"),
			Value: strPtr("v"),
		})
		tt.Frame()
	}

	p.SelectSubTab(presenters.SettingsSubTabHeaders)
	tt.Frame()
	if !tt.HasText("Default Headers (applied to all requests in scope)") {
		t.Error("expected Default Headers section")
	}
	// Click + Add Row
	err := tt.Click("+ Add Row")
	if err != nil {
		t.Fatalf("failed to click + Add Row in headers: %v", err)
	}
	tt.Frame()

	p.SelectSubTab(presenters.SettingsSubTabVars)
	tt.Frame()
	if !tt.HasText("Collection Variables") {
		t.Error("expected Collection Variables section")
	}
	// Click + Add Row
	err = tt.Click("+ Add Row")
	if err != nil {
		t.Fatalf("failed to click + Add Row in vars: %v", err)
	}
	tt.Frame()

	p.SelectSubTab(presenters.SettingsSubTabScript)
	tt.Frame()
	if !tt.HasText("Collection Pre-Request Script") {
		t.Error("expected Collection Pre-Request Script section")
	}

	p.SelectSubTab(presenters.SettingsSubTabDocs)
	tt.Frame()
	if !tt.HasText("Collection Documentation (Markdown)") {
		t.Error("expected Collection Documentation section")
	}

	// Save collection settings
	if !p.IsDirty() {
		t.Error("expected collection settings to be dirty after edits")
	}
	err = tt.Click("Save *")
	if err != nil {
		t.Fatalf("failed to click Save: %v", err)
	}

	if colSaver.saveCount != 1 || colSaver.savedUID != "col-1" {
		t.Errorf("expected colSaver to save once with col-1, got count=%d, uid=%q",
			colSaver.saveCount, colSaver.savedUID)
	}
	if p.IsDirty() {
		t.Error("expected not dirty after save")
	}

	// Test folder settings
	folder := &model.Item{
		UID:  "f-1",
		Name: "Pets Folder",
		Type: model.ItemFolder,
	}
	pFolder := presenters.NewFolderSettingsPresenter("col-1", folder, folderSaver)
	ttFolder := ui.NewTester(func(c *ui.Context) {
		CollectionSettingsScreen(c, pFolder)
	}, 800, 600)
	ttFolder.Frame()

	if !ttFolder.HasText("Pets Folder") {
		t.Error("expected 'Pets Folder' target title")
	}

	// Accessibility
	nodes := tt.AccessAll()
	if len(nodes) == 0 {
		t.Error("expected accessibility nodes in CollectionSettingsScreen")
	}

	// Nil presenter check
	ttNil := ui.NewTester(func(c *ui.Context) {
		CollectionSettingsScreen(c, nil)
	}, 400, 300)
	ttNil.Frame()
}
