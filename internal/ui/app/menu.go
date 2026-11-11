package app

import (
	"github.com/egoist/mygo"
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
	OnToggleConsole    func()
	OnZoomIn           func()
	OnZoomOut          func()
	OnResetZoom        func()
	OnToggleFullScreen func()
	OnAbout            func()
	OnDocumentation    func()
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
			ID:          "toggle-console",
			Label:       "Toggle Console",
			Accelerator: "CmdOrCtrl+J",
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
