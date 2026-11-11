package request

import (
	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/widgets"
)

// renderParamsTab renders the Query and Path parameter grids.
func renderParamsTab(c *ui.Context, p *presenters.RequestPresenter, req *model.Request) ui.Element {
	tk := lobetheme.TokensOf(c)

	// Filter Query and Path params from req.Params
	var queryRows []model.KeyValue
	var pathRows []model.KeyValue

	for _, param := range req.Params {
		if param.Type == model.ParamPath {
			pathRows = append(pathRows, param.KeyValue)
		} else {
			queryRows = append(queryRows, param.KeyValue)
		}
	}

	return ui.Column(c).FillWidth().Gap(16).Children(func() {
		// 1. Query Parameters
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			ui.Text(c, "Query Parameters").
				FontSize(12).
				Bold().
				TextColor(tk.Text)

			widgets.KeyValueGrid(c, &queryRows, widgets.KeyValueGridProps{
				ShowDescription: true,
				KeyPlaceholder:  "Parameter",
				ValPlaceholder:  "Value",
				OnChanged: func() {
					// Rebuild req.Params from queryRows and pathRows
					var newParams []model.Param
					for _, r := range queryRows {
						newParams = append(newParams, model.Param{
							KeyValue: r,
							Type:     model.ParamQuery,
						})
					}
					for _, r := range pathRows {
						newParams = append(newParams, model.Param{
							KeyValue: r,
							Type:     model.ParamPath,
						})
					}
					req.Params = newParams
				},
			})
		})

		// 2. Path Parameters
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			ui.Text(c, "Path Parameters").
				FontSize(12).
				Bold().
				TextColor(tk.Text)

			widgets.KeyValueGrid(c, &pathRows, widgets.KeyValueGridProps{
				ShowDescription: true,
				KeyPlaceholder:  "Param Name",
				ValPlaceholder:  "Value",
				OnChanged: func() {
					var newParams []model.Param
					for _, r := range queryRows {
						newParams = append(newParams, model.Param{
							KeyValue: r,
							Type:     model.ParamQuery,
						})
					}
					for _, r := range pathRows {
						newParams = append(newParams, model.Param{
							KeyValue: r,
							Type:     model.ParamPath,
						})
					}
					req.Params = newParams
				},
			})
		})
	})
}
