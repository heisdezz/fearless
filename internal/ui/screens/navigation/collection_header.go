package navigation

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// CollectionHeader renders the environment selector and collection actions strip.
func (s *Screen) CollectionHeader(c *ui.Context) ui.Element {
	tk := lobetheme.TokensOf(c)
	snap := s.Snapshot
	actions := s.Actions

	colName := snap.CollectionName
	if colName == "" {
		colName = "fearless"
	}

	return ui.Row(c).
		FillWidth().
		Height(40).
		Background(tk.BgLayout).
		BorderWidth(0, 0, 1, 0).
		BorderColor(tk.Border).
		Padding(0, 12).
		AlignItems(ui.Center).
		Children(func() {
			// Collection Name
			ui.Text(c, colName).
				FontSize(theme.TitleSize).
				FontWeight(500).
				TextColor(tk.Text)

			ui.Box(c).Grow(1) // Spacer between collection name and env selector

			// Environment Selector
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				theme.PanelHeader(c, "Environment")

				// Environment select dropdown
				var envOptions []field.SelectOption
				for _, env := range snap.Environments {
					envOptions = append(envOptions, field.SelectOption{
						Label: env,
						Value: env,
					})
				}
				if len(envOptions) == 0 {
					envOptions = append(envOptions, field.SelectOption{
						Label: "No Environment",
						Value: "",
					})
				}

				activeEnv := snap.ActiveEnvironment
				ui.Box(c).Width(140).Children(func() {
					widgets.FieldSelect(c, &activeEnv, widgets.FieldSelectProps{
						Options: envOptions,
						Size:    lobetheme.Small,
					})
				})
				if activeEnv != snap.ActiveEnvironment && actions.OnSelectEnvironment != nil {
					actions.OnSelectEnvironment(activeEnv)
				}

				// Environment settings / manage button
				widgets.IconButton(c, widgets.IconButtonProps{
					Icon:    lucide.Settings(),
					Label:   "Environment settings",
					Tooltip: "Environment settings",
					Variant: action.Text,
					Size:    lobetheme.Small,
				}).OnClick(func() {
					if actions.OnManageEnvironments != nil {
						actions.OnManageEnvironments()
					}
				})

				// Run button
				widgets.Button(c, "Run", widgets.ButtonProps{
					Size:    lobetheme.Small,
					Variant: action.Text,
				}).OnClick(func() {
					if actions.OnRunCollection != nil {
						actions.OnRunCollection(snap.CollectionUID)
					}
				})
			})
		})
}
