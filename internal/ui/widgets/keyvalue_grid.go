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
		// Bordered Table Card
		ui.Column(c).FillWidth().
			Border(1, tk.BorderSecondary).
			Radius(tk.BorderRadiusSM).
			Background(tk.BgContainer).
			Children(func() {
				// Header row
				ui.Row(c).FillWidth().Gap(8).Padding(6, 8).
					Background(tk.BgElevated).
					BorderWidth(0, 0, 1, 0).
					BorderColor(tk.BorderSecondary).
					AlignItems(ui.Center).
					Children(func() {
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
				if rows != nil && len(*rows) > 0 {
					for idx := range *rows {
						rowIdx := idx
						row := &(*rows)[rowIdx]

						ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Padding(4, 8).
							BorderWidth(0, 0, 1, 0).
							BorderColor(tk.BorderSecondary).
							Children(func() {
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
				} else {
					ui.Row(c).FillWidth().Padding(8, 12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "No parameters").FontSize(11).TextColor(tk.TextTertiary)
					})
				}
			})

		// Add Row button
		if !props.ReadOnly {
			ui.Row(c).Padding(2, 0).Children(func() {
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
