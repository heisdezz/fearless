package widgets

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/choice"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
)

// KeyValueGridProps configures the KeyValueGrid widget.
type KeyValueGridProps struct {
	ShowDescription bool
	ReadOnly        bool
	KeyPlaceholder  string
	ValPlaceholder  string
	OnChanged       func()
}

// KeyValueGrid renders an editable parameter grid for headers, query params, or variables.
func KeyValueGrid(c *ui.Context, rows *[]model.KeyValue, props KeyValueGridProps) ui.Element {
	tk := lobetheme.TokensOf(c)

	keyPlaceholder := props.KeyPlaceholder
	if keyPlaceholder == "" {
		keyPlaceholder = "Name"
	}
	valPlaceholder := props.ValPlaceholder
	if valPlaceholder == "" {
		valPlaceholder = "Value"
	}

	return ui.Column(c).FillWidth().Gap(6).Children(func() {
		// Header row
		ui.Row(c).FillWidth().Gap(8).Padding(4, 0).Children(func() {
			ui.Box(c).Width(24) // Checkbox column space
			ui.Box(c).Grow(1).Children(func() {
				ui.Text(c, "KEY").FontSize(11).Bold().TextColor(tk.TextSecondary)
			})
			ui.Box(c).Grow(1).Children(func() {
				ui.Text(c, "VALUE").FontSize(11).Bold().TextColor(tk.TextSecondary)
			})
			if props.ShowDescription {
				ui.Box(c).Grow(1).Children(func() {
					ui.Text(c, "DESCRIPTION").FontSize(11).Bold().TextColor(tk.TextSecondary)
				})
			}
			ui.Box(c).Width(28) // Delete action space
		})

		// Data rows
		if rows != nil {
			for idx := range *rows {
				rowIdx := idx
				row := &(*rows)[rowIdx]

				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					// 1. Checkbox
					choice.Checkbox(c, &row.Enabled, "", choice.CheckboxProps{})

					// 2. Key Input
					if row.Name == nil {
						empty := ""
						row.Name = &empty
					}
					FieldInput(c, row.Name, FieldInputProps{
						Placeholder: keyPlaceholder,
						Size:        lobetheme.Small,
					})

					// 3. Value Input
					if row.Value == nil {
						empty := ""
						row.Value = &empty
					}
					FieldInput(c, row.Value, FieldInputProps{
						Placeholder: valPlaceholder,
						Size:        lobetheme.Small,
					})

					// 4. Description (optional)
					if props.ShowDescription {
						if row.Description == nil {
							empty := ""
							row.Description = &empty
						}
						FieldInput(c, row.Description, FieldInputProps{
							Placeholder: "Description",
							Size:        lobetheme.Small,
						})
					}

					// 5. Delete action
					if !props.ReadOnly {
						IconButton(c, IconButtonProps{
							Icon:    lucide.X(),
							Label:   "Remove row",
							Tooltip: "Remove row",
							Size:    lobetheme.Small,
							Variant: action.Text,
						}).OnClick(func() {
							if rows != nil && rowIdx < len(*rows) {
								*rows = append((*rows)[:rowIdx], (*rows)[rowIdx+1:]...)
								if props.OnChanged != nil {
									props.OnChanged()
								}
							}
						})
					}
				})
			}
		}

		// Add Row button
		if !props.ReadOnly {
			ui.Row(c).Padding(4, 0).Children(func() {
				// A quiet control, like the rest of the window's actions:
				// the grid's own add-row is not a primary action.
				Button(c, "+ Add Row", ButtonProps{
					Size:    lobetheme.Small,
					Variant: action.Text,
				}).OnClick(func() {
					if rows != nil {
						emptyKey := ""
						emptyVal := ""
						*rows = append(*rows, model.KeyValue{
							Name:    &emptyKey,
							Value:   &emptyVal,
							Enabled: true,
						})
						if props.OnChanged != nil {
							props.OnChanged()
						}
					}
				})
			})
		}
	})
}
