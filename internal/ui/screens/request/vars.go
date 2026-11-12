package request

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	uitheme "fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// renderVarsTab renders the pre-request and post-response variable grids.
func renderVarsTab(c *ui.Context, p *presenters.RequestPresenter, req *model.Request) ui.Element {
	tk := uitheme.TokensOf(c)

	if req.Vars == nil {
		req.Vars = &model.VarBlocks{}
	}

	// 1. Pre-Request Variables
	var preRows []model.KeyValue
	for _, v := range req.Vars.PreRequest {
		valStr := ""
		if v.Value != nil {
			valStr = fmt.Sprintf("%v", v.Value)
		}
		preRows = append(preRows, model.KeyValue{
			Name:        v.Name,
			Value:       &valStr,
			Description: v.Description,
			Enabled:     v.Enabled,
		})
	}

	// 2. Post-Response Variables
	var postRows []model.KeyValue
	for _, v := range req.Vars.PostResponse {
		valStr := ""
		if v.Value != nil {
			valStr = fmt.Sprintf("%v", v.Value)
		}
		postRows = append(postRows, model.KeyValue{
			Name:        v.Name,
			Value:       &valStr,
			Description: v.Description,
			Enabled:     v.Enabled,
		})
	}

	return ui.Column(c).FillWidth().Gap(24).Children(func() {
		// Pre-Request Variables Section
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			uitheme.PanelHeader(c, "Pre-Request Variables")

			widgets.KeyValueGrid(c, &preRows, widgets.KeyValueGridProps{
				ShowDescription: true,
				KeyPlaceholder:  "Variable Name",
				ValPlaceholder:  "Value (expr)",
				OnChanged: func() {
					var newVars []model.Variable
					for _, r := range preRows {
						var val any
						if r.Value != nil {
							val = *r.Value
						}
						newVars = append(newVars, model.Variable{
							Name:        r.Name,
							Value:       val,
							Description: r.Description,
							Enabled:     r.Enabled,
						})
					}
					req.Vars.PreRequest = newVars
				},
			})
		})

		// Distinct horizontal divider between Pre-Request and Post-Response
		ui.Box(c).Height(1).FillWidth().Background(tk.BorderSecondary).Margin(8, 0)

		// Post-Response Variables Section
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			uitheme.PanelHeader(c, "Post-Response Variables")

			widgets.KeyValueGrid(c, &postRows, widgets.KeyValueGridProps{
				ShowDescription: true,
				KeyPlaceholder:  "Variable Name",
				ValPlaceholder:  "Expression",
				OnChanged: func() {
					var newVars []model.Variable
					for _, r := range postRows {
						var val any
						if r.Value != nil {
							val = *r.Value
						}
						newVars = append(newVars, model.Variable{
							Name:        r.Name,
							Value:       val,
							Description: r.Description,
							Enabled:     r.Enabled,
						})
					}
					req.Vars.PostResponse = newVars
				},
			})
		})
	})
}
