package widgets

import (
	"strings"

	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"
)

// VariableLookup reports whether a variable name resolves, so a field can
// colour a defined variable apart from an undefined one.
type VariableLookup func(name string) bool

// TemplateField is a one-line editable field that shows `{{variable}}` tokens
// highlighted and clickable.
type TemplateField struct {
	// Editing is true while the raw input is shown instead of the token run.
	Editing bool
	// WasFocused remembers that the input had focus, so the field can return
	// to its token form when focus leaves.
	WasFocused bool
}

// TemplateFieldProps configures a TemplateField.
type TemplateFieldProps struct {
	Placeholder string
	Size        lobetheme.Size
	// Resolves reports whether a variable is defined; nil means "all defined".
	Resolves VariableLookup
	// OnVariableClick runs when a `{{variable}}` token is clicked, with the
	// name without braces, so a variable can be created or edited.
	OnVariableClick func(name string)
	OnChanged       func(string)
}

// TemplateFieldView renders the field.
//
// MyGo's text input paints one colour for its whole value, so `{{variables}}`
// could not be marked defined or undefined and could not be click targets of
// their own. The field therefore shows a token run when idle and swaps to a real
// input when clicked; clicking a variable token opens that variable instead of
// starting an edit.
//
// Keep one TemplateField per logical field across frames: it holds edit state.
func TemplateFieldView(c *ui.Context, f *TemplateField, value *string, p TemplateFieldProps) ui.Element {
	tk := lobetheme.TokensOf(c)
	val := ""
	if value != nil {
		val = *value
	}
	size := p.Size
	if size == 0 {
		size = lobetheme.Small
	}

	// A token click is recorded during the run and dispatched after the frame.
	var toOpen string

	box := ui.Row(c).
		FillWidth().
		Height(fieldHeight(size)).
		PaddingX(fieldPaddingX(size)).
		Radius(tk.BorderRadiusSM).
		Background(tk.BgContainer).
		Border(1, tk.Border).
		AlignItems(ui.Center)

	box.Children(func() {
		if f.Editing {
			input := FieldInput(c, value, FieldInputProps{Placeholder: p.Placeholder, Size: size})
			input.AlignSelf(ui.Stretch)
			if value != nil && *value != val && p.OnChanged != nil {
				p.OnChanged(*value)
			}
			// Leaving the field returns it to its token form, so the variables
			// are visible again once the user looks away.
			if input.Focused() {
				f.WasFocused = true
			} else if f.WasFocused {
				f.Editing, f.WasFocused = false, false
			}
			return
		}

		// Idle: a click anywhere in the run starts editing, except on a
		// variable token, which opens the variable. The run fills the box so
		// the whole field is one click target, not just the typed text.
		run := ui.Row(c).FillWidth().Grow(1).FillHeight().AlignItems(ui.Center)
		run.OnClick(func() { f.Editing = true })
		run.Children(func() {
			if strings.TrimSpace(val) == "" {
				ui.Text(c, p.Placeholder).FontSize(fieldFontSize(size)).TextColor(tk.TextPlaceholder)
				return
			}
			line := ui.Row(c).FillWidth().AlignItems(ui.Center)
			line.Children(func() { drawTemplate(tk, c, val, p, &toOpen) })
		})
	})

	if toOpen != "" && p.OnVariableClick != nil {
		// Opening a variable must not also start an edit.
		f.Editing, f.WasFocused = false, false
		p.OnVariableClick(toOpen)
	}
	return box
}

// drawTemplate lays out value as alternating plain text and variable tokens.
func drawTemplate(tk *lobetheme.Tokens, c *ui.Context, val string, p TemplateFieldProps, toOpen *string) {
	spans := ParseVariables(val)
	cursor := 0
	for _, sp := range spans {
		if sp.Start > cursor {
			ui.Text(c, val[cursor:sp.Start]).TextColor(tk.Text)
		}
		seg := sp
		// A defined variable is the accent; an undefined one is the danger
		// colour, so a typo reads before the request is sent.
		colour, tint := tk.Primary, tk.PrimaryFillTertiary
		if p.Resolves != nil && !p.Resolves(seg.Name) {
			colour, tint = tk.Error, tk.ErrorFillTertiary
		}
		token := ui.Box(c).Padding(1, 3).Radius(3).Background(tint)
		// The token takes the click itself: a click over its text would
		// otherwise land on the text, which offers no handler.
		if token.Clicked() {
			*toOpen = seg.Name
		}
		token.Children(func() {
			ui.Text(c, val[seg.Start:seg.End]).TextColor(colour)
		})
		cursor = seg.End
	}
	if cursor < len(val) {
		ui.Text(c, val[cursor:]).TextColor(tk.Text)
	}
}

// fieldHeight, fieldPaddingX and fieldFontSize mirror fieldSurface so a
// TemplateField lines up with the FieldInput beside it.
func fieldHeight(size lobetheme.Size) float32 {
	switch size {
	case lobetheme.Large:
		return 40
	case lobetheme.Small:
		return 24
	}
	return 32
}

func fieldPaddingX(size lobetheme.Size) float32 {
	switch size {
	case lobetheme.Large, 0:
		return 12
	case lobetheme.Small:
		return 8
	}
	return 12
}

func fieldFontSize(size lobetheme.Size) float32 {
	switch size {
	case lobetheme.Large:
		return 14
	case lobetheme.Small:
		return 12
	}
	return 13
}
