package navigation

import (
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"

	"fearless/internal/ui/presenters"
)

// buildNodeMenuItems builds the context menu items for a tree node.
func (s *Screen) buildNodeMenuItems(n *presenters.TreeNode) []overlay.MenuItem {
	if n == nil {
		return nil
	}
	if n.IsFolder {
		if n.Level == 0 {
			// Top-level collection
			return []overlay.MenuItem{
				{Key: "new_request", Label: "New Request", Icon: lucide.FilePlus()},
				{Key: "new_folder", Label: "New Folder", Icon: lucide.FolderPlus()},
				{Key: "new_app", Label: "New App", Icon: lucide.LayoutGrid()},
				{Key: "new_js", Label: "New JS File", Icon: lucide.FileCode()},
				{Key: "run", Label: "Run", Icon: lucide.Play()},
				{Key: "clone", Label: "Clone", Icon: lucide.Copy()},
				{Key: "openapi", Label: "OpenAPI", Icon: lucide.Compass()},
				{Key: "rename", Label: "Rename", Icon: lucide.Pencil()},
				{Key: "share", Label: "Share", Icon: lucide.Share2()},
				{Key: "generate_docs", Label: "Generate Docs", Icon: lucide.BookOpen()},
				{Key: "collapse", Label: "Collapse", Icon: lucide.FoldVertical()},
				{Key: "reveal", Label: "Reveal in File Manager", Icon: lucide.Folder()},
				{Divider: true},
				{Key: "settings", Label: "Settings", Icon: lucide.Settings()},
				{Key: "terminal", Label: "Open in Terminal", Icon: lucide.Terminal()},
				{Key: "remove", Label: "Remove", Icon: lucide.X(), Danger: true},
			}
		}
		// Subfolder
		return []overlay.MenuItem{
			{Key: "new_request", Label: "New Request", Icon: lucide.FilePlus()},
			{Key: "new_folder", Label: "New Folder", Icon: lucide.FolderPlus()},
			{Key: "clone", Label: "Clone", Icon: lucide.Copy()},
			{Key: "rename", Label: "Rename", Icon: lucide.Pencil()},
			{Key: "collapse", Label: "Collapse", Icon: lucide.FoldVertical()},
			{Key: "reveal", Label: "Reveal in File Manager", Icon: lucide.Folder()},
			{Divider: true},
			{Key: "terminal", Label: "Open in Terminal", Icon: lucide.Terminal()},
			{Key: "remove", Label: "Delete", Icon: lucide.X(), Danger: true},
		}
	}
	// Request item
	return []overlay.MenuItem{
		{Key: "clone", Label: "Clone", Icon: lucide.Copy()},
		{Key: "rename", Label: "Rename", Icon: lucide.Pencil()},
		{Key: "remove", Label: "Delete", Icon: lucide.X(), Danger: true},
	}
}

// handleMenuItem dispatches a menu item key to the corresponding action.
func (s *Screen) handleMenuItem(key string, nodeUID string, actions Actions) {
	s.OpenMenuUID = ""
	switch key {
	case "new_request":
		if actions.OnCreateRequest != nil {
			actions.OnCreateRequest(nodeUID)
		}
	case "new_folder":
		if actions.OnCreateFolder != nil {
			actions.OnCreateFolder(nodeUID)
		}
	case "run":
		if actions.OnRunCollection != nil {
			actions.OnRunCollection(nodeUID)
		}
	case "clone":
		if actions.OnCloneCollection != nil {
			actions.OnCloneCollection(nodeUID)
		}
	case "rename":
		if actions.OnRenameItem != nil {
			actions.OnRenameItem(nodeUID)
		}
	case "collapse":
		if actions.OnCollapseAll != nil {
			actions.OnCollapseAll(nodeUID)
		}
	case "reveal":
		if actions.OnRevealItem != nil {
			actions.OnRevealItem(nodeUID)
		}
	case "settings":
		if actions.OnCollectionSettings != nil {
			actions.OnCollectionSettings(nodeUID)
		}
	case "terminal":
		if actions.OnOpenTerminal != nil {
			actions.OnOpenTerminal(nodeUID)
		}
	case "remove":
		if actions.OnDeleteItem != nil {
			actions.OnDeleteItem(nodeUID)
		}
	}
}
