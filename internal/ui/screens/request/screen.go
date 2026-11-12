package request

import (
	"context"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/presenters"
	uitheme "fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

var methodSelectOptions = []field.SelectOption{
	{Value: "GET", Label: "GET"},
	{Value: "POST", Label: "POST"},
	{Value: "PUT", Label: "PUT"},
	{Value: "DELETE", Label: "DELETE"},
	{Value: "PATCH", Label: "PATCH"},
	{Value: "OPTIONS", Label: "OPTIONS"},
	{Value: "HEAD", Label: "HEAD"},
}

var subTabItems = []struct {
	tab   presenters.RequestSubTab
	label string
}{
	{presenters.SubTabParams, "Params"},
	{presenters.SubTabBody, "Body"},
	{presenters.SubTabHeaders, "Headers"},
	{presenters.SubTabAuth, "Auth"},
	{presenters.SubTabVars, "Vars"},
	{presenters.SubTabScript, "Script"},
	{presenters.SubTabAssert, "Assert"},
	{presenters.SubTabTests, "Tests"},
	{presenters.SubTabDocs, "Docs"},
}

// RequestScreenState persists editor state across view passes.
type RequestScreenState struct {
	JSONEditor   *editor.State
	TextEditor   *editor.State
	XMLEditor    *editor.State
	ScriptEditor *editor.State
	TestsEditor  *editor.State
	DocsEditor   *editor.State
}

// RequestScreen renders the comprehensive request editor screen.
func RequestScreen(c *ui.Context, p *presenters.RequestPresenter, state ...*RequestScreenState) ui.Element {
	return RequestScreenWith(c, p, Options{}, state...)
}

// Options carries the environment context the editor needs to interpret
// `{{variables}}`: which names resolve, and what to do when one is clicked.
type Options struct {
	// Resolves reports whether a variable name is defined in the current
	// scopes, so the URL bar can mark an undefined one.
	Resolves widgets.VariableLookup
	// OnOpenVariable runs when a `{{variable}}` in the URL bar is clicked.
	OnOpenVariable func(name string)
	// Field holds the URL bar's edit state across frames.
	Field *widgets.TemplateField
}

// RequestScreenWith is RequestScreen with environment options.
func RequestScreenWith(c *ui.Context, p *presenters.RequestPresenter, opts Options, state ...*RequestScreenState) ui.Element {
	tk := lobetheme.TokensOf(c)

	var st *RequestScreenState
	if len(state) > 0 && state[0] != nil {
		st = state[0]
	} else {
		st = &RequestScreenState{}
	}

	if p == nil || p.EffectiveRequest() == nil {
		// Empty state when no request is selected
		return ui.Column(c).Fill().Center().Gap(8).Children(func() {
			ui.Text(c, "No Request Selected").
				FontSize(14).
				Bold().
				TextColor(tk.TextSecondary)
			ui.Text(c, "Select a request from the sidebar to start editing.").
				FontSize(12).
				TextColor(tk.TextSecondary)
		})
	}

	req := p.EffectiveRequest()
	if req.Method == "" {
		req.Method = "GET"
	}

	return ui.Column(c).Fill().Padding(12, 16).Gap(12).Children(func() {
		// 1. Top URL & Action Bar
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			// HTTP Method Badge / Select
			ui.Box(c).Width(100).Children(func() {
				widgets.FieldSelect(c, &req.Method, widgets.FieldSelectProps{
					Options: methodSelectOptions,
					Size:    lobetheme.Small,
				})
			})

			// Target URL Input (styled with DataGroup for monospace tabular font)
			ui.Box(c).Grow(1).Children(func() {
				uitheme.DataGroup(c, func() {
					field := opts.Field
					if field == nil {
						field = &widgets.TemplateField{}
					}
					widgets.TemplateFieldView(c, field, &req.URL, widgets.TemplateFieldProps{
						Placeholder:     "https://api.example.com/v1/endpoint",
						Size:            lobetheme.Small,
						Resolves:        opts.Resolves,
						OnVariableClick: opts.OnOpenVariable,
						OnChanged: func(v string) {
							req.URL = v
						},
					})
				})
			})

			// Send / Cancel Action Button
			if p.InFlight() {
				widgets.Button(c, "Cancel", widgets.ButtonProps{
					Variant: action.Default,
					Size:    lobetheme.Small,
				}).OnClick(func() {
					p.Cancel(context.Background())
				})
			} else {
				widgets.Button(c, "Send", widgets.ButtonProps{
					Variant: action.Primary,
					Size:    lobetheme.Small,
				}).OnClick(func() {
					p.Send(context.Background())
				})
			}

			// Save Button
			saveLabel := "Save"
			if p.IsDirty() {
				saveLabel = "Save *"
			}
			widgets.Button(c, saveLabel, widgets.ButtonProps{
				Variant: action.Default,
				Size:    lobetheme.Small,
			}).OnClick(func() {
				p.Save(context.Background())
			})
		})

		// 2. Sub-Tabs Bar: the world's strip, shared with the sidebar's
		// switcher and the response pane's sub-tabs.
		//
		// It was Lobe's nav.Tabs until the finish review: that component
		// paints its own blue underline, so the window carried two blues
		// for one selection.
		selectedTab := p.ActiveSubTab()
		selectedIdx := 0
		items := make([]uitheme.TabItem, 0, len(subTabItems))
		for i, item := range subTabItems {
			items = append(items, uitheme.TabItem{Label: item.label})
			if item.tab == selectedTab {
				selectedIdx = i
			}
		}

		uitheme.TabStrip(c, uitheme.TabStripProps{
			Items:  items,
			Active: selectedIdx,
			OnSelect: func(i int) {
				if i < 0 || i >= len(subTabItems) {
					return
				}
				p.SelectSubTab(subTabItems[i].tab)
			},
		})

		if selectedIdx >= 0 && selectedIdx < len(subTabItems) {
			newTab := subTabItems[selectedIdx].tab
			if newTab != selectedTab {
				p.SelectSubTab(newTab)
				selectedTab = newTab
			}
		}

		// 3. Active Sub-Tab Panel (scrollable)
		ui.Box(c).Fill().Padding(8, 0).Children(func() {
			switch selectedTab {
			case presenters.SubTabParams:
				renderParamsTab(c, p, req)
			case presenters.SubTabBody:
				renderBodyTab(c, p, req, &st.JSONEditor, &st.TextEditor, &st.XMLEditor)
			case presenters.SubTabHeaders:
				renderHeadersTab(c, p, req)
			case presenters.SubTabAuth:
				renderAuthTab(c, p, req)
			case presenters.SubTabVars:
				renderVarsTab(c, p, req)
			case presenters.SubTabScript:
				renderScriptTab(c, p, req, &st.ScriptEditor)
			case presenters.SubTabAssert:
				renderAssertTab(c, p, req)
			case presenters.SubTabTests:
				renderTestsTab(c, p, req, &st.TestsEditor)
			case presenters.SubTabDocs:
				renderDocsTab(c, p, req, &st.DocsEditor)
			default:
				renderParamsTab(c, p, req)
			}
		})
	})
}
