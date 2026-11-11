package presenters

import (
	"context"
	"errors"
	"testing"

	"fearless/internal/core/model"
)

type fakeCollectionSettingsSaver struct {
	savedColUID string
	savedConfig model.CollectionConfig
	err         error
}

func (f *fakeCollectionSettingsSaver) SaveCollectionConfig(ctx context.Context, colUID string, config model.CollectionConfig) error {
	if f.err != nil {
		return f.err
	}
	f.savedColUID = colUID
	f.savedConfig = config
	return nil
}

type fakeFolderSettingsSaver struct {
	savedColUID string
	savedFolder *model.Item
	err         error
}

func (f *fakeFolderSettingsSaver) SaveFolder(ctx context.Context, colUID string, folder *model.Item) error {
	if f.err != nil {
		return f.err
	}
	f.savedColUID = colUID
	f.savedFolder = folder
	return nil
}

func TestSettingsPresenter_CollectionTarget(t *testing.T) {
	docs := "API Documentation"
	col := &model.Collection{
		UID:  "col-set-1",
		Name: "Store API",
		Config: model.CollectionConfig{
			Name: "Store API",
			Docs: &docs,
			Auth: &model.Auth{Mode: model.AuthBearer},
		},
	}

	saver := &fakeCollectionSettingsSaver{}
	p := NewCollectionSettingsPresenter(col, saver)

	changed := 0
	p.OnChanged(func() {
		changed++
	})

	if p.IsFolder() {
		t.Error("expected collection target, not folder")
	}
	if p.TargetName() != "Store API" {
		t.Errorf("TargetName mismatch: %s", p.TargetName())
	}
	if p.ActiveSubTab() != SettingsSubTabAuth {
		t.Errorf("expected default subtab Auth, got %s", p.ActiveSubTab())
	}
	if p.IsDirty() {
		t.Error("expected not dirty initially")
	}
	if p.Docs() != "API Documentation" {
		t.Errorf("expected docs text 'API Documentation', got %q", p.Docs())
	}

	// SubTab selection
	for _, tab := range AllSettingsSubTabs {
		p.SelectSubTab(tab)
		if p.ActiveSubTab() != tab {
			t.Errorf("expected tab %s, got %s", tab, p.ActiveSubTab())
		}
	}

	// Modifications
	p.SetAuth(&model.Auth{Mode: model.AuthBasic})
	if !p.IsDirty() {
		t.Error("expected dirty after SetAuth")
	}
	if p.Auth().Mode != model.AuthBasic {
		t.Errorf("expected auth basic, got %s", p.Auth().Mode)
	}

	nameHdr := "X-Test"
	valHdr := "true"
	p.SetHeaders([]model.KeyValue{{Name: &nameHdr, Value: &valHdr, Enabled: true}})
	if len(p.Headers()) != 1 {
		t.Errorf("expected 1 header, got %d", len(p.Headers()))
	}

	nameVar := "env_val"
	p.SetVars([]model.Variable{{Name: &nameVar, Value: "123", Enabled: true}})
	if len(p.Vars()) != 1 {
		t.Errorf("expected 1 var, got %d", len(p.Vars()))
	}

	reqScpt := "console.log('hi');"
	p.SetScripts(&model.Scripts{Req: &reqScpt})
	if p.Scripts().Req == nil || *p.Scripts().Req != "console.log('hi');" {
		t.Errorf("scripts mismatch: %v", p.Scripts())
	}

	p.SetDocs("Updated Docs")
	if p.Docs() != "Updated Docs" {
		t.Errorf("docs mismatch: %s", p.Docs())
	}

	// Original collection config remains unmutated before Save
	if col.Config.Auth.Mode != model.AuthBearer {
		t.Errorf("original collection mutated before Save: %s", col.Config.Auth.Mode)
	}

	// Revert test
	p.Revert()
	if p.IsDirty() {
		t.Error("expected clean after revert")
	}
	if p.Auth().Mode != model.AuthBearer {
		t.Errorf("revert failed to restore AuthBearer: %s", p.Auth().Mode)
	}
	if p.Docs() != "API Documentation" {
		t.Errorf("revert failed to restore original docs: %s", p.Docs())
	}

	// Edit and Save
	p.SetAuth(&model.Auth{Mode: model.AuthBasic})
	p.SetDocs("New Saved Docs")
	if err := p.Save(context.Background()); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if p.IsDirty() {
		t.Error("expected clean after save")
	}
	if saver.savedColUID != "col-set-1" {
		t.Errorf("saver colUID mismatch: %s", saver.savedColUID)
	}
	if col.Config.Auth.Mode != model.AuthBasic {
		t.Errorf("collection was not updated: %s", col.Config.Auth.Mode)
	}
	if col.Config.Docs == nil || *col.Config.Docs != "New Saved Docs" {
		t.Errorf("collection docs not updated: %v", col.Config.Docs)
	}
}

func TestSettingsPresenter_FolderTarget(t *testing.T) {
	folder := &model.Item{
		UID:  "folder-set-1",
		Name: "Users Module",
		Type: model.ItemFolder,
		Auth: &model.Auth{Mode: model.AuthInherit},
	}

	saver := &fakeFolderSettingsSaver{}
	p := NewFolderSettingsPresenter("col-parent-1", folder, saver)

	if !p.IsFolder() {
		t.Error("expected folder target")
	}
	if p.TargetName() != "Users Module" {
		t.Errorf("expected target name 'Users Module', got %s", p.TargetName())
	}

	p.SetAuth(&model.Auth{Mode: model.AuthAPIKey})
	if !p.IsDirty() {
		t.Error("expected dirty after SetAuth")
	}

	if err := p.Save(context.Background()); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if p.IsDirty() {
		t.Error("expected clean after save")
	}
	if saver.savedColUID != "col-parent-1" || saver.savedFolder != folder {
		t.Errorf("saver args mismatch: %s, %v", saver.savedColUID, saver.savedFolder)
	}
	if folder.Auth.Mode != model.AuthAPIKey {
		t.Errorf("folder auth not updated: %s", folder.Auth.Mode)
	}

	// Target switching
	newCol := &model.Collection{UID: "col-new", Name: "New Col"}
	p.SetTargetCollection(newCol)
	if p.IsFolder() {
		t.Error("expected collection target after SetTargetCollection")
	}
	if p.TargetName() != "New Col" {
		t.Errorf("TargetName mismatch: %s", p.TargetName())
	}

	// Target switching to folder
	p.SetTargetFolder("col-new", folder)
	if !p.IsFolder() {
		t.Error("expected folder target after SetTargetFolder")
	}

	// Saver error handling
	saver.err = errors.New("save failed")
	p.SetAuth(&model.Auth{Mode: model.AuthDigest})
	if err := p.Save(context.Background()); err == nil {
		t.Error("expected save error")
	}
	if !p.IsDirty() {
		t.Error("expected still dirty after failed save")
	}
}
