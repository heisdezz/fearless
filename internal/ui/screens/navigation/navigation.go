package navigation

import (
	"github.com/egoist/mygo/ui"
	lobe "github.com/lobehub/lobe-mygo/pkg/layout"
)

// Sidebar renders the navigation sidebar.
//
// The nav is resizable but not collapsible, so it always draws the expanded
// view. DraggableSideNav mounts this body inside a scrolling column, so the
// sidebar must size itself from its content: a percentage height
// (Fill/FillHeight) resolves against the scroll's unbounded height and
// collapses to zero.
func (s *Screen) Sidebar(c *ui.Context, nav lobe.SideNavState) ui.Element {
	return s.renderExpanded(c, s.Snapshot, s.Actions)
}

// Sidebar renders the navigation sidebar from an instantiated screen.
func Sidebar(c *ui.Context, s *Screen, nav lobe.SideNavState) ui.Element {
	if s == nil {
		s = &Screen{}
	}
	return s.Sidebar(c, nav)
}

// CollectionHeader renders the collection header strip from an instantiated screen.
func CollectionHeader(c *ui.Context, s *Screen) ui.Element {
	if s == nil {
		s = &Screen{}
	}
	return s.CollectionHeader(c)
}

// View renders the navigation screen: the sidebar with its section switcher.
func (s *Screen) View(c *ui.Context) {
	// The sidebar sizes itself by content, so the row pins it to the top
	// rather than centering it across the full height.
	ui.Row(c).FillHeight().Grow(1).AlignItems(ui.Start).Children(func() {
		s.Sidebar(c, lobe.SideNavState{})
	})
}
