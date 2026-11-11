package request

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/widgets"
)

// renderScriptTab renders the Pre-Request JavaScript script editor.
func renderScriptTab(
	c *ui.Context,
	p *presenters.RequestPresenter,
	req *model.Request,
	scriptEditorState **editor.State,
) ui.Element {
	tk := lobetheme.TokensOf(c)

	if req.Script == nil {
		req.Script = &model.Scripts{}
	}

	initialScript := ""
	if req.Script.Req != nil {
		initialScript = *req.Script.Req
	}

	return ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, "Pre-Request Script (runs before request is sent)").
			FontSize(12).
			Bold().
			TextColor(tk.Text)

		widgets.CodeEditor(c, scriptEditorState, initialScript, widgets.CodeEditorProps{
			Language:    "javascript",
			Placeholder: "// Example:\n// bru.setEnvVar('token', 'abc');",
			OnChanged: func(newContent string) {
				req.Script.Req = &newContent
			},
		})
	})
}

// renderAssertTab renders the declarative Assertions key/value grid.
func renderAssertTab(c *ui.Context, p *presenters.RequestPresenter, req *model.Request) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, "Response Assertions").
			FontSize(12).
			Bold().
			TextColor(tk.Text)

		widgets.KeyValueGrid(c, &req.Assertions, widgets.KeyValueGridProps{
			ShowDescription: false,
			KeyPlaceholder:  "Expression (e.g. res.status)",
			ValPlaceholder:  "Expected Value (e.g. eq 200)",
		})
	})
}

// renderTestsTab renders the Post-Response JavaScript tests editor.
func renderTestsTab(
	c *ui.Context,
	p *presenters.RequestPresenter,
	req *model.Request,
	testsEditorState **editor.State,
) ui.Element {
	tk := lobetheme.TokensOf(c)

	initialTests := ""
	if req.Tests != nil {
		initialTests = *req.Tests
	}

	return ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, "Tests (runs after response is received)").
			FontSize(12).
			Bold().
			TextColor(tk.Text)

		widgets.CodeEditor(c, testsEditorState, initialTests, widgets.CodeEditorProps{
			Language:    "javascript",
			Placeholder: "test('status is 200', function() {\n  expect(res.getStatus()).to.equal(200);\n});",
			OnChanged: func(newContent string) {
				req.Tests = &newContent
			},
		})
	})
}

// renderDocsTab renders the markdown documentation editor.
func renderDocsTab(
	c *ui.Context,
	p *presenters.RequestPresenter,
	req *model.Request,
	docsEditorState **editor.State,
) ui.Element {
	tk := lobetheme.TokensOf(c)

	initialDocs := ""
	if req.Docs != nil {
		initialDocs = *req.Docs
	}

	return ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, "Documentation (Markdown)").
			FontSize(12).
			Bold().
			TextColor(tk.Text)

		widgets.CodeEditor(c, docsEditorState, initialDocs, widgets.CodeEditorProps{
			Language:    "markdown",
			Placeholder: "# Request Documentation\n\nDescribe endpoint behavior and parameters.",
			OnChanged: func(newContent string) {
				req.Docs = &newContent
			},
		})
	})
}
