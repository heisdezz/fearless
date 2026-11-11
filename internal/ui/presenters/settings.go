package presenters

import (
	"context"
	"sync"

	"fearless/internal/core/model"
)

// SettingsSubTab selects a configuration tab in the settings editor.
type SettingsSubTab string

const (
	SettingsSubTabAuth    SettingsSubTab = "auth"
	SettingsSubTabHeaders SettingsSubTab = "headers"
	SettingsSubTabVars    SettingsSubTab = "vars"
	SettingsSubTabScript  SettingsSubTab = "script"
	SettingsSubTabDocs    SettingsSubTab = "docs"
)

// AllSettingsSubTabs lists all settings tabs in standard UI order.
var AllSettingsSubTabs = []SettingsSubTab{
	SettingsSubTabAuth,
	SettingsSubTabHeaders,
	SettingsSubTabVars,
	SettingsSubTabScript,
	SettingsSubTabDocs,
}

// CollectionSettingsSaverPort is the consumer port to save collection config changes.
type CollectionSettingsSaverPort interface {
	SaveCollectionConfig(ctx context.Context, colUID string, config model.CollectionConfig) error
}

// FolderSettingsSaverPort is the consumer port to save folder item changes.
type FolderSettingsSaverPort interface {
	SaveFolder(ctx context.Context, colUID string, folder *model.Item) error
}

// SettingsPresenter coordinates collection or folder level configuration view-models.
type SettingsPresenter struct {
	mu           sync.RWMutex
	colUID       string
	collection   *model.Collection
	folder       *model.Item
	activeSubTab SettingsSubTab
	colSaver     CollectionSettingsSaverPort
	folderSaver  FolderSettingsSaverPort
	onChanged    func()

	// Draft state
	isDirty   bool
	draftAuth *model.Auth
	draftHdrs []model.KeyValue
	draftVars []model.Variable
	draftScpt *model.Scripts
	draftDocs string
}

// NewCollectionSettingsPresenter creates a presenter targeting collection-level configuration.
func NewCollectionSettingsPresenter(col *model.Collection, saver CollectionSettingsSaverPort) *SettingsPresenter {
	p := &SettingsPresenter{
		activeSubTab: SettingsSubTabAuth,
		colSaver:     saver,
	}
	p.SetTargetCollection(col)
	return p
}

// NewFolderSettingsPresenter creates a presenter targeting folder-level configuration.
func NewFolderSettingsPresenter(colUID string, folder *model.Item, saver FolderSettingsSaverPort) *SettingsPresenter {
	p := &SettingsPresenter{
		colUID:       colUID,
		activeSubTab: SettingsSubTabAuth,
		folderSaver:  saver,
	}
	p.SetTargetFolder(colUID, folder)
	return p
}

// SetTargetCollection binds a collection as the active target.
func (p *SettingsPresenter) SetTargetCollection(col *model.Collection) {
	p.mu.Lock()
	p.collection = col
	p.folder = nil
	if col != nil {
		p.colUID = col.UID
	}
	p.resetDraftLocked()
	p.mu.Unlock()
	p.notifyChanged()
}

// SetTargetFolder binds a folder item as the active target.
func (p *SettingsPresenter) SetTargetFolder(colUID string, folder *model.Item) {
	p.mu.Lock()
	p.collection = nil
	p.colUID = colUID
	p.folder = folder
	p.resetDraftLocked()
	p.mu.Unlock()
	p.notifyChanged()
}

// IsFolder reports whether the presenter is currently editing a folder rather than a collection.
func (p *SettingsPresenter) IsFolder() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.folder != nil
}

// TargetName returns the name of the collection or folder being configured.
func (p *SettingsPresenter) TargetName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.folder != nil {
		return p.folder.Name
	}
	if p.collection != nil {
		return p.collection.Name
	}
	return ""
}

// ActiveSubTab returns the currently selected sub-tab.
func (p *SettingsPresenter) ActiveSubTab() SettingsSubTab {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeSubTab
}

// SelectSubTab switches the active settings sub-tab.
func (p *SettingsPresenter) SelectSubTab(tab SettingsSubTab) {
	p.mu.Lock()
	p.activeSubTab = tab
	p.mu.Unlock()
	p.notifyChanged()
}

// IsDirty reports whether there are uncommitted settings changes.
func (p *SettingsPresenter) IsDirty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isDirty
}

func (p *SettingsPresenter) resetDraftLocked() {
	p.isDirty = false
	if p.folder != nil {
		p.draftAuth = p.folder.Auth
		p.draftHdrs = p.folder.Headers
		p.draftVars = p.folder.Vars
		p.draftScpt = p.folder.Script
		p.draftDocs = ""
	} else if p.collection != nil {
		cfg := p.collection.Config
		p.draftAuth = cfg.Auth
		p.draftHdrs = cfg.Headers
		p.draftVars = cfg.Vars
		p.draftScpt = cfg.Script
		if cfg.Docs != nil {
			p.draftDocs = *cfg.Docs
		} else {
			p.draftDocs = ""
		}
	}
}

// Auth returns the effective auth configuration.
func (p *SettingsPresenter) Auth() *model.Auth {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.draftAuth
}

// SetAuth updates the draft auth configuration.
func (p *SettingsPresenter) SetAuth(auth *model.Auth) {
	p.mu.Lock()
	p.draftAuth = auth
	p.isDirty = true
	p.mu.Unlock()
	p.notifyChanged()
}

// Headers returns the effective headers list.
func (p *SettingsPresenter) Headers() []model.KeyValue {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.draftHdrs
}

// SetHeaders updates the draft headers list.
func (p *SettingsPresenter) SetHeaders(hdrs []model.KeyValue) {
	p.mu.Lock()
	p.draftHdrs = hdrs
	p.isDirty = true
	p.mu.Unlock()
	p.notifyChanged()
}

// Vars returns the effective variables list.
func (p *SettingsPresenter) Vars() []model.Variable {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.draftVars
}

// SetVars updates the draft variables list.
func (p *SettingsPresenter) SetVars(vars []model.Variable) {
	p.mu.Lock()
	p.draftVars = vars
	p.isDirty = true
	p.mu.Unlock()
	p.notifyChanged()
}

// Scripts returns the effective scripts configuration.
func (p *SettingsPresenter) Scripts() *model.Scripts {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.draftScpt
}

// SetScripts updates the draft scripts configuration.
func (p *SettingsPresenter) SetScripts(scripts *model.Scripts) {
	p.mu.Lock()
	p.draftScpt = scripts
	p.isDirty = true
	p.mu.Unlock()
	p.notifyChanged()
}

// Docs returns the documentation text.
func (p *SettingsPresenter) Docs() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.draftDocs
}

// SetDocs updates the draft documentation text.
func (p *SettingsPresenter) SetDocs(docs string) {
	p.mu.Lock()
	p.draftDocs = docs
	p.isDirty = true
	p.mu.Unlock()
	p.notifyChanged()
}

// Revert discards draft changes and resets to saved values.
func (p *SettingsPresenter) Revert() {
	p.mu.Lock()
	p.resetDraftLocked()
	p.mu.Unlock()
	p.notifyChanged()
}

// Save commits draft modifications to the target collection or folder.
func (p *SettingsPresenter) Save(ctx context.Context) error {
	p.mu.Lock()
	if !p.isDirty {
		p.mu.Unlock()
		return nil
	}

	colUID := p.colUID
	isFolder := p.folder != nil
	folder := p.folder
	col := p.collection
	colSaver := p.colSaver
	folderSaver := p.folderSaver

	auth := p.draftAuth
	hdrs := p.draftHdrs
	vars := p.draftVars
	scpt := p.draftScpt
	docs := p.draftDocs
	p.mu.Unlock()

	var err error
	if isFolder {
		if folderSaver != nil && folder != nil {
			folder.Auth = auth
			folder.Headers = hdrs
			folder.Vars = vars
			folder.Script = scpt
			err = folderSaver.SaveFolder(ctx, colUID, folder)
		}
	} else {
		if colSaver != nil && col != nil {
			col.Config.Auth = auth
			col.Config.Headers = hdrs
			col.Config.Vars = vars
			col.Config.Script = scpt
			if docs != "" {
				col.Config.Docs = &docs
			} else {
				col.Config.Docs = nil
			}
			err = colSaver.SaveCollectionConfig(ctx, colUID, col.Config)
		}
	}

	if err == nil {
		p.mu.Lock()
		p.isDirty = false
		p.mu.Unlock()
	}
	p.notifyChanged()
	return err
}

// OnChanged registers an invalidation callback hook.
func (p *SettingsPresenter) OnChanged(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChanged = fn
}

func (p *SettingsPresenter) notifyChanged() {
	p.mu.RLock()
	fn := p.onChanged
	p.mu.RUnlock()
	if fn != nil {
		fn()
	}
}
