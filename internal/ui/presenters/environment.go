package presenters

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"fearless/internal/core/model"
)

// EnvSelectorPort is the consumer port to select an active environment.
type EnvSelectorPort interface {
	SelectEnvironment(ctx context.Context, colUID string, envName string) error
}

// EnvSaverPort is the consumer port to persist modifications to an environment.
type EnvSaverPort interface {
	SaveEnvironment(ctx context.Context, colUID string, env *model.Environment) error
}

// EnvCreatorPort is the consumer port to create a new environment.
type EnvCreatorPort interface {
	CreateEnvironment(ctx context.Context, colUID string, name string) (*model.Environment, error)
}

// EnvDeleterPort is the consumer port to delete an environment.
type EnvDeleterPort interface {
	DeleteEnvironment(ctx context.Context, colUID string, envName string) error
}

// EnvironmentPresenter coordinates environment selection and variable management view-models.
type EnvironmentPresenter struct {
	mu              sync.RWMutex
	collection      *model.Collection
	editingEnvName  string
	draftEnv        *model.Environment
	revealedSecrets map[string]bool
	selector        EnvSelectorPort
	saver           EnvSaverPort
	creator         EnvCreatorPort
	deleter         EnvDeleterPort
	onChanged       func()
}

// NewEnvironmentPresenter creates an initialized EnvironmentPresenter.
func NewEnvironmentPresenter(
	selector EnvSelectorPort,
	saver EnvSaverPort,
	creator EnvCreatorPort,
	deleter EnvDeleterPort,
) *EnvironmentPresenter {
	return &EnvironmentPresenter{
		revealedSecrets: make(map[string]bool),
		selector:        selector,
		saver:           saver,
		creator:         creator,
		deleter:         deleter,
	}
}

// SetCollection binds the active collection to the environment presenter.
func (p *EnvironmentPresenter) SetCollection(col *model.Collection) {
	p.mu.Lock()
	p.collection = col
	p.draftEnv = nil
	p.editingEnvName = ""
	if col != nil && col.ActiveEnvironment != "" {
		p.editingEnvName = col.ActiveEnvironment
	}
	p.mu.Unlock()
	p.notifyChanged()
}

// Collection returns the currently bound collection.
func (p *EnvironmentPresenter) Collection() *model.Collection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.collection
}

// Environments returns all environments in the current collection.
func (p *EnvironmentPresenter) Environments() []*model.Environment {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.collection == nil {
		return nil
	}
	out := make([]*model.Environment, len(p.collection.Environments))
	copy(out, p.collection.Environments)
	return out
}

// ActiveEnvironmentName returns the name of the active environment, or empty.
func (p *EnvironmentPresenter) ActiveEnvironmentName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.collection == nil {
		return ""
	}
	return p.collection.ActiveEnvironment
}

// ActiveEnvironment returns the active model.Environment, or nil.
func (p *EnvironmentPresenter) ActiveEnvironment() *model.Environment {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.collection == nil || p.collection.ActiveEnvironment == "" {
		return nil
	}
	for _, env := range p.collection.Environments {
		if env.Name == p.collection.ActiveEnvironment {
			return env
		}
	}
	return nil
}

// EditingEnvironmentName returns the name of the environment being edited.
func (p *EnvironmentPresenter) EditingEnvironmentName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.editingEnvName
}

// SetEditingEnvironment sets which environment is open for variable editing.
func (p *EnvironmentPresenter) SetEditingEnvironment(name string) {
	p.mu.Lock()
	p.editingEnvName = name
	p.draftEnv = nil
	p.mu.Unlock()
	p.notifyChanged()
}

// EffectiveEnvironment returns the draft if modified, otherwise the saved environment.
func (p *EnvironmentPresenter) EffectiveEnvironment() *model.Environment {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.draftEnv != nil {
		return p.draftEnv
	}
	return p.findSavedEnvironment(p.editingEnvName)
}

func (p *EnvironmentPresenter) findSavedEnvironment(name string) *model.Environment {
	if p.collection == nil || name == "" {
		return nil
	}
	for _, env := range p.collection.Environments {
		if env.Name == name {
			return env
		}
	}
	return nil
}

// SelectEnvironment dispatches active environment selection.
func (p *EnvironmentPresenter) SelectEnvironment(ctx context.Context, envName string) error {
	p.mu.Lock()
	col := p.collection
	selector := p.selector
	p.mu.Unlock()

	if col == nil || selector == nil {
		return nil
	}

	err := selector.SelectEnvironment(ctx, col.UID, envName)
	if err == nil {
		p.mu.Lock()
		col.ActiveEnvironment = envName
		if p.editingEnvName == "" {
			p.editingEnvName = envName
		}
		p.mu.Unlock()
	}
	p.notifyChanged()
	return err
}

// CreateEnvironment creates a new environment.
func (p *EnvironmentPresenter) CreateEnvironment(ctx context.Context, name string) (*model.Environment, error) {
	p.mu.Lock()
	col := p.collection
	creator := p.creator
	p.mu.Unlock()

	if col == nil || creator == nil {
		return nil, nil
	}

	env, err := creator.CreateEnvironment(ctx, col.UID, name)
	if err == nil && env != nil {
		p.mu.Lock()
		col.Environments = append(col.Environments, env)
		p.editingEnvName = env.Name
		p.draftEnv = nil
		p.mu.Unlock()
	}
	p.notifyChanged()
	return env, err
}

// DeleteEnvironment deletes an environment.
func (p *EnvironmentPresenter) DeleteEnvironment(ctx context.Context, name string) error {
	p.mu.Lock()
	col := p.collection
	deleter := p.deleter
	p.mu.Unlock()

	if col == nil || deleter == nil {
		return nil
	}

	err := deleter.DeleteEnvironment(ctx, col.UID, name)
	if err == nil {
		p.mu.Lock()
		var remaining []*model.Environment
		for _, e := range col.Environments {
			if e.Name != name {
				remaining = append(remaining, e)
			}
		}
		col.Environments = remaining
		if col.ActiveEnvironment == name {
			col.ActiveEnvironment = ""
		}
		if p.editingEnvName == name {
			p.editingEnvName = ""
			p.draftEnv = nil
		}
		p.mu.Unlock()
	}
	p.notifyChanged()
	return err
}

// IsDirty reports whether the editing environment has uncommitted variable changes.
func (p *EnvironmentPresenter) IsDirty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.draftEnv != nil
}

func (p *EnvironmentPresenter) ensureDraft() *model.Environment {
	if p.draftEnv != nil {
		return p.draftEnv
	}
	saved := p.findSavedEnvironment(p.editingEnvName)
	if saved == nil {
		return nil
	}
	p.draftEnv = cloneEnvironment(saved)
	return p.draftEnv
}

func cloneEnvironment(e *model.Environment) *model.Environment {
	if e == nil {
		return nil
	}
	cloned := &model.Environment{
		UID:      e.UID,
		Name:     e.Name,
		Color:    e.Color,
		Extends:  e.Extends,
		Format:   e.Format,
		Filename: e.Filename,
	}
	if len(e.Secrets) > 0 {
		cloned.Secrets = make([]string, len(e.Secrets))
		copy(cloned.Secrets, e.Secrets)
	}
	if len(e.Variables) > 0 {
		cloned.Variables = make([]model.Variable, len(e.Variables))
		for i, v := range e.Variables {
			var namePtr *string
			if v.Name != nil {
				nameStr := *v.Name
				namePtr = &nameStr
			}
			cloned.Variables[i] = model.Variable{
				UID:      v.UID,
				Name:     namePtr,
				Value:    v.Value,
				Enabled:  v.Enabled,
				Secret:   v.Secret,
				Local:    v.Local,
				DataType: v.DataType,
			}
		}
	}
	return cloned
}

// SetVariable adds or updates a variable in the editing environment draft.
func (p *EnvironmentPresenter) SetVariable(name string, value any, secret bool, enabled bool) {
	p.mu.Lock()
	draft := p.ensureDraft()
	if draft == nil {
		p.mu.Unlock()
		return
	}

	found := false
	for i, v := range draft.Variables {
		if v.NameOrEmpty() == name {
			draft.Variables[i].Value = value
			draft.Variables[i].Secret = secret
			draft.Variables[i].Enabled = enabled
			found = true
			break
		}
	}
	if !found {
		nameStr := name
		draft.Variables = append(draft.Variables, model.Variable{
			Name:    &nameStr,
			Value:   value,
			Secret:  secret,
			Enabled: enabled,
		})
	}

	// Update secrets list in environment
	updateSecretsList(draft, name, secret)
	p.mu.Unlock()
	p.notifyChanged()
}

func updateSecretsList(env *model.Environment, name string, secret bool) {
	if secret {
		for _, s := range env.Secrets {
			if s == name {
				return
			}
		}
		env.Secrets = append(env.Secrets, name)
	} else {
		var remaining []string
		for _, s := range env.Secrets {
			if s != name {
				remaining = append(remaining, s)
			}
		}
		env.Secrets = remaining
	}
}

// DeleteVariable removes a variable from the editing environment draft.
func (p *EnvironmentPresenter) DeleteVariable(name string) {
	p.mu.Lock()
	draft := p.ensureDraft()
	if draft == nil {
		p.mu.Unlock()
		return
	}

	var remaining []model.Variable
	for _, v := range draft.Variables {
		if v.NameOrEmpty() != name {
			remaining = append(remaining, v)
		}
	}
	draft.Variables = remaining
	updateSecretsList(draft, name, false)
	delete(p.revealedSecrets, name)

	p.mu.Unlock()
	p.notifyChanged()
}

// Save commits draft modifications to the environment.
func (p *EnvironmentPresenter) Save(ctx context.Context) error {
	p.mu.Lock()
	if p.draftEnv == nil || p.collection == nil || p.saver == nil {
		p.mu.Unlock()
		return nil
	}

	colUID := p.collection.UID
	draft := p.draftEnv
	saver := p.saver
	p.mu.Unlock()

	err := saver.SaveEnvironment(ctx, colUID, draft)
	if err == nil {
		p.mu.Lock()
		for i, env := range p.collection.Environments {
			if env.Name == draft.Name {
				p.collection.Environments[i] = draft
				break
			}
		}
		p.draftEnv = nil
		p.mu.Unlock()
	}
	p.notifyChanged()
	return err
}

// Revert discards draft modifications.
func (p *EnvironmentPresenter) Revert() {
	p.mu.Lock()
	p.draftEnv = nil
	p.mu.Unlock()
	p.notifyChanged()
}

// ToggleSecretMask toggles masking/revealing of a secret variable value.
func (p *EnvironmentPresenter) ToggleSecretMask(varName string) {
	p.mu.Lock()
	p.revealedSecrets[varName] = !p.revealedSecrets[varName]
	p.mu.Unlock()
	p.notifyChanged()
}

// IsSecretMasked reports whether a variable's value is currently hidden.
func (p *EnvironmentPresenter) IsSecretMasked(v model.Variable) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !v.Secret {
		return false
	}
	name := v.NameOrEmpty()
	return !p.revealedSecrets[name]
}

// FormattedVariableValue returns the value string or masked with '*' per character (preserving newlines) if masked (Decision R-5).
func (p *EnvironmentPresenter) FormattedVariableValue(v model.Variable) string {
	if v.Value == nil {
		return ""
	}
	valStr := fmt.Sprintf("%v", v.Value)
	if p.IsSecretMasked(v) {
		return MaskSecret(valStr)
	}
	return valStr
}

// MaskSecret masks a secret value with literal '*' per character, preserving newlines (Decision R-5).
func MaskSecret(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\r' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('*')
		}
	}
	return sb.String()
}

// OnChanged registers an invalidation callback hook.
func (p *EnvironmentPresenter) OnChanged(fn func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onChanged = fn
}

func (p *EnvironmentPresenter) notifyChanged() {
	p.mu.RLock()
	fn := p.onChanged
	p.mu.RUnlock()
	if fn != nil {
		fn()
	}
}
