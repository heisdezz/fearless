package request

import (
	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	uitheme "fearless/internal/ui/theme"
	"fearless/internal/ui/widgets"
)

// renderHeadersTab renders the request headers editor grid.
func renderHeadersTab(c *ui.Context, p *presenters.RequestPresenter, req *model.Request) ui.Element {
	return ui.Column(c).FillWidth().Gap(8).Children(func() {
		uitheme.PanelHeader(c, "Headers")

		widgets.KeyValueGrid(c, &req.Headers, widgets.KeyValueGridProps{
			ShowDescription: true,
			KeyPlaceholder:  "Header Name",
			ValPlaceholder:  "Header Value",
		})
	})
}
