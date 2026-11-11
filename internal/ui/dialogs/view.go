package dialogs

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
	"github.com/lobehub/lobe-mygo/pkg/field"
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
		return 620
	}
	return 520
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
				var input ui.Element
				if f.Multiline {
					input = field.TextArea(c, &value, field.TextAreaProps{
						Variant:     field.InputOutlined,
						Placeholder: f.Placeholder,
						MinRows:     6,
						MaxRows:     14,
					})
				} else {
					input = field.Input(c, &value, field.InputProps{
						Variant:     field.InputOutlined,
						Placeholder: f.Placeholder,
						Invalid:     q.Phase() == PhaseFailed && f.Required && value == "",
					})
				}
				input.AlignSelf(ui.Stretch)
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
			feedback.Alert(c, "Some details were adapted", feedback.AlertProps{
				Type:        feedback.Warning,
				Description: joinWarnings(q.Warnings()),
				Variant:     feedback.Soft,
				Selectable:  true,
			}).AlignSelf(ui.Stretch)
		}
	})
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
		if action.Button(c, "Cancel", action.ButtonProps{}).Clicked() {
			q.Cancel()
		}
		b := action.Button(c, q.OKText(), action.ButtonProps{Variant: action.Primary})
		if q.Busy() {
			b.Disabled(true)
		}
		if b.Clicked() && !q.Busy() {
			q.OK()
		}
	})
}
