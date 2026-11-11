package request

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/lobehub/lobe-mygo/pkg/action"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/widgets"
)

var bodyModeOptions = []string{"none", "json", "text", "xml", "formUrlEncoded", "multipartForm"}

// renderBodyTab renders the body mode selector and corresponding body content editor.
func renderBodyTab(
	c *ui.Context,
	p *presenters.RequestPresenter,
	req *model.Request,
	jsonEditorState **editor.State,
	textEditorState **editor.State,
	xmlEditorState **editor.State,
) ui.Element {
	tk := lobetheme.TokensOf(c)

	if req.Body == nil {
		req.Body = &model.Body{Mode: model.BodyNone}
	}

	// Find active mode index
	currentModeIndex := 0
	for i, opt := range bodyModeOptions {
		if model.BodyMode(opt) == req.Body.Mode {
			currentModeIndex = i
			break
		}
	}

	return ui.Column(c).FillWidth().Gap(12).Children(func() {
		// 1. Mode selector segmented control
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, "Body Mode:").FontSize(11).Bold().TextColor(tk.TextSecondary)

			action.Segmented(c, &currentModeIndex, bodyModeOptions, action.SegmentedProps{
				Size: lobetheme.Small,
			})
			req.Body.Mode = model.BodyMode(bodyModeOptions[currentModeIndex])
		})

		// 2. Editor based on active mode
		switch req.Body.Mode {
		case model.BodyNone:
			ui.Box(c).Padding(24, 0).Center().Children(func() {
				ui.Text(c, "This request does not have a body.").
					FontSize(12).
					TextColor(tk.TextSecondary)
			})

		case model.BodyJSON:
			initialJSON := ""
			if req.Body.JSON != nil {
				initialJSON = *req.Body.JSON
			}
			widgets.CodeEditor(c, jsonEditorState, initialJSON, widgets.CodeEditorProps{
				Language:    "json",
				Placeholder: "{\n  \"key\": \"value\"\n}",
				OnChanged: func(newContent string) {
					req.Body.JSON = &newContent
				},
			})

		case model.BodyText:
			initialText := ""
			if req.Body.Text != nil {
				initialText = *req.Body.Text
			}
			widgets.CodeEditor(c, textEditorState, initialText, widgets.CodeEditorProps{
				Language:    "text",
				Placeholder: "Enter plain text payload...",
				OnChanged: func(newContent string) {
					req.Body.Text = &newContent
				},
			})

		case model.BodyXML:
			initialXML := ""
			if req.Body.XML != nil {
				initialXML = *req.Body.XML
			}
			widgets.CodeEditor(c, xmlEditorState, initialXML, widgets.CodeEditorProps{
				Language:    "xml",
				Placeholder: "<request>\n  <item>value</item>\n</request>",
				OnChanged: func(newContent string) {
					req.Body.XML = &newContent
				},
			})

		case model.BodyFormURLEncoded:
			widgets.KeyValueGrid(c, &req.Body.FormURLEncoded, widgets.KeyValueGridProps{
				ShowDescription: true,
				KeyPlaceholder:  "Field Name",
				ValPlaceholder:  "Field Value",
			})

		case model.BodyMultipartForm:
			// Adapt MultipartEntry to KeyValue for editing
			var mpRows []model.KeyValue
			for _, entry := range req.Body.MultipartForm {
				name := entry.Name
				val := entry.Value
				desc := entry.Description
				mpRows = append(mpRows, model.KeyValue{
					Name:        name,
					Value:       val,
					Description: desc,
					Enabled:     entry.Enabled,
				})
			}

			widgets.KeyValueGrid(c, &mpRows, widgets.KeyValueGridProps{
				ShowDescription: true,
				KeyPlaceholder:  "Part Name",
				ValPlaceholder:  "Part Value / File",
				OnChanged: func() {
					var newEntries []model.MultipartEntry
					for _, r := range mpRows {
						newEntries = append(newEntries, model.MultipartEntry{
							Name:        r.Name,
							Value:       r.Value,
							Description: r.Description,
							Enabled:     r.Enabled,
							Type:        model.MultipartText,
						})
					}
					req.Body.MultipartForm = newEntries
				},
			})
		}
	})
}
