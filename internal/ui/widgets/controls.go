package widgets

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"
)

// This file is the app's single door to Lobe's field and action controls.
//
// Lobe builds these on MyGo the way Lobe UI builds on CSS: FieldInput styles a
// container row and then returns the inner input, and StyleButton's surface
// lands on a node that the returned element does not paint. On this MyGo
// version that leaves the control painting MyGo's default near-white surface,
// so every field and button came out white in the dark appearance even though
// the tokens it was handed were graphite (the white-input defect, D-017).
//
// These wrappers give the control a surface the app owns: an element the app
// builds with world tokens, which cannot be detached from what paints. Every
// screen goes through them instead of calling field.Input or action.Button
// directly, so the defect has one place to retire in when the toolkit fixes it.

// FieldInputProps configures the app's text input.
type FieldInputProps struct {
	Placeholder string
	Size        lobetheme.Size
	// Invalid marks the field with the danger seam and fill.
	Invalid bool
	// Multiline renders a text area between MinRows and MaxRows tall.
	Multiline bool
	MinRows   int
	MaxRows   int
}

// FieldInput is a single-line text input on the world's field surface.
func FieldInput(c *ui.Context, value *string, p FieldInputProps) ui.Element {
	tk := lobetheme.TokensOf(c)
	return fieldSurface(c, tk, p.Size, p.Invalid, func() {
		if p.Multiline {
			field.TextArea(c, value, field.TextAreaProps{
				Placeholder: p.Placeholder,
				Variant:     field.InputBorderless,
				MinRows:     p.MinRows,
				MaxRows:     p.MaxRows,
			})
			return
		}
		field.Input(c, value, field.InputProps{
			Placeholder: p.Placeholder,
			Size:        p.Size,
			Variant:     field.InputBorderless,
		})
	})
}

// fieldSurface draws the field box — panel ground, one seam, rounded — and
// builds the control borderless inside it, so the control's own surface cannot
// show through.
func fieldSurface(c *ui.Context, tk *lobetheme.Tokens, size lobetheme.Size, invalid bool, children func()) ui.Element {
	height, paddingX, radius := float32(32), float32(12), tk.BorderRadius
	switch size {
	case lobetheme.Small:
		height, paddingX, radius = 24, 8, tk.BorderRadiusSM
	case lobetheme.Large:
		height, paddingX, radius = 40, 12, tk.BorderRadiusLG
	}
	bg, border := tk.BgContainer, tk.Border
	if invalid {
		bg, border = tk.ErrorBg, tk.ErrorBorder
	}
	box := ui.Row(c).
		FillWidth().
		PaddingX(paddingX).
		Radius(radius).
		Background(bg).
		Border(1, border).
		AlignItems(ui.Center)
	if size == lobetheme.Small {
		box.Height(height)
	}
	if size != lobetheme.Small {
		box.MinHeight(height)
	}
	return box.Children(children)
}

// FieldSelectProps configures the app's select.
type FieldSelectProps struct {
	Options     []field.SelectOption
	Placeholder string
	Size        lobetheme.Size
	Disabled    bool
}

// FieldSelect is a select on the world's field surface. It takes the same
// borderless-in-an-owned-box treatment as FieldInput, for the same reason.
func FieldSelect(c *ui.Context, value *string, p FieldSelectProps) ui.Element {
	tk := lobetheme.TokensOf(c)
	return fieldSurface(c, tk, p.Size, false, func() {
		field.Select(c, value, field.SelectProps{
			Options:     p.Options,
			Placeholder: p.Placeholder,
			Size:        p.Size,
			Disabled:    p.Disabled,
			Variant:     field.InputBorderless,
		})
	})
}

// ButtonProps configures the app's button.
type ButtonProps struct {
	Variant action.ButtonVariant
	Size    lobetheme.Size
}

// Button is a Lobe button painted entirely on the world's tokens instead of
// Lobe's own fills, so no variant falls back to MyGo's default surface.
func Button(c *ui.Context, label string, p ButtonProps) ui.Element {
	tk := lobetheme.TokensOf(c)
	height, paddingX, fontSize, radius := float32(32), float32(14), float32(13), tk.BorderRadiusSM
	switch p.Size {
	case lobetheme.Small:
		height, paddingX, fontSize = 24, 8, 12
	case lobetheme.Large:
		height, paddingX, fontSize, radius = 40, 16, 14, tk.BorderRadius
	}

	bg, border, fg := buttonColors(tk, p.Variant)
	b := ui.ButtonBase(c).
		Height(height).
		PaddingX(paddingX).
		Radius(radius).
		FontSize(fontSize).
		FontWeight(500).
		Cursor(ui.CursorPointer).
		Background(bg).
		Border(1, border).
		TextColor(fg)
	if label != "" {
		b.Children(func() { ui.Text(c, label).SingleLine() })
	}
	return b
}

// buttonColors is the world's button table: the primary action carries the one
// accent, a normal button sits on the panel with a seam, and text buttons have
// no surface at all.
func buttonColors(tk *lobetheme.Tokens, v action.ButtonVariant) (bg, border, fg ui.Color) {
	switch v {
	case action.Primary:
		return tk.Primary, tk.PrimaryBorder, tk.PrimaryForeground
	case action.Text:
		return ui.Transparent, ui.Transparent, tk.Text
	case action.Link:
		return ui.Transparent, ui.Transparent, tk.Primary
	case action.Filled:
		return tk.FillTertiary, ui.Transparent, tk.Text
	case action.Dashed:
		return tk.BgContainer, tk.Border, tk.Text
	default:
		return tk.BgContainer, tk.Border, tk.Text
	}
}
