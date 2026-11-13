package app

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"fearless/internal/core/model"
)

// TestTabSwitchFromEveryHitArea clicks the second tab at points across its
// width. A tab is a row drawn with children on top (method badge, title, close
// button); a click on any of them must still select the tab, since the row's
// own background is only the padding around them.
func TestTabSwitchFromEveryHitArea(t *testing.T) {
	build := func() (*App, *ui.Tester) {
		a := New()
		col := &model.Collection{
			UID: "c1", Name: "Cren",
			Items: []*model.Item{
				{UID: "r1", Name: "App Controller root", Type: model.ItemHTTPRequest,
					Request: &model.Request{Method: "GET", URL: "{{baseUrl}}/api"}},
				{UID: "r2", Name: "Content Controller get Galleries", Type: model.ItemHTTPRequest,
					Request: &model.Request{Method: "GET", URL: "{{baseUrl}}/api/get/galleries"}},
			},
		}
		a.CollectionPresenter.SetCollections([]*model.Collection{col})
		a.EnvironmentPresenter.SetCollection(col)

		a.TabsPresenter.OpenTab(col.Items[0])
		a.TabsPresenter.OpenTab(col.Items[1])
		a.TabsPresenter.SelectTab(a.TabsPresenter.Tabs()[0].UID)
		a.RequestPresenter.SetItem(col.Items[0])

		tt := ui.NewTester(func(c *ui.Context) {
			a.ShellScreen.RequestPane(c)
		}, 900, 400)
		tt.Frame()
		return a, tt
	}

	// Locate the second tab by its title, then probe left (badge) and middle
	// (title).
	for _, probe := range []struct {
		name string
		dx   float32
	}{
		{"badge (left)", -float32(len("Content Controller get Galleries")) - 6},
		{"title (middle)", 0},
	} {
		probe := probe
		t.Run(probe.name, func(t *testing.T) {
			a, tt := build()
			title, ok := tt.Find("Content Controller get Galleries")
			if !ok {
				t.Fatalf("second tab not found")
			}
			x := title.X + title.W/2 + probe.dx
			tt.ClickAt(x, title.Y+title.H/2)
			tt.Frame()

			if item := a.RequestPresenter.Item(); item == nil || item.UID != "r2" {
				t.Errorf("clicking the %s selected %v, want r2", probe.name, item)
			}
		})
	}
}
