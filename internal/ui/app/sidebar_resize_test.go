package app

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"

	"fearless/internal/core/model"
	"fearless/internal/ui/layout"
	"fearless/internal/ui/presenters"
	"fearless/internal/ui/screens/navigation"
)

// TestSidebarContentTracksNavWidth is the resize regression: the sidebar body
// must fill the width DraggableSideNav gives it. It used to pin itself to the
// snapshot width, which is only refreshed after a drag, so dragging the seam
// widened the container while the inner rows stayed at their old size.
//
// The toolbar's right-aligned "Open" button is the probe: its x follows the
// inner width.
func TestSidebarContentTracksNavWidth(t *testing.T) {
	openX := func(navWidth float32) float32 {
		st := layout.DefaultLayoutState()
		st.SidebarWidth = navWidth

		col := &model.Collection{
			UID:  "col-1",
			Name: "Tracking",
			Items: []*model.Item{{
				UID:     "req-1",
				Name:    "Tracked",
				Type:    model.ItemHTTPRequest,
				Request: &model.Request{Method: "GET", URL: "https://api.test"},
			}},
		}
		colPres := presenters.NewCollectionPresenter(nil, nil, nil, nil)
		colPres.SetCollections([]*model.Collection{col})
		envPres := presenters.NewEnvironmentPresenter(nil, nil, nil, nil)
		nav := navigation.NewScreen(presenters.NewSidebarPresenter(colPres, envPres), navigation.Actions{})

		root := &layout.RootView{
			State:       &st,
			Budget:      layout.DefaultRegionBudget(),
			TitleBar:    func(c *ui.Context) { ui.Text(c, "TITLE") },
			Sidebar:     func(c *ui.Context, s lobe.SideNavState) { nav.Sidebar(c, s) },
			RequestPane: func(c *ui.Context) { ui.Text(c, "REQUEST") },
		}
		tt := ui.NewTester(root.View, 1200, 800)
		tt.Frame()
		tt.Advance(600 * time.Millisecond)
		tt.Frame()

		open, ok := tt.Find("Open")
		if !ok {
			t.Fatalf("toolbar not rendered at nav width %.0f", navWidth)
		}
		return open.X
	}

	narrow := openX(240)
	wide := openX(420)

	// The nav is 180 DIPs wider; the content must move with it.
	if wide-narrow < 100 {
		t.Fatalf("sidebar content did not follow the nav width: \"Open\" at x=%.1f "+
			"for a 240-wide nav and x=%.1f for a 420-wide one — the body is pinned "+
			"to a stale width", narrow, wide)
	}
}
