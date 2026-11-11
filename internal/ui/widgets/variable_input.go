package widgets

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"
)

// VariableInputProps configures the VariableInput widget.
type VariableInputProps struct {
	Placeholder string
	Size        lobetheme.Size
	Variables   []string // List of available variable names for interpolation
	OnChanged   func(newVal string)
}

// VariableInput renders an input field with variable completion suggestions.
func VariableInput(c *ui.Context, value *string, props VariableInputProps) ui.Element {
	tk := lobetheme.TokensOf(c)

	val := ""
	if value != nil {
		val = *value
	}

	size := props.Size
	if size == 0 {
		size = lobetheme.Small
	}

	// Detect if last characters are "{{'
	showSuggestions := strings.HasSuffix(val, "{{") && len(props.Variables) > 0

	return ui.Column(c).FillWidth().Gap(2).Children(func() {
		// Input field
		field.Input(c, value, field.InputProps{
			Placeholder: props.Placeholder,
			Size:        size,
		})

		// Suggestions dropdown popup if trigger active
		if showSuggestions {
			ui.Box(c).
				FillWidth().
				Background(tk.BgElevated).
				Border(1, tk.BorderSecondary).
				Radius(4).
				Padding(4, 6).
				Children(func() {
					ui.Text(c, "Available Variables:").
						FontSize(10).
						Bold().
						TextColor(tk.TextSecondary)

					for _, v := range props.Variables {
						varName := v
						ui.Row(c).
							FillWidth().
							Padding(2, 4).
							Radius(2).
							OnClick(func() {
								if value != nil {
									*value = (*value) + varName + "}}"
									if props.OnChanged != nil {
										props.OnChanged(*value)
									}
								}
							}).
							Children(func() {
								ui.Text(c, "{{"+varName+"}}").
									FontSize(11).
									TextColor(tk.Primary)
							})
					}
				})
		}
	})
}
