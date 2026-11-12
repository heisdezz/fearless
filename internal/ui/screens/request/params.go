package request

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	uitheme "fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// renderParamsTab renders the Query and Path parameter grids.
func renderParamsTab(c *ui.Context, p *presenters.RequestPresenter, req *model.Request) ui.Element {
	tk := uitheme.TokensOf(c)

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

	return ui.Column(c).FillWidth().Gap(24).Children(func() {
		// 1. Query Parameters Section
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
				uitheme.PanelHeader(c, "Query Parameters")
				if len(queryRows) > 0 {
					uitheme.DataText(c, fmt.Sprintf("(%d)", len(queryRows))).
						FontSize(11).
						TextColor(tk.TextTertiary)
				}
			})

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

		// Distinct horizontal divider between Query and Path parameters
		ui.Box(c).Height(1).FillWidth().Background(tk.BorderSecondary).Margin(8, 0)

		// 2. Path Parameters Section
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
				uitheme.PanelHeader(c, "Path Parameters")
				if len(pathRows) > 0 {
					uitheme.DataText(c, fmt.Sprintf("(%d)", len(pathRows))).
						FontSize(11).
						TextColor(tk.TextTertiary)
				}
			})

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
