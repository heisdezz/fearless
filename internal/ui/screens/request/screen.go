package request

import (
	"context"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/nav"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/presenters"
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
				TextColor(tk.TextTertiary)
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
				field.Select(c, &req.Method, field.SelectProps{
					Options: methodSelectOptions,
					Size:    lobetheme.Small,
				})
			})

			// Target URL Input
			ui.Box(c).Grow(1).Children(func() {
				widgets.VariableInput(c, &req.URL, widgets.VariableInputProps{
					Placeholder: "https://api.example.com/v1/endpoint",
					Size:        lobetheme.Small,
					OnChanged: func(v string) {
						req.URL = v
					},
				})
			})

			// Send / Cancel Action Button
			if p.InFlight() {
				action.Button(c, "Cancel", action.ButtonProps{
					Variant: action.Default,
					Size:    lobetheme.Small,
				}).OnClick(func() {
					p.Cancel(context.Background())
				})
			} else {
				action.Button(c, "Send", action.ButtonProps{
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
			action.Button(c, saveLabel, action.ButtonProps{
				Variant: action.Default,
				Size:    lobetheme.Small,
			}).OnClick(func() {
				p.Save(context.Background())
			})
		})

		// 2. Sub-Tabs Bar
		selectedTab := p.ActiveSubTab()
		selectedIdx := 0
		var navItems []nav.TabItem
		for i, item := range subTabItems {
			navItems = append(navItems, nav.TabItem{Label: item.label})
			if item.tab == selectedTab {
				selectedIdx = i
			}
		}

		nav.Tabs(c, &selectedIdx, nav.TabsProps{
			Items:   navItems,
			Variant: nav.TabsSquare,
			Size:    lobetheme.Small,
		}, nil)

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
