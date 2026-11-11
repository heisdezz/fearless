package settings

import (
	"context"
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/nav"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/widgets"
)

var collectionSubTabItems = []struct {
	tab   presenters.SettingsSubTab
	label string
}{
	{presenters.SettingsSubTabAuth, "Auth"},
	{presenters.SettingsSubTabHeaders, "Headers"},
	{presenters.SettingsSubTabVars, "Vars"},
	{presenters.SettingsSubTabScript, "Script"},
	{presenters.SettingsSubTabDocs, "Docs"},
}

// CollectionSettingsState preserves editor state between frames.
type CollectionSettingsState struct {
	ScriptEditor *editor.State
	DocsEditor   *editor.State
}

// CollectionSettingsScreen renders the collection or folder settings pane.
func CollectionSettingsScreen(
	c *ui.Context,
	p *presenters.SettingsPresenter,
	state ...*CollectionSettingsState,
) ui.Element {
	tk := lobetheme.TokensOf(c)

	if p == nil {
		return ui.Box(c)
	}

	var st *CollectionSettingsState
	if len(state) > 0 && state[0] != nil {
		st = state[0]
	} else {
		st = &CollectionSettingsState{}
	}

	targetName := p.TargetName()
	if targetName == "" {
		targetName = "Collection Settings"
	}
	activeTab := p.ActiveSubTab()

	return ui.Column(c).Fill().Padding(16, 20).Gap(14).Children(func() {
		// Header title
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, targetName).
				FontSize(16).
				Bold().
				TextColor(tk.Text)
		})

		// Sub-Tabs navigation
		selectedIdx := 0
		var navItems []nav.TabItem
		for i, item := range collectionSubTabItems {
			navItems = append(navItems, nav.TabItem{Label: item.label})
			if item.tab == activeTab {
				selectedIdx = i
			}
		}

		nav.Tabs(c, &selectedIdx, nav.TabsProps{
			Items:   navItems,
			Variant: nav.TabsSquare,
			Size:    lobetheme.Small,
		}, nil)

		if selectedIdx >= 0 && selectedIdx < len(collectionSubTabItems) {
			newTab := collectionSubTabItems[selectedIdx].tab
			if newTab != activeTab {
				p.SelectSubTab(newTab)
				activeTab = newTab
			}
		}

		// Content Panel
		ui.Box(c).Fill().Padding(10, 0).Children(func() {
			switch activeTab {
			case presenters.SettingsSubTabAuth:
				renderCollectionAuth(c, p)
			case presenters.SettingsSubTabHeaders:
				renderCollectionHeaders(c, p)
			case presenters.SettingsSubTabVars:
				renderCollectionVars(c, p)
			case presenters.SettingsSubTabScript:
				renderCollectionScript(c, p, &st.ScriptEditor)
			case presenters.SettingsSubTabDocs:
				renderCollectionDocs(c, p, &st.DocsEditor)
			default:
				renderCollectionAuth(c, p)
			}
		})

		// Bottom Action Bar: Save and Revert
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Box(c).Grow(1)

			saveLabel := "Save"
			if p.IsDirty() {
				saveLabel = "Save *"
			}

			action.Button(c, saveLabel, action.ButtonProps{
				Variant: action.Primary,
				Size:    lobetheme.Small,
			}).OnClick(func() {
				p.Save(context.Background())
			})

			action.Button(c, "Revert", action.ButtonProps{
				Variant: action.Default,
				Size:    lobetheme.Small,
			}).OnClick(func() {
				p.Revert()
			})
		})
	})
}

var collectionAuthOptions = []field.SelectOption{
	{Value: string(model.AuthNone), Label: "No Auth"},
	{Value: string(model.AuthBasic), Label: "Basic Auth"},
	{Value: string(model.AuthBearer), Label: "Bearer Token"},
	{Value: string(model.AuthAPIKey), Label: "API Key"},
	{Value: string(model.AuthDigest), Label: "Digest Auth"},
	{Value: string(model.AuthOAuth2), Label: "OAuth 2.0"},
}

func renderCollectionAuth(c *ui.Context, p *presenters.SettingsPresenter) {
	tk := lobetheme.TokensOf(c)

	auth := p.Auth()
	if auth == nil {
		auth = &model.Auth{Mode: model.AuthNone}
	}

	modeStr := string(auth.Mode)
	if modeStr == "" {
		modeStr = string(model.AuthNone)
	}

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Text(c, "Default Authentication:").FontSize(11).Bold().TextColor(tk.TextSecondary)

			field.Select(c, &modeStr, field.SelectProps{
				Options: collectionAuthOptions,
				Size:    lobetheme.Small,
			})
			auth.Mode = model.AuthMode(modeStr)
		})

		switch auth.Mode {
		case model.AuthNone:
			ui.Text(c, "Requests without explicit authentication will have no auth applied.").
				FontSize(12).
				TextColor(tk.TextSecondary)

		case model.AuthBasic, model.AuthDigest:
			userStr := ""
			if auth.Username != nil {
				userStr = *auth.Username
			}
			passStr := ""
			if auth.Password != nil {
				passStr = *auth.Password
			}

			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Username").FontSize(12).TextColor(tk.Text)
					})
					field.Input(c, &userStr, field.InputProps{
						Placeholder: "Username",
						Size:        lobetheme.Small,
					})
					auth.Username = &userStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Password").FontSize(12).TextColor(tk.Text)
					})
					field.Input(c, &passStr, field.InputProps{
						Placeholder: "Password",
						Size:        lobetheme.Small,
					})
					auth.Password = &passStr
				})
			})

		case model.AuthBearer:
			tokenStr := ""
			if auth.Token != nil {
				tokenStr = *auth.Token
			}
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Box(c).Width(80).Children(func() {
					ui.Text(c, "Token").FontSize(12).TextColor(tk.Text)
				})
				field.Input(c, &tokenStr, field.InputProps{
					Placeholder: "Bearer token",
					Size:        lobetheme.Small,
				})
				auth.Token = &tokenStr
			})

		case model.AuthAPIKey:
			keyStr := ""
			if auth.Key != nil {
				keyStr = *auth.Key
			}
			valStr := ""
			if auth.Value != nil {
				valStr = *auth.Value
			}

			ui.Column(c).FillWidth().Gap(8).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Key").FontSize(12).TextColor(tk.Text)
					})
					field.Input(c, &keyStr, field.InputProps{
						Placeholder: "Header / Param name",
						Size:        lobetheme.Small,
					})
					auth.Key = &keyStr
				})

				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Width(80).Children(func() {
						ui.Text(c, "Value").FontSize(12).TextColor(tk.Text)
					})
					field.Input(c, &valStr, field.InputProps{
						Placeholder: "Value",
						Size:        lobetheme.Small,
					})
					auth.Value = &valStr
				})
			})
		}

		p.SetAuth(auth)
	})
}

func renderCollectionHeaders(c *ui.Context, p *presenters.SettingsPresenter) {
	tk := lobetheme.TokensOf(c)

	hdrs := p.Headers()

	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, "Default Headers (applied to all requests in scope)").
			FontSize(12).
			Bold().
			TextColor(tk.Text)

		widgets.KeyValueGrid(c, &hdrs, widgets.KeyValueGridProps{
			ShowDescription: true,
			KeyPlaceholder:  "Header Name",
			ValPlaceholder:  "Header Value",
			OnChanged: func() {
				p.SetHeaders(hdrs)
			},
		})
	})
}

func renderCollectionVars(c *ui.Context, p *presenters.SettingsPresenter) {
	tk := lobetheme.TokensOf(c)

	vars := p.Vars()

	var rows []model.KeyValue
	for _, v := range vars {
		valStr := ""
		if v.Value != nil {
			valStr = fmt.Sprintf("%v", v.Value)
		}
		rows = append(rows, model.KeyValue{
			Name:        v.Name,
			Value:       &valStr,
			Description: v.Description,
			Enabled:     v.Enabled,
		})
	}

	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, "Collection Variables").
			FontSize(12).
			Bold().
			TextColor(tk.Text)

		widgets.KeyValueGrid(c, &rows, widgets.KeyValueGridProps{
			ShowDescription: true,
			KeyPlaceholder:  "Variable Name",
			ValPlaceholder:  "Default Value",
			OnChanged: func() {
				var newVars []model.Variable
				for _, r := range rows {
					var val any
					if r.Value != nil {
						val = *r.Value
					}
					newVars = append(newVars, model.Variable{
						Name:        r.Name,
						Value:       val,
						Description: r.Description,
						Enabled:     r.Enabled,
					})
				}
				p.SetVars(newVars)
			},
		})
	})
}

func renderCollectionScript(c *ui.Context, p *presenters.SettingsPresenter, scriptEditorState **editor.State) {
	tk := lobetheme.TokensOf(c)

	scpt := p.Scripts()
	if scpt == nil {
		scpt = &model.Scripts{}
	}

	initialScript := ""
	if scpt.Req != nil {
		initialScript = *scpt.Req
	}

	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, "Collection Pre-Request Script").
			FontSize(12).
			Bold().
			TextColor(tk.Text)

		widgets.CodeEditor(c, scriptEditorState, initialScript, widgets.CodeEditorProps{
			Language:    "javascript",
			Placeholder: "// Pre-request script applied to all requests in collection...",
			OnChanged: func(newContent string) {
				scpt.Req = &newContent
				p.SetScripts(scpt)
			},
		})
	})
}

func renderCollectionDocs(c *ui.Context, p *presenters.SettingsPresenter, docsEditorState **editor.State) {
	tk := lobetheme.TokensOf(c)

	initialDocs := p.Docs()

	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, "Collection Documentation (Markdown)").
			FontSize(12).
			Bold().
			TextColor(tk.Text)

		widgets.CodeEditor(c, docsEditorState, initialDocs, widgets.CodeEditorProps{
			Language:    "markdown",
			Placeholder: "# Collection Documentation\n\nOverview of this collection's APIs and conventions.",
			OnChanged: func(newContent string) {
				p.SetDocs(newContent)
			},
		})
	})
}
