package request

import (
	"github.com/egoist/mygo/ui"
	lobetheme "github.com/lobehub/lobe-mygo/pkg/theme"

	"fearless/internal/core/model"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/widgets"
)

// renderHeadersTab renders the request headers editor grid.
func renderHeadersTab(c *ui.Context, p *presenters.RequestPresenter, req *model.Request) ui.Element {
	tk := lobetheme.TokensOf(c)

	return ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, "Headers").
			FontSize(12).
			Bold().
			TextColor(tk.Text)

		widgets.KeyValueGrid(c, &req.Headers, widgets.KeyValueGridProps{
			ShowDescription: true,
			KeyPlaceholder:  "Header Name",
			ValPlaceholder:  "Header Value",
		})
	})
}
