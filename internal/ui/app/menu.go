package app

import (
	"context"

	"github.com/egoist/mygo"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
)

// MenuActions collects callbacks triggered from application and window menus.
type MenuActions struct {
	OnNewRequest       func()
	OnNewCollection    func()
	OnImportCollection func()
	OnSave             func()
	OnCloseTab         func()
	OnPreferences      func()
	OnQuit             func()
	OnSendRequest      func()
	OnCancelRequest    func()
	OnToggleSidebar    func()
	OnChangeLayout     func()
	OnToggleConsole    func()
	OnZoomIn           func()
	OnZoomOut          func()
	OnResetZoom        func()
	OnToggleFullScreen func()
	OnAbout            func()
	OnDocumentation    func()
}

// MenuActions returns the bound actions for window and application menus.
func (a *App) MenuActions() MenuActions {
	return MenuActions{
		OnNewRequest: func() {
			a.NewRequest("")
		},
		OnNewCollection: func() {
			a.CreateCollection()
		},
		OnImportCollection: func() {
			a.ShowImport()
		},
		OnSave: func() {
			a.SaveActiveRequest()
		},
		OnCloseTab: func() {
			a.CloseActiveTab()
		},
		OnPreferences: func() {
			a.ShowPreferences()
		},
		OnQuit: func() {
			mygo.App.Quit()
		},
		OnSendRequest: func() {
			a.SendActiveRequest()
		},
		OnCancelRequest: func() {
			a.CancelActiveRequest()
		},
		OnToggleSidebar: func() {
			a.ToggleSidebar()
		},
		OnChangeLayout: func() {
			a.ToggleLayout()
		},
		OnToggleConsole: func() {
			a.ToggleConsole()
		},
		OnAbout: func() {
			mygo.App.ShowAboutPanel(mygo.AboutPanelOptions{
				ApplicationName: "fearless",
				Version:         "1.0.0",
			})
		},
		OnDocumentation: func() {
			browser := a.Services.BrowserOr()
			_ = browser.Open(context.Background(), "https://github.com/usebruno/bruno")
		},
	}
}

func (a *App) buildLobeMenuItems() []overlay.MenuItem {
	return []overlay.MenuItem{
		{Key: "new-request", Label: "New Request", Icon: lucide.Plus(), Hotkey: "mod+n"},
		{Key: "new-collection", Label: "New Collection", Icon: lucide.FolderPlus(), Hotkey: "mod+shift+n"},
		{Key: "import-collection", Label: "Import Collection...", Icon: lucide.Upload(), Hotkey: "mod+o"},
		{Divider: true},
		{Key: "save", Label: "Save", Icon: lucide.Save(), Hotkey: "mod+s"},
		{Key: "close-tab", Label: "Close Tab", Icon: lucide.X(), Hotkey: "mod+w"},
		{Divider: true},
		{Key: "send-request", Label: "Send Request", Icon: lucide.Send(), Hotkey: "mod+enter"},
		{Key: "cancel-request", Label: "Cancel Request", Icon: lucide.Square(), Hotkey: "escape"},
		{Divider: true},
		{Key: "toggle-sidebar", Label: "Toggle Sidebar", Icon: lucide.Sidebar(), Hotkey: "mod+\\"},
		{Key: "toggle-console", Label: "Toggle Console", Icon: lucide.Terminal(), Hotkey: "mod+t"},
		{Key: "change-layout", Label: "Change Orientation", Icon: lucide.Columns(), Hotkey: "mod+j"},
		{Divider: true},
		{Key: "preferences", Label: "Preferences...", Icon: lucide.Settings(), Hotkey: "mod+,"},
		{Divider: true},
		{Key: "documentation", Label: "Documentation", Icon: lucide.BookOpen()},
		{Key: "about", Label: "About fearless", Icon: lucide.Info()},
		{Divider: true},
		{Key: "quit", Label: "Quit fearless", Icon: lucide.LogOut(), Danger: true, Hotkey: "mod+q"},
	}
}

func (a *App) handleMenuKey(key string) {
	actions := a.MenuActions()
	switch key {
	case "new-request":
		if actions.OnNewRequest != nil {
			actions.OnNewRequest()
		}
	case "new-collection":
		if actions.OnNewCollection != nil {
			actions.OnNewCollection()
		}
	case "import-collection":
		if actions.OnImportCollection != nil {
			actions.OnImportCollection()
		}
	case "save":
		if actions.OnSave != nil {
			actions.OnSave()
		}
	case "close-tab":
		if actions.OnCloseTab != nil {
			actions.OnCloseTab()
		}
	case "send-request":
		if actions.OnSendRequest != nil {
			actions.OnSendRequest()
		}
	case "cancel-request":
		if actions.OnCancelRequest != nil {
			actions.OnCancelRequest()
		}
	case "toggle-sidebar":
		if actions.OnToggleSidebar != nil {
			actions.OnToggleSidebar()
		}
	case "toggle-console":
		if actions.OnToggleConsole != nil {
			actions.OnToggleConsole()
		}
	case "change-layout":
		if actions.OnChangeLayout != nil {
			actions.OnChangeLayout()
		}
	case "preferences":
		if actions.OnPreferences != nil {
			actions.OnPreferences()
		}
	case "documentation":
		if actions.OnDocumentation != nil {
			actions.OnDocumentation()
		}
	case "about":
		if actions.OnAbout != nil {
			actions.OnAbout()
		}
	case "quit":
		if actions.OnQuit != nil {
			actions.OnQuit()
		}
	}
}

// BuildMenu constructs the standard fearless menu hierarchy:
// File, Edit, View, Request, and Help.
func BuildMenu(actions MenuActions, router *KeybindingRouter) *mygo.Menu {
	// 1. File Menu
	fileItems := []*mygo.MenuItem{
		{
			ID:          "new-request",
			Label:       "New Request",
			Accelerator: "CmdOrCtrl+N",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnNewRequest != nil {
					actions.OnNewRequest()
				}
			},
		},
		{
			ID:          "new-collection",
			Label:       "New Collection",
			Accelerator: "CmdOrCtrl+Shift+N",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnNewCollection != nil {
					actions.OnNewCollection()
				}
			},
		},
		{
			ID:          "import-collection",
			Label:       "Import Collection...",
			Accelerator: "CmdOrCtrl+O",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnImportCollection != nil {
					actions.OnImportCollection()
				}
			},
		},
		mygo.Separator(),
		{
			ID:          "save",
			Label:       "Save",
			Accelerator: "CmdOrCtrl+S",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnSave != nil {
					actions.OnSave()
				}
			},
		},
		{
			ID:          "close-tab",
			Label:       "Close Tab",
			Accelerator: "CmdOrCtrl+W",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnCloseTab != nil {
					actions.OnCloseTab()
				}
			},
		},
		mygo.Separator(),
		{
			ID:          "preferences",
			Label:       "Preferences...",
			Accelerator: "CmdOrCtrl+,",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnPreferences != nil {
					actions.OnPreferences()
				}
			},
		},
		mygo.Separator(),
		{
			ID:    "quit",
			Label: "Quit fearless",
			Role:  mygo.RoleQuit,
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnQuit != nil {
					actions.OnQuit()
				}
			},
		},
	}

	// 2. Edit Menu
	editItems := []*mygo.MenuItem{
		{Role: mygo.RoleUndo},
		{Role: mygo.RoleRedo},
		mygo.Separator(),
		{Role: mygo.RoleCut},
		{Role: mygo.RoleCopy},
		{Role: mygo.RolePaste},
		{Role: mygo.RoleSelectAll},
	}

	// 3. View Menu
	viewItems := []*mygo.MenuItem{
		{
			ID:          "toggle-sidebar",
			Label:       "Toggle Sidebar",
			Accelerator: "CmdOrCtrl+\\",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnToggleSidebar != nil {
					actions.OnToggleSidebar()
				}
			},
		},
		{
			ID:          "change-layout",
			Label:       "Change Orientation",
			Accelerator: "CmdOrCtrl+J",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnChangeLayout != nil {
					actions.OnChangeLayout()
				}
			},
		},
		{
			ID:          "toggle-console",
			Label:       "Toggle Console",
			Accelerator: "CmdOrCtrl+T",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnToggleConsole != nil {
					actions.OnToggleConsole()
				}
			},
		},
		mygo.Separator(),
		{
			Role: mygo.RoleResetZoom,
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnResetZoom != nil {
					actions.OnResetZoom()
				}
			},
		},
		{
			Role: mygo.RoleZoomIn,
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnZoomIn != nil {
					actions.OnZoomIn()
				}
			},
		},
		{
			Role: mygo.RoleZoomOut,
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnZoomOut != nil {
					actions.OnZoomOut()
				}
			},
		},
		mygo.Separator(),
		{
			Role: mygo.RoleToggleFullScreen,
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnToggleFullScreen != nil {
					actions.OnToggleFullScreen()
				}
			},
		},
	}

	// 4. Request Menu
	requestItems := []*mygo.MenuItem{
		{
			ID:          "send-request",
			Label:       "Send Request",
			Accelerator: "CmdOrCtrl+Enter",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnSendRequest != nil {
					actions.OnSendRequest()
				}
			},
		},
		{
			ID:          "cancel-request",
			Label:       "Cancel Request",
			Accelerator: "Escape",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnCancelRequest != nil {
					actions.OnCancelRequest()
				}
			},
		},
	}

	// 5. Help Menu
	helpItems := []*mygo.MenuItem{
		{
			ID:    "documentation",
			Label: "Documentation",
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnDocumentation != nil {
					actions.OnDocumentation()
				}
			},
		},
		mygo.Separator(),
		{
			ID:    "about",
			Label: "About fearless",
			Role:  mygo.RoleAbout,
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				if actions.OnAbout != nil {
					actions.OnAbout()
				}
			},
		},
	}

	topLevelItems := []*mygo.MenuItem{
		{
			Label:   "File",
			Submenu: fileItems,
		},
		{
			Label:   "Edit",
			Submenu: editItems,
		},
		{
			Label:   "View",
			Submenu: viewItems,
		},
		{
			Label:   "Request",
			Submenu: requestItems,
		},
		{
			Label:   "Help",
			Submenu: helpItems,
		},
	}

	return mygo.NewMenu(topLevelItems)
}
