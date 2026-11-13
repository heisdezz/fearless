// Package environment renders the environment editor: the list of a
// collection's environments and the variable table for the one being edited.
//
// It is deliberately its own screen package rather than a tab of settings:
// environments are edited far more often than collection settings, and the
// editor is reached straight from the Env button and from a `{{variable}}`
// click in the request URL bar.
package environment

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/widgets"
)

// State preserves the transient fields of the trailing "new variable" row
// across frames.
type State struct {
	NewName   string
	NewValue  string
	NewSecret bool

	// Focus is the variable the editor was opened for — a `{{variable}}`
	// double-clicked in the request URL bar. The table highlights it on the next
	// frame, creating it when the environment has no variable of that name yet,
	// and then clears it so the highlight belongs to that navigation only.
	Focus string

	// Back, when set, closes the editor. It is offered only when the editor was
	// opened for a variable, so a user can return to the request they came from
	// once the variable is edited or created.
	Back func()
}

// focusNewVariable gives the focus request its variable: an existing variable of
// that name is already in the table, and any other name is added — with no
// value, so the user fills it in — for the highlight to land on.
func focusNewVariable(st *State, p *presenters.EnvironmentPresenter) {
	name := strings.TrimSpace(st.Focus)
	st.Focus = ""
	if name == "" {
		return
	}
	env := p.EffectiveEnvironment()
	if env == nil {
		return
	}
	for _, v := range env.Variables {
		if v.NameOrEmpty() == name {
			return
		}
	}
	p.SetVariable(name, "", false, true)
}

// Screen renders the environment editor.
func Screen(c *ui.Context, p *presenters.EnvironmentPresenter, st *State) ui.Element {
	tk := lobetheme.TokensOf(c)
	if st == nil {
		st = &State{}
	}

	if p == nil || p.Collection() == nil {
		return empty(c, tk, "No collection selected", "Open a collection to edit its environments.")
	}

	return ui.Column(c).Fill().Gap(12).Padding(16).Children(func() {
		header(c, tk, p, st)
		ui.Row(c).Fill().Grow(1).Gap(12).Children(func() {
			list(c, tk, p)
			ui.Box(c).Grow(1).Children(func() {
				table(c, tk, p, st)
			})
		})
	})
}

// header is the title row and its save/revert controls.
func header(c *ui.Context, tk *lobetheme.Tokens, p *presenters.EnvironmentPresenter, st *State) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
		if st.Back != nil {
			widgets.Button(c, "Back", widgets.ButtonProps{Variant: action.Text, Size: lobetheme.Small}).
				OnClick(st.Back)
		}
		ui.Text(c, "Environments").FontSize(15).FontWeight(600).TextColor(tk.Text)
		ui.Box(c).Grow(1)
		if p.IsDirty() {
			ui.Text(c, "Unsaved changes").FontSize(11).TextColor(tk.WarningText)
			widgets.Button(c, "Revert", widgets.ButtonProps{Size: lobetheme.Small}).OnClick(p.Revert)
		}
		widgets.Button(c, "Save", widgets.ButtonProps{Variant: action.Primary, Size: lobetheme.Small}).
			OnClick(func() { _ = p.Save(context.Background()) })
	})
}

// list shows every environment, the active one marked, with create/delete.
func list(c *ui.Context, tk *lobetheme.Tokens, p *presenters.EnvironmentPresenter) {
	envs := p.Environments()
	active := p.ActiveEnvironmentName()
	editing := p.EditingEnvironmentName()

	// A create/delete request is recorded during the walk and applied after,
	// so presenters are not mutated mid-layout.
	var toDelete string
	var createNew bool

	ui.Column(c).Width(220).FillHeight().Padding(8).Radius(tk.BorderRadius).
		Background(tk.BgBase).Border(1, tk.Border).Gap(2).Children(func() {
		if len(envs) == 0 {
			ui.Text(c, "No environments yet").FontSize(12).TextColor(tk.TextTertiary).Padding(6)
		}
		for _, env := range envs {
			e := env
			row := ui.Row(c).FillWidth().Padding(6, 8).Radius(tk.BorderRadiusSM).
				AlignItems(ui.Center).Gap(6)
			if e.Name == editing {
				row.Background(tk.PrimaryFillQuaternary)
			}
			row.OnClick(func() { p.SetEditingEnvironment(e.Name) })
			row.Children(func() {
				ui.Text(c, e.Name).Grow(1).FontSize(12).TextColor(tk.Text)
				if e.Name == active {
					ui.Text(c, "active").FontSize(10).TextColor(tk.SuccessText)
				}
				widgets.Button(c, "x", widgets.ButtonProps{Variant: action.Text, Size: lobetheme.Small}).
					OnClick(func() { toDelete = e.Name })
			})
		}
		ui.Box(c).Height(4)
		widgets.Button(c, "+ New environment", widgets.ButtonProps{Size: lobetheme.Small}).
			OnClick(func() { createNew = true })
	})

	if toDelete != "" {
		_ = p.DeleteEnvironment(context.Background(), toDelete)
	}
	if createNew {
		_, _ = p.CreateEnvironment(context.Background(), presenters.NextEnvironmentName(envs))
	}
}

// table renders the variables of the environment being edited.
func table(c *ui.Context, tk *lobetheme.Tokens, p *presenters.EnvironmentPresenter, st *State) {
	focusNewVariable(st, p)

	env := p.EffectiveEnvironment()
	if env == nil {
		empty(c, tk, "No environment selected", "Choose an environment to edit its variables, or create one.").
			AlignSelf(ui.Stretch)
		return
	}

	var toDelete string

	ui.Column(c).FillHeight().Padding(12).Radius(tk.BorderRadius).
		Background(tk.BgBase).Border(1, tk.Border).Gap(6).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, env.Name).Grow(1).FontSize(13).FontWeight(600).TextColor(tk.Text)
			widgets.Button(c, "Use this environment", widgets.ButtonProps{Size: lobetheme.Small}).
				OnClick(func() { _ = p.SelectEnvironment(context.Background(), env.Name) })
		})

		ui.Row(c).FillWidth().Gap(8).Padding(2, 4).Children(func() {
			ui.Text(c, "NAME").Width(180).FontSize(10).FontWeight(600).TextColor(tk.TextTertiary)
			ui.Text(c, "VALUE").Grow(1).FontSize(10).FontWeight(600).TextColor(tk.TextTertiary)
			ui.Text(c, "SECRET").Width(56).FontSize(10).FontWeight(600).TextColor(tk.TextTertiary)
			ui.Box(c).Width(32)
		})

		for i := range env.Variables {
			v := env.Variables[i]
			row(c, tk, p, v, v.NameOrEmpty() == st.Focus, &toDelete)
		}
		addRow(c, p, st)
	})

	if toDelete != "" {
		p.DeleteVariable(toDelete)
	}
}

// row is one variable. Edits write into the draft, so Save commits them. The
// row is marked when it is the variable the editor was opened for.
func row(c *ui.Context, tk *lobetheme.Tokens, p *presenters.EnvironmentPresenter, v model.Variable, marked bool, toDelete *string) {
	name := v.NameOrEmpty()
	shown := p.FormattedVariableValue(v)

	r := ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Padding(2, 4)
	if marked {
		r.Background(tk.PrimaryFillQuaternary)
	}
	r.Children(func() {
		n := name
		nameInput := widgets.FieldInput(c, &n, widgets.FieldInputProps{Placeholder: "name"})
		nameInput.Width(180)
		if n != name && strings.TrimSpace(n) != "" {
			// Rename by adding the new name and dropping the old.
			p.SetVariable(n, v.Value, v.Secret, v.Enabled)
			p.DeleteVariable(name)
		}

		// A secret's value shows as '*' until revealed; do not write the
		// mask back over the real value.
		if !p.IsSecretMasked(v) {
			val := shown
			valueInput := widgets.FieldInput(c, &val, widgets.FieldInputProps{Placeholder: "value"})
			valueInput.Grow(1)
			if val != shown {
				p.SetVariable(name, val, v.Secret, v.Enabled)
			}
		} else {
			masked := shown
			maskedInput := widgets.FieldInput(c, &masked, widgets.FieldInputProps{Placeholder: "value"})
			maskedInput.Grow(1)
			_ = masked
		}

		if v.Secret {
			widgets.Button(c, "reveal", widgets.ButtonProps{Variant: action.Text, Size: lobetheme.Small}).
				Width(56).OnClick(func() { p.ToggleSecretMask(name) })
		} else {
			widgets.Button(c, "secret", widgets.ButtonProps{Variant: action.Text, Size: lobetheme.Small}).
				Width(56).OnClick(func() { p.SetVariable(name, v.Value, true, v.Enabled) })
		}

		widgets.Button(c, "x", widgets.ButtonProps{Variant: action.Text, Size: lobetheme.Small}).
			Width(32).OnClick(func() { *toDelete = name })
	})
}

// addRow is the trailing row: typing a name and pressing Add creates the
// variable.
func addRow(c *ui.Context, p *presenters.EnvironmentPresenter, st *State) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Padding(2, 4).Children(func() {
		nameInput := widgets.FieldInput(c, &st.NewName, widgets.FieldInputProps{Placeholder: "new variable"})
		nameInput.Width(180)
		valueInput := widgets.FieldInput(c, &st.NewValue, widgets.FieldInputProps{Placeholder: "value"})
		valueInput.Grow(1)

		variant := action.Text
		if st.NewSecret {
			variant = action.Primary
		}
		widgets.Button(c, "secret", widgets.ButtonProps{Variant: variant, Size: lobetheme.Small}).
			Width(56).OnClick(func() { st.NewSecret = !st.NewSecret })

		add := widgets.Button(c, "Add", widgets.ButtonProps{Size: lobetheme.Small}).Width(32)
		if strings.TrimSpace(st.NewName) == "" {
			add.Disabled(true)
		} else {
			add.OnClick(func() {
				p.SetVariable(strings.TrimSpace(st.NewName), st.NewValue, st.NewSecret, true)
				st.NewName, st.NewValue, st.NewSecret = "", "", false
			})
		}
	})
}

// empty is the shared placeholder for the "nothing to edit" states.
func empty(c *ui.Context, tk *lobetheme.Tokens, title, desc string) ui.Element {
	return ui.Column(c).Fill().Gap(6).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(tk.TextSecondary)
		ui.Text(c, desc).FontSize(12).TextColor(tk.TextTertiary)
	})
}
