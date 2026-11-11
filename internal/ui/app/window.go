package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// CreateWindowOptions returns the resolved WindowOptions for the fearless main window,
// merging any user overrides with standard defaults.
func (a *App) CreateWindowOptions(opts ...mygo.WindowOptions) mygo.WindowOptions {
	opt := mygo.WindowOptions{
		Title:    "fearless",
		Width:    1280,
		Height:   800,
		StateKey: "fearless.main",
		Content:  ui.View(a.RootView),
	}

	if len(opts) > 0 {
		userOpt := opts[0]
		if userOpt.Title != "" {
			opt.Title = userOpt.Title
		}
		if userOpt.Width > 0 {
			opt.Width = userOpt.Width
		}
		if userOpt.Height > 0 {
			opt.Height = userOpt.Height
		}
		if userOpt.StateKey != "" {
			opt.StateKey = userOpt.StateKey
		}
		if userOpt.Content != nil {
			opt.Content = userOpt.Content
		}
	}

	return opt
}

// CreateWindow creates and shows the fearless main window with the root view
// and top-level application menus attached.
func (a *App) CreateWindow(opts ...mygo.WindowOptions) *mygo.Window {
	opt := a.CreateWindowOptions(opts...)
	win := mygo.NewWindow(opt)

	a.mu.Lock()
	a.Window = win
	a.mu.Unlock()

	// Build and attach application menu
	menu := BuildMenu(a.MenuActions(), a.Router)
	win.SetMenu(menu)
	mygo.App.SetMenu(menu)

	return win
}

// Run schedules CreateWindow in App.WhenReady and hands control to mygo.App.Run().
func (a *App) Run() error {
	mygo.App.WhenReady(func() {
		a.CreateWindow()
	})
	return mygo.App.Run()
}
