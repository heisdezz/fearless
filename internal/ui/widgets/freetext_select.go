package widgets

import (
	"strings"

	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"
)

// FreeTextSelectProps configures the FreeTextSelect widget.
type FreeTextSelectProps struct {
	Placeholder string
	Options     []string
	Size        lobetheme.Size
	OnSelect    func(val string)
	OnChanged   func(val string)
}

// FreeTextSelect renders a combobox combining an editable text input with an autocomplete dropdown.
// It allows picking from suggested options or typing any arbitrary value.
func FreeTextSelect(c *ui.Context, value *string, props FreeTextSelectProps) ui.Element {
	tk := lobetheme.TokensOf(c)

	val := ""
	if value != nil {
		val = *value
	}

	size := props.Size
	if size == 0 {
		size = lobetheme.Small
	}

	// Filter options matching current value (case-insensitive)
	var matches []string
	lowerVal := strings.ToLower(strings.TrimSpace(val))
	for _, opt := range props.Options {
		if lowerVal == "" || strings.Contains(strings.ToLower(opt), lowerVal) {
			matches = append(matches, opt)
		}
	}

	// Show dropdown if there are matching options and input isn't an exact match
	isExactMatch := false
	for _, opt := range props.Options {
		if strings.EqualFold(opt, val) {
			isExactMatch = true
			break
		}
	}
	showDropdown := len(matches) > 0 && !isExactMatch && val != ""

	return ui.Column(c).FillWidth().Gap(2).Children(func() {
		// Input field
		FieldInput(c, value, FieldInputProps{
			Placeholder: props.Placeholder,
			Size:        size,
		})

		// Dropdown menu
		if showDropdown {
			ui.Box(c).
				FillWidth().
				Background(tk.BgElevated).
				Border(1, tk.BorderSecondary).
				Radius(4).
				Padding(2, 4).
				Children(func() {
					maxDisplay := 8
					if len(matches) < maxDisplay {
						maxDisplay = len(matches)
					}
					for i := 0; i < maxDisplay; i++ {
						opt := matches[i]
						ui.Row(c).
							FillWidth().
							Padding(4, 6).
							Radius(2).
							OnClick(func() {
								if value != nil {
									*value = opt
									if props.OnSelect != nil {
										props.OnSelect(opt)
									}
									if props.OnChanged != nil {
										props.OnChanged(opt)
									}
								}
							}).
							Children(func() {
								ui.Text(c, opt).
									FontSize(12).
									TextColor(tk.Text)
							})
					}
				})
		}
	})
}
