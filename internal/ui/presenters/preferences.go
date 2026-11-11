package presenters

import (
	"context"
	"sync"

	"fearless/internal/core/model"
)

// PreferencesTab selects a category in the preferences modal.
type PreferencesTab string

const (
	PrefTabGeneral    PreferencesTab = "general"
	PrefTabAppearance PreferencesTab = "appearance"
	PrefTabRequest    PreferencesTab = "request"
	PrefTabProxy      PreferencesTab = "proxy"
	PrefTabTLS        PreferencesTab = "tls"
	PrefTabEditor     PreferencesTab = "editor"
	PrefTabPrivacy    PreferencesTab = "privacy"
	PrefTabOAuth      PreferencesTab = "oauth"
)

// AllPreferencesTabs lists all preference tabs in standard UI order.
var AllPreferencesTabs = []PreferencesTab{
	PrefTabGeneral,
	PrefTabAppearance,
	PrefTabRequest,
	PrefTabProxy,
	PrefTabTLS,
	PrefTabEditor,
	PrefTabPrivacy,
	PrefTabOAuth,
}

// PreferencesSaverPort is the consumer port to persist application settings.
type PreferencesSaverPort interface {
	SavePreferences(ctx context.Context, settings model.Settings) error
}

// PreferencesLoaderPort is the consumer port to load application settings from disk.
type PreferencesLoaderPort interface {
	LoadPreferences(ctx context.Context) (*model.Settings, error)
}

// OAuthCacheCleanerPort is the consumer port to purge cached OAuth2 tokens.
type OAuthCacheCleanerPort interface {
	ClearOAuth2Cache(ctx context.Context, credentialsID string) error
}

// PreferencesPresenter coordinates application preferences view-models.
type PreferencesPresenter struct {
	mu           sync.RWMutex
	saved        model.Settings
	draft        *model.Settings
	activeTab    PreferencesTab
	saver        PreferencesSaverPort
	loader       PreferencesLoaderPort
	oauthCleaner OAuthCacheCleanerPort
	onChanged    func()
}

// NewPreferencesPresenter creates an initialized PreferencesPresenter.
func NewPreferencesPresenter(
	initial model.Settings,
	saver PreferencesSaverPort,
	loader PreferencesLoaderPort,
	oauthCleaner OAuthCacheCleanerPort,
) *PreferencesPresenter {
	return &PreferencesPresenter{
		saved:        initial,
		activeTab:    PrefTabGeneral,
		saver:        saver,
		loader:       loader,
		oauthCleaner: oauthCleaner,
	}
}

// ActiveTab returns the currently selected preferences category.
func (p *PreferencesPresenter) ActiveTab() PreferencesTab {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeTab
}

// SelectTab changes the active preferences category.
func (p *PreferencesPresenter) SelectTab(tab PreferencesTab) {
	p.mu.Lock()
	p.activeTab = tab
	p.mu.Unlock()
	p.notifyChanged()
}

// Settings returns the effective settings (draft if dirty, otherwise saved).
func (p *PreferencesPresenter) Settings() model.Settings {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.draft != nil {
		return *p.draft
	}
	return p.saved
}

// IsDirty reports whether preferences have uncommitted modifications.
func (p *PreferencesPresenter) IsDirty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.draft != nil
}

func (p *PreferencesPresenter) ensureDraft() *model.Settings {
	if p.draft == nil {
		cloned := p.saved
		p.draft = &cloned
	}
	return p.draft
}

// SetGeneral updates general preferences in the draft.
func (p *PreferencesPresenter) SetGeneral(g model.GeneralSettings) {
	p.mu.Lock()
	draft := p.ensureDraft()
	draft.General = g
	p.mu.Unlock()
	p.notifyChanged()
}

// SetAppearance updates appearance preferences in the draft.
func (p *PreferencesPresenter) SetAppearance(a model.AppearanceSettings) {
	p.mu.Lock()
	draft := p.ensureDraft()
	draft.Appearance = a
	p.mu.Unlock()
	p.notifyChanged()
}

// SetRequest updates default request preferences in the draft.
func (p *PreferencesPresenter) SetRequest(r model.RequestDefaults) {
	p.mu.Lock()
	draft := p.ensureDraft()
	draft.Request = r
	p.mu.Unlock()
	p.notifyChanged()
}

// SetProxy updates proxy preferences in the draft.
func (p *PreferencesPresenter) SetProxy(proxy model.ProxySettings) {
	p.mu.Lock()
	draft := p.ensureDraft()
	draft.Proxy = proxy
	p.mu.Unlock()
	p.notifyChanged()
}

// SetTLS updates TLS/certificate preferences in the draft.
func (p *PreferencesPresenter) SetTLS(tls model.TLSSettings) {
	p.mu.Lock()
	draft := p.ensureDraft()
	draft.TLS = tls
	p.mu.Unlock()
	p.notifyChanged()
}

// SetEditor updates editor preferences in the draft.
func (p *PreferencesPresenter) SetEditor(e model.EditorSettings) {
	p.mu.Lock()
	draft := p.ensureDraft()
	draft.Editor = e
	p.mu.Unlock()
	p.notifyChanged()
}

// SetPrivacy updates privacy preferences in the draft.
func (p *PreferencesPresenter) SetPrivacy(priv model.PrivacySettings) {
	p.mu.Lock()
	draft := p.ensureDraft()
	draft.Privacy = priv
	p.mu.Unlock()
	p.notifyChanged()
}

// ResetDefaults resets the draft to factory default settings.
func (p *PreferencesPresenter) ResetDefaults() {
	p.mu.Lock()
	defaults := model.DefaultSettings()
	p.draft = &defaults
	p.mu.Unlock()
	p.notifyChanged()
}

// Revert discards draft modifications.
func (p *PreferencesPresenter) Revert() {
	p.mu.Lock()
	p.draft = nil
	p.mu.Unlock()
	p.notifyChanged()
}

// Save commits the draft preferences via PreferencesSaverPort.
func (p *PreferencesPresenter) Save(ctx context.Context) error {
	p.mu.Lock()
	if p.draft == nil {
		p.mu.Unlock()
		return nil
	}
	draft := *p.draft
	saver := p.saver
	p.mu.Unlock()

	var err error
	if saver != nil {
		err = saver.SavePreferences(ctx, draft)
	}
	if err == nil {
		p.mu.Lock()
		p.saved = draft
		p.draft = nil
		p.mu.Unlock()
	}
	p.notifyChanged()
	return err
}

// Load reloads preferences from disk via PreferencesLoaderPort.
func (p *PreferencesPresenter) Load(ctx context.Context) error {
	p.mu.Lock()
	loader := p.loader
	p.mu.Unlock()

	if loader == nil {
		return nil
	}

	loaded, err := loader.LoadPreferences(ctx)
	if err == nil && loaded != nil {
		p.mu.Lock()
		p.saved = *loaded
		p.draft = nil
		p.mu.Unlock()
	}
	p.notifyChanged()
	return err
}

// ClearOAuthCache purges cached OAuth tokens via OAuthCacheCleanerPort.
func (p *PreferencesPresenter) ClearOAuthCache(ctx context.Context, credentialsID string) error {
	p.mu.RLock()
	cleaner := p.oauthCleaner
	p.mu.RUnlock()

	if cleaner == nil {
		return nil
	}
	err := cleaner.ClearOAuth2Cache(ctx, credentialsID)
	p.notifyChanged()
	return err
}

// OnChanged registers an invalidation callback hook.
func (p *PreferencesPresenter) OnChanged(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChanged = fn
}

func (p *PreferencesPresenter) notifyChanged() {
	p.mu.RLock()
	fn := p.onChanged
	p.mu.RUnlock()
	if fn != nil {
		fn()
	}
}
