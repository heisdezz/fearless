package dialogs

import (
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/widgets"
)

// View renders whatever the queue currently holds, or nothing when it is closed.
//
// It is the single entry point the shell calls: one call site means the "only
// one dialog at a time" rule is structural, not a convention each flow has to
// remember. open mirrors queue state so Lobe's Modal can animate; the queue
// remains the source of truth, and View pushes queue state into *open each
// frame rather than letting Modal close the queue behind its back.
func View(c *ui.Context, q *Queue) {
	if q == nil || !q.Active() {
		return
	}

	open := true
	props := feedback.ModalProps{
		Title:      q.Title(),
		OkText:     q.OKText(),
		CancelText: "Cancel",
		Width:      widthFor(q.Kind()),
	}
	if q.Required() {
		// No Cancel and no Esc: draw our own Continue button in the body so the
		// only way out is to answer.
		props.NoFooter = true
	}

	ok := feedback.Modal(c, &open, props, func() {
		body(c, q)
	})
	if ok {
		q.OK()
	}
	if !open {
		q.Cancel()
	}
}

// widthFor sizes the panel to its content: a confirmation is narrow, a form and
// an import area are wide.
func widthFor(k Kind) float32 {
	switch k {
	case KindConfirm:
		return 420
	case KindImport:
		return 640
	case KindCreateCollection:
		return 560
	case KindStorageLocation:
		return 520
	case KindNewRequest:
		return 560
	}
	return 520
}

// newRequestKinds are the types the New Request dialog offers, in the order the
// mock shows them: three across, then the remaining two.
var newRequestKinds = []struct {
	kind  string
	label string
}{
	{string(RequestKindHTTP), "HTTP"},
	{string(RequestKindGRPC), "gRPC"},
	{string(RequestKindCURL), "From cURL"},
	{string(RequestKindGraphQL), "GraphQL"},
	{string(RequestKindWebSocket), "WebSocket"},
}

// newRequestMethods are the HTTP verbs offered by the method picker.
var newRequestMethods = []field.SelectOption{
	{Value: "GET", Label: "GET"},
	{Value: "POST", Label: "POST"},
	{Value: "PUT", Label: "PUT"},
	{Value: "PATCH", Label: "PATCH"},
	{Value: "DELETE", Label: "DELETE"},
	{Value: "HEAD", Label: "HEAD"},
	{Value: "OPTIONS", Label: "OPTIONS"},
}

// renderNewRequestBody draws the New Request form: a type selector, a name
// field, and a method + URL row.
func renderNewRequestBody(c *ui.Context, q *Queue) {
	tk := lobetheme.TokensOf(c)

	ui.Column(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
		// Type selector: a three-column grid of radio-style choices.
		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "Type").FontSize(tk.FontSize).FontWeight(500).TextColor(tk.TextSecondary)
			current := q.Value("kind")
			if current == "" {
				current = string(RequestKindHTTP)
			}

			for rowStart := 0; rowStart < len(newRequestKinds); rowStart += 3 {
				end := rowStart + 3
				if end > len(newRequestKinds) {
					end = len(newRequestKinds)
				}
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					for _, opt := range newRequestKinds[rowStart:end] {
						opt := opt
						selected := current == opt.kind
						ui.ButtonBase(c).
							Grow(1).
							Row().AlignItems(ui.Center).Gap(8).
							Padding(6, 4).
							Cursor(ui.CursorPointer).
							OnClick(func() {
								q.SetValue("kind", opt.kind)
							}).
							Children(func() {
								// A ring, filled when chosen, drawn rather than themed so it
								// reads the same in both appearances.
								ui.Box(c).Size(14, 14).Radius(7).Shrink(0).
									BorderWidth(1).BorderColor(tk.Border).
									AlignSelf(ui.Center).
									Children(func() {
										if selected {
											ui.Box(c).Size(6, 6).Radius(3).Background(tk.Primary).
												Margin(3, 0, 0, 0)
										}
									})
								ui.Text(c, opt.label).FontSize(13).TextColor(tk.Text)
							})
					}
				})
			}
		})

		// Request Name
		ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "Request Name").FontSize(tk.FontSize).FontWeight(500).TextColor(tk.TextSecondary)
			name := q.Value("name")
			input := widgets.FieldInput(c, &name, widgets.FieldInputProps{Placeholder: "Request Name"})
			input.AlignSelf(ui.Stretch)
			if name != q.Value("name") {
				q.SetValue("name", name)
			}
		})

		// URL, with the method picker to its left.
		ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "URL").FontSize(tk.FontSize).FontWeight(500).TextColor(tk.TextSecondary)
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				method := q.Value("method")
				if method == "" {
					method = "GET"
				}
				ui.Box(c).Width(96).Shrink(0).Children(func() {
					widgets.FieldSelect(c, &method, widgets.FieldSelectProps{Options: newRequestMethods})
				})
				if method != q.Value("method") {
					q.SetValue("method", method)
				}

				url := q.Value("url")
				input := widgets.FieldInput(c, &url, widgets.FieldInputProps{Placeholder: "Request URL"})
				input.Grow(1)
				if url != q.Value("url") {
					q.SetValue("url", url)
				}
			})
		})

		if n := len(q.Warnings()); n > 0 {
			feedback.Alert(c, "Some details adapted", feedback.AlertProps{
				Type:        feedback.Warning,
				Description: joinWarnings(q.Warnings()),
				Variant:     feedback.Soft,
				Selectable:  true,
			}).AlignSelf(ui.Stretch)
		}
	})
}

type importTabItem struct {
	id    string
	label string
	icon  func() *ui.SVG
}

// renderImportBody renders Bruno-style import tabs and a drag-and-drop file dropzone.
func renderImportBody(c *ui.Context, q *Queue) {
	tk := lobetheme.TokensOf(c)
	bodyCol := ui.Column(c).Gap(16).AlignSelf(ui.Stretch)

	tabState := ui.Local(bodyCol, "importTab", func() string {
		return "file"
	})

	tabs := []importTabItem{
		{id: "file", label: "File", icon: lucide.FileInput},
		{id: "git", label: "Git Repository", icon: lucide.GitFork},
		{id: "github", label: "GitHub", icon: lucide.Globe},
		{id: "url", label: "URL", icon: lucide.Link2},
		{id: "raw", label: "Raw / cURL", icon: lucide.CodeXml},
	}

	// Subtabs strip matching Bruno design
	ui.Row(c).FillWidth().Gap(4).BorderWidth(0, 0, 1, 0).BorderColor(tk.Border).Children(func() {
		for _, t := range tabs {
			t := t
			isActive := *tabState == t.id

			btn := ui.ButtonBase(c).
				Row().AlignItems(ui.Center).Gap(6).
				Padding(8, 14).
				Cursor(ui.CursorPointer).
				BorderWidth(0, 0, 2, 0)

			if isActive {
				btn.BorderColor(tk.Primary)
			} else {
				btn.BorderColor(ui.Transparent)
			}

			btn.Children(func() {
				iconColor := tk.TextSecondary
				textColor := tk.TextSecondary
				if isActive {
					iconColor = tk.Text
					textColor = tk.Text
				}
				ui.Icon(c, t.icon()).FontSize(16).TextColor(iconColor)
				ui.Text(c, t.label).FontSize(13).FontWeight(ternary(isActive, 600, 400)).TextColor(textColor)
			})

			btn.OnClick(func() {
				*tabState = t.id
			})
		}
	})

	doc := q.Value("document")

	switch *tabState {
	case "file":
		// File dropzone area supporting desktop drag-and-drop and file picking
		up := field.Upload(c, field.UploadProps{
			Dragger:  true,
			Multiple: false,
			Disabled: q.Phase() == PhaseRunning,
			OnFiles: func(files []field.File) {
				if len(files) > 0 {
					f := files[0]
					if f.Path != "" {
						q.SetValue("document", f.Path)
					} else if len(f.Data) > 0 {
						q.SetValue("document", string(f.Data))
					}
				}
			},
		}, func() {
			if doc != "" {
				ui.Icon(c, lucide.FileCheck()).FontSize(36).TextColor(tk.Success)
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, filepath.Base(doc)).FontSize(15).FontWeight(600).TextColor(tk.Text)
				})
				if doc != filepath.Base(doc) {
					ui.Text(c, doc).FontSize(12).TextColor(tk.TextTertiary)
				}
				ui.Text(c, "Ready to import. Click or drop another file to replace.").
					FontSize(12).TextColor(tk.Primary)
			} else {
				ui.Icon(c, lucide.FileInput()).FontSize(36).TextColor(tk.TextTertiary)
				ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "Drop file(s) to import or ").FontSize(14).FontWeight(500).TextColor(tk.Text)
					ui.Text(c, "choose file(s)").FontSize(14).FontWeight(500).TextColor(tk.Primary).Underline()
				})
				ui.Text(c, "Supports Bruno, OpenCollection, Postman, Insomnia, OpenAPI 3.x / Swagger 2.0, WSDL, and ZIP formats").
					FontSize(12).TextColor(tk.TextTertiary).TextAlign(ui.Center)
			}
		})
		up.FillWidth().MinHeight(180)

		if doc != "" {
			ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
				widgets.Button(c, "Clear selected file", widgets.ButtonProps{}).OnClick(func() {
					q.SetValue("document", "")
				})
			})
		}

	case "git":
		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "Git Repository URL").FontSize(tk.FontSize).FontWeight(500).TextColor(tk.TextSecondary)
			val := doc
			input := widgets.FieldInput(c, &val, widgets.FieldInputProps{
				Placeholder: "https://github.com/organization/collection-repo.git",
			})
			input.AlignSelf(ui.Stretch)
			if val != doc {
				q.SetValue("document", val)
			}
			ui.Text(c, "Enter a Git repository clone URL. Fearless will parse and import collection requests.").
				FontSize(12).TextColor(tk.TextTertiary)
		})

	case "github":
		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "GitHub Repository or Gist URL").FontSize(tk.FontSize).FontWeight(500).TextColor(tk.TextSecondary)
			val := doc
			input := widgets.FieldInput(c, &val, widgets.FieldInputProps{
				Placeholder: "https://github.com/owner/repo or https://gist.github.com/...",
			})
			input.AlignSelf(ui.Stretch)
			if val != doc {
				q.SetValue("document", val)
			}
			ui.Text(c, "Enter a GitHub repository or raw Gist URL to import collection definitions directly.").
				FontSize(12).TextColor(tk.TextTertiary)
		})

	case "url":
		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "Collection or OpenAPI URL").FontSize(tk.FontSize).FontWeight(500).TextColor(tk.TextSecondary)
			val := doc
			input := widgets.FieldInput(c, &val, widgets.FieldInputProps{
				Placeholder: "https://example.com/api/openapi.json or Postman shared collection URL",
			})
			input.AlignSelf(ui.Stretch)
			if val != doc {
				q.SetValue("document", val)
			}
			ui.Text(c, "Enter a publicly accessible URL to a Postman collection or OpenAPI 3.x / Swagger 2.0 JSON or YAML.").
				FontSize(12).TextColor(tk.TextTertiary)
		})

	case "raw":
		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "Raw Specification or cURL").FontSize(tk.FontSize).FontWeight(500).TextColor(tk.TextSecondary)
			val := doc
			input := widgets.FieldInput(c, &val, widgets.FieldInputProps{
				Placeholder: "Paste cURL command, Postman JSON, Insomnia export, OpenAPI YAML, or .bru file content here...",
				Multiline:   true,
				MinRows:     8,
				MaxRows:     14,
			})
			input.AlignSelf(ui.Stretch)
			if val != doc {
				q.SetValue("document", val)
			}
			ui.Text(c, "Paste raw JSON, YAML, or curl commands directly into this area.").
				FontSize(12).TextColor(tk.TextTertiary)
		})
	}
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

// body draws the dialog's content: the message, then one control per field.
func body(c *ui.Context, q *Queue) {
	tk := lobetheme.TokensOf(c)

	ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
		if msg := q.Message(); msg != "" {
			feedback.Alert(c, msg, feedback.AlertProps{
				Type:       messageKind(q),
				Variant:    feedback.Plain,
				Selectable: true,
			}).AlignSelf(ui.Stretch)
		}

		if q.Kind() == KindImport {
			renderImportBody(c, q)
			if n := len(q.Warnings()); n > 0 {
				feedback.Alert(c, "Some details were adapted", feedback.AlertProps{
					Type:        feedback.Warning,
					Description: joinWarnings(q.Warnings()),
					Variant:     feedback.Soft,
					Selectable:  true,
				}).AlignSelf(ui.Stretch)
			}
			return
		}

		if q.Kind() == KindNewRequest {
			renderNewRequestBody(c, q)
			return
		}

		for _, f := range q.Fields() {
			if f.Key == "collection" {
				// Drawn below as a picker rather than a text field.
				continue
			}
			ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				if f.Label != "" {
					ui.Text(c, f.Label).FontSize(tk.FontSize).FontWeight(500).TextColor(tk.TextSecondary)
				}
				value := q.Value(f.Key)
				inputRow := ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center)
				inputRow.Children(func() {
					ui.Box(c).Grow(1).Children(func() {
						input := widgets.FieldInput(c, &value, widgets.FieldInputProps{
							Placeholder: f.Placeholder,
							Multiline:   f.Multiline,
							MinRows:     6,
							MaxRows:     14,
							Invalid:     q.Phase() == PhaseFailed && f.Required && value == "",
						})
						input.AlignSelf(ui.Stretch)
					})
					if h := browseHeight(f); h > 0 && q.CanBrowse() {
						widgets.Button(c, "Browse", widgets.ButtonProps{}).Height(h).OnClick(q.Browse)
					}
				})
				if value != q.Value(f.Key) {
					q.SetValue(f.Key, value)
				}
			})
		}

		if q.Kind() == KindSaveTransient {
			ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "Collection").FontSize(tk.FontSize).FontWeight(500).TextColor(tk.TextSecondary)
				value := q.Value("collection")
				widgets.FreeTextSelect(c, &value, widgets.FreeTextSelectProps{
					Options:     q.TargetLabels(),
					Placeholder: "choose a collection",
				})
				if value != q.Value("collection") {
					q.SetValue("collection", value)
				}
			})
		}

		if n := len(q.Warnings()); n > 0 {
			feedback.Alert(c, "Some details adapted", feedback.AlertProps{
				Type:        feedback.Warning,
				Description: joinWarnings(q.Warnings()),
				Variant:     feedback.Soft,
				Selectable:  true,
			}).AlignSelf(ui.Stretch)
		}

		if q.Required() {
			// The footer is hidden for a required dialog, so its only button lives
			// here: full width, right-aligned, disabled until the path is set.
			ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
				widgets.Button(c, q.OKText(), widgets.ButtonProps{Variant: action.Primary}).
					OnClick(q.OK)
			})
		}
	})
}

// browseHeight is the control height to match a field, or 0 for no Browse
// button: only single-line path fields get one.
func browseHeight(f Field) float32 {
	if f.Key != "location" || f.Multiline {
		return 0
	}
	return 32
}

// messageKind maps a failed dialog to the alert colour it deserves.
func messageKind(q *Queue) feedback.AlertType {
	switch q.Phase() {
	case PhaseFailed:
		return feedback.Error
	case PhaseRunning:
		return feedback.Info
	}
	return feedback.Secondary
}

// joinWarnings keeps a long warning list inside a dialog.
func joinWarnings(ws []string) string {
	const max = 6
	out := ""
	for i, w := range ws {
		if i == max {
			out += "\n…\n"
			break
		}
		if i > 0 {
			out += "\n"
		}
		out += "• " + w
	}
	return out
}

// Actions returns the buttons a footer needs, for shells that draw their own
// chrome instead of using feedback.Modal.
func Actions(c *ui.Context, q *Queue) ui.Element {
	return ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
		if widgets.Button(c, "Cancel", widgets.ButtonProps{}).Clicked() {
			q.Cancel()
		}
		b := widgets.Button(c, q.OKText(), widgets.ButtonProps{Variant: action.Primary})
		if q.Busy() {
			b.Disabled(true)
		}
		if b.Clicked() && !q.Busy() {
			q.OK()
		}
	})
}
