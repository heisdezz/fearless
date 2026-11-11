package presenters

import (
	"context"
	"errors"
	"testing"

	"fearless/internal/core/model"
)

type fakeEnvSelector struct {
	selectedColUID string
	selectedName   string
	err            error
}

func (f *fakeEnvSelector) SelectEnvironment(ctx context.Context, colUID string, envName string) error {
	if f.err != nil {
		return f.err
	}
	f.selectedColUID = colUID
	f.selectedName = envName
	return nil
}

type fakeEnvSaver struct {
	savedColUID string
	savedEnv    *model.Environment
	err         error
}

func (f *fakeEnvSaver) SaveEnvironment(ctx context.Context, colUID string, env *model.Environment) error {
	if f.err != nil {
		return f.err
	}
	f.savedColUID = colUID
	f.savedEnv = env
	return nil
}

type fakeEnvCreator struct {
	createdColUID string
	createdName   string
	err           error
}

func (f *fakeEnvCreator) CreateEnvironment(ctx context.Context, colUID string, name string) (*model.Environment, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.createdColUID = colUID
	f.createdName = name
	return &model.Environment{
		UID:  "new-env-uid",
		Name: name,
	}, nil
}

type fakeEnvDeleter struct {
	deletedColUID string
	deletedName   string
	err           error
}

func (f *fakeEnvDeleter) DeleteEnvironment(ctx context.Context, colUID string, envName string) error {
	if f.err != nil {
		return f.err
	}
	f.deletedColUID = colUID
	f.deletedName = envName
	return nil
}

func buildTestEnvCollection() *model.Collection {
	valHost := "https://api.dev.local"
	nameHost := "host"
	valKey := "super-secret-key"
	nameKey := "apiKey"

	devEnv := &model.Environment{
		UID:  "env-dev",
		Name: "Development",
		Variables: []model.Variable{
			{
				Name:    &nameHost,
				Value:   valHost,
				Enabled: true,
			},
			{
				Name:    &nameKey,
				Value:   valKey,
				Secret:  true,
				Enabled: true,
			},
		},
		Secrets: []string{"apiKey"},
	}

	prodEnv := &model.Environment{
		UID:  "env-prod",
		Name: "Production",
	}

	return &model.Collection{
		UID:               "col-env-1",
		Name:              "API Project",
		ActiveEnvironment: "Development",
		Environments:      []*model.Environment{devEnv, prodEnv},
	}
}

func TestEnvironmentPresenter_BasicsAndSelection(t *testing.T) {
	selector := &fakeEnvSelector{}
	saver := &fakeEnvSaver{}
	creator := &fakeEnvCreator{}
	deleter := &fakeEnvDeleter{}

	p := NewEnvironmentPresenter(selector, saver, creator, deleter)

	if p.Collection() != nil {
		t.Error("expected nil collection initially")
	}
	if p.ActiveEnvironment() != nil {
		t.Error("expected nil active environment initially")
	}
	if len(p.Environments()) != 0 {
		t.Error("expected empty environments initially")
	}

	col := buildTestEnvCollection()
	p.SetCollection(col)

	if p.Collection() != col {
		t.Error("collection mismatch")
	}
	if len(p.Environments()) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(p.Environments()))
	}
	if p.ActiveEnvironmentName() != "Development" {
		t.Errorf("expected active env 'Development', got %s", p.ActiveEnvironmentName())
	}
	if p.ActiveEnvironment() == nil || p.ActiveEnvironment().UID != "env-dev" {
		t.Errorf("active environment mismatch: %v", p.ActiveEnvironment())
	}
	if p.EditingEnvironmentName() != "Development" {
		t.Errorf("editing environment should default to active: %s", p.EditingEnvironmentName())
	}

	// SelectEnvironment
	if err := p.SelectEnvironment(context.Background(), "Production"); err != nil {
		t.Fatalf("SelectEnvironment failed: %v", err)
	}
	if selector.selectedName != "Production" || selector.selectedColUID != "col-env-1" {
		t.Errorf("selector args mismatch: %s, %s", selector.selectedName, selector.selectedColUID)
	}
	if p.ActiveEnvironmentName() != "Production" {
		t.Errorf("active env was not updated to Production: %s", p.ActiveEnvironmentName())
	}

	// Selection error
	selector.err = errors.New("selector error")
	if err := p.SelectEnvironment(context.Background(), "Development"); err == nil {
		t.Error("expected error from selector")
	}
}

func TestEnvironmentPresenter_EditingAndDirty(t *testing.T) {
	saver := &fakeEnvSaver{}
	p := NewEnvironmentPresenter(nil, saver, nil, nil)
	col := buildTestEnvCollection()
	p.SetCollection(col)

	p.SetEditingEnvironment("Development")
	if p.IsDirty() {
		t.Error("presenter should not be dirty before edits")
	}

	// Update existing variable
	p.SetVariable("host", "https://api.test.local", false, true)
	if !p.IsDirty() {
		t.Error("presenter should be dirty after SetVariable")
	}
	effective := p.EffectiveEnvironment()
	val, ok := effective.VariableValue("host")
	if !ok || val != "https://api.test.local" {
		t.Errorf("effective variable mismatch: %v", val)
	}

	// Original saved environment remains unchanged before Save
	savedVal, _ := col.Environments[0].VariableValue("host")
	if savedVal != "https://api.dev.local" {
		t.Errorf("saved environment mutated before Save: %v", savedVal)
	}

	// Add new secret variable
	p.SetVariable("token", "xyz-123", true, true)
	effective = p.EffectiveEnvironment()
	tokenVal, ok := effective.VariableValue("token")
	if !ok || tokenVal != "xyz-123" {
		t.Errorf("token variable not found: %v", tokenVal)
	}

	// Revert test
	p.Revert()
	if p.IsDirty() {
		t.Error("presenter should not be dirty after revert")
	}
	effective = p.EffectiveEnvironment()
	if _, ok := effective.VariableValue("token"); ok {
		t.Error("revert failed to discard token variable")
	}

	// Make edit and Save
	p.SetVariable("host", "https://api.test.local", false, true)
	if err := p.Save(context.Background()); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if p.IsDirty() {
		t.Error("presenter should not be dirty after save")
	}
	if saver.savedEnv == nil || saver.savedColUID != "col-env-1" {
		t.Errorf("saver not called correctly: %v", saver.savedEnv)
	}
	savedVal, _ = col.Environments[0].VariableValue("host")
	if savedVal != "https://api.test.local" {
		t.Errorf("saved environment was not updated after save: %v", savedVal)
	}

	// DeleteVariable
	p.DeleteVariable("host")
	if !p.IsDirty() {
		t.Error("presenter should be dirty after DeleteVariable")
	}
	if _, ok := p.EffectiveEnvironment().VariableValue("host"); ok {
		t.Error("variable was not removed in draft")
	}
}

func TestEnvironmentPresenter_SecretMasking(t *testing.T) {
	p := NewEnvironmentPresenter(nil, nil, nil, nil)
	col := buildTestEnvCollection()
	p.SetCollection(col)

	namePlain := "plain"
	plainVar := model.Variable{
		Name:   &namePlain,
		Value:  "public-value",
		Secret: false,
	}

	nameSecret := "apiKey"
	secretVar := model.Variable{
		Name:   &nameSecret,
		Value:  "secret-12345",
		Secret: true,
	}

	if p.IsSecretMasked(plainVar) {
		t.Error("plain variable should not be masked")
	}
	if p.FormattedVariableValue(plainVar) != "public-value" {
		t.Errorf("plain value mismatch: %s", p.FormattedVariableValue(plainVar))
	}

	// Secret variable: masked by default
	if !p.IsSecretMasked(secretVar) {
		t.Error("secret variable should be masked by default")
	}
	if p.FormattedVariableValue(secretVar) != "••••••••" {
		t.Errorf("masked value mismatch: %s", p.FormattedVariableValue(secretVar))
	}

	// Toggle unmask
	p.ToggleSecretMask("apiKey")
	if p.IsSecretMasked(secretVar) {
		t.Error("secret variable should be revealed after toggle")
	}
	if p.FormattedVariableValue(secretVar) != "secret-12345" {
		t.Errorf("revealed value mismatch: %s", p.FormattedVariableValue(secretVar))
	}

	// Toggle mask again
	p.ToggleSecretMask("apiKey")
	if !p.IsSecretMasked(secretVar) {
		t.Error("secret variable should be masked after second toggle")
	}

	// Nil value formatted
	nilVar := model.Variable{Name: &namePlain, Value: nil}
	if p.FormattedVariableValue(nilVar) != "" {
		t.Errorf("expected empty string for nil value")
	}
}

func TestEnvironmentPresenter_CreateAndDelete(t *testing.T) {
	creator := &fakeEnvCreator{}
	deleter := &fakeEnvDeleter{}
	p := NewEnvironmentPresenter(nil, nil, creator, deleter)
	col := buildTestEnvCollection()
	p.SetCollection(col)

	// Create
	newEnv, err := p.CreateEnvironment(context.Background(), "Staging")
	if err != nil || newEnv == nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}
	if creator.createdName != "Staging" || creator.createdColUID != "col-env-1" {
		t.Errorf("creator args mismatch: %s, %s", creator.createdName, creator.createdColUID)
	}
	if len(p.Environments()) != 3 {
		t.Fatalf("expected 3 environments after create, got %d", len(p.Environments()))
	}
	if p.EditingEnvironmentName() != "Staging" {
		t.Errorf("editing environment should be Staging, got %s", p.EditingEnvironmentName())
	}

	// Delete
	if err := p.DeleteEnvironment(context.Background(), "Production"); err != nil {
		t.Fatalf("DeleteEnvironment failed: %v", err)
	}
	if deleter.deletedName != "Production" || deleter.deletedColUID != "col-env-1" {
		t.Errorf("deleter args mismatch: %s, %s", deleter.deletedName, deleter.deletedColUID)
	}
	if len(p.Environments()) != 2 {
		t.Fatalf("expected 2 environments after delete, got %d", len(p.Environments()))
	}

	// Error paths
	creator.err = errors.New("create error")
	if _, err := p.CreateEnvironment(context.Background(), "ErrorEnv"); err == nil {
		t.Error("expected error from creator")
	}

	deleter.err = errors.New("delete error")
	if err := p.DeleteEnvironment(context.Background(), "Development"); err == nil {
		t.Error("expected error from deleter")
	}
}
