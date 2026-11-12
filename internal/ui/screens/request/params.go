package request

import (
	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	uitheme "fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// renderParamsTab renders the Query and Path parameter grids.
func renderParamsTab(c *ui.Context, p *presenters.RequestPresenter, req *model.Request) ui.Element {
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
			uitheme.PanelHeader(c, "Query Parameters")

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
			uitheme.PanelHeader(c, "Path Parameters")

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
