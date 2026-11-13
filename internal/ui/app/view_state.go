package app

import (
	"context"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
)

// resolvesVariable reports whether name is defined for the active collection:
// in the active environment, or in the collection's own variables.
//
// It backs the URL bar's `{{variable}}` colouring, so an undefined name reads
// as a mistake before the request is sent.
func (a *App) resolvesVariable(name string) bool {
	if name == "" {
		return false
	}
	if a.EnvironmentPresenter != nil {
		if _, ok := a.EnvironmentPresenter.VariableValue(name); ok {
			return true
		}
	}
	return false
}

// ShowEnvironment displays the environment editor for the active collection.
func (a *App) ShowEnvironment() {
	a.SafeUpdate(func() {
		if a.environmentState != nil {
			a.environmentState.Focus, a.environmentState.Back = "", nil
		}
		a.ActiveView = ViewKindEnvironment
	})
}

// EditVariable opens the environment editor on one variable, for a `{{variable}}`
// clicked in the request URL bar.
//
// A name the environment does not have yet is created for the user to fill in,
// so one gesture covers both editing a variable and making a missing one. The
// editor offers Back, so the user returns to the request they came from.
func (a *App) EditVariable(name string) {
	a.SafeUpdate(func() {
		p := a.EnvironmentPresenter
		if p == nil || p.Collection() == nil {
			return
		}
		// The variable is defined somewhere else — a collection or folder
		// variable, say — so there is nothing for the environment table to
		// create or highlight.
		if a.resolvesVariable(name) && !environmentDefines(p.EffectiveEnvironment(), name) {
			return
		}
		env := p.EffectiveEnvironment()
		if env == nil {
			// No environment is being edited: edit the one in use, or make one.
			env = p.ActiveEnvironment()
			if env == nil {
				created, err := p.CreateEnvironment(context.Background(), presenters.NextEnvironmentName(p.Environments()))
				if err != nil || created == nil {
					return
				}
				env = created
			}
			p.SetEditingEnvironment(env.Name)
		}
		if a.environmentState != nil {
			a.environmentState.Focus = name
			a.environmentState.Back = a.ShowMain
		}
		a.ActiveView = ViewKindEnvironment
	})
}

// environmentDefines reports whether env itself has a variable called name.
func environmentDefines(env *model.Environment, name string) bool {
	if env == nil {
		return false
	}
	for _, v := range env.Variables {
		if v.NameOrEmpty() == name {
			return true
		}
	}
	return false
}

// ShowPreferences displays the application preferences screen.
func (a *App) ShowPreferences() {
	a.SafeUpdate(func() {
		a.ActiveView = ViewKindPreferences
	})
}

// ShowCollectionSettings displays the collection settings screen.
func (a *App) ShowCollectionSettings() {
	a.SafeUpdate(func() {
		a.ActiveView = ViewKindCollectionSettings
	})
}

// ShowMain switches back to the main request/workspace view.
func (a *App) ShowMain() {
	a.SafeUpdate(func() {
		a.ActiveView = ViewKindMain
	})
}

// TogglePreferences toggles between preferences and main view.
func (a *App) TogglePreferences() {
	a.SafeUpdate(func() {
		if a.ActiveView == ViewKindPreferences {
			a.ActiveView = ViewKindMain
		} else {
			a.ActiveView = ViewKindPreferences
		}
	})
}
