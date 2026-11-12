package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// CreateWindowOptions returns the resolved WindowOptions for the fearless main window,
// merging any user overrides with standard defaults.
//
// The window has no native chrome. The desktop's title bar and its menu bar
// are the window manager's furniture, not the app's: the app draws its own
// title bar — the 30 DIP strip the layout composes, with the window's
// controls drawn in it (layout.TitleBarView) — and the app's own menu opens
// from that strip's menu button. A GTK title bar above a MyGo title strip is
// two title bars, and the native menu bar under it a third piece of chrome
// the app does not need.
//
// Frameless is the option that removes the window manager's decoration on
// every desktop; the toolkit's hidden-title-bar mode keeps the compositor's
// title bar on the desktops that decorate windows themselves (KWin), which
// would leave the app with exactly the chrome this window drops.
func (a *App) CreateWindowOptions(opts ...mygo.WindowOptions) mygo.WindowOptions {
	opt := mygo.WindowOptions{
		Title:    "fearless",
		Width:    1280,
		Height:   800,
		StateKey: "fearless.main",
		Content:  ui.View(a.RootView),

		// The app's own title strip is the window's title bar.
		Frameless: true,
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
		if userOpt.TitleBarStyle != "" {
			opt.TitleBarStyle = userOpt.TitleBarStyle
		}
		if userOpt.TitleBarHeight > 0 {
			opt.TitleBarHeight = userOpt.TitleBarHeight
		}
	}

	return opt
}

// CreateWindow creates and shows the fearless main window with the root view
// and the application menu.
func (a *App) CreateWindow(opts ...mygo.WindowOptions) *mygo.Window {
	opt := a.CreateWindowOptions(opts...)
	win := mygo.NewWindow(opt)

	a.mu.Lock()
	a.Window = win
	a.mu.Unlock()

	// The menu is the app's, opened from the title strip's menu button. It
	// is not attached to the window: a native menu bar is chrome the app
	// does not draw, and the toolkit installs a default one unless the
	// application menu is explicitly removed.
	mygo.App.SetMenu(nil)

	return win
}

// Open creates a default App instance and initializes the main application window.
func Open() {
	a := New()
	a.CreateWindow()
}

// Run schedules CreateWindow in App.WhenReady and hands control to mygo.App.Run().
func (a *App) Run() error {
	mygo.App.WhenReady(func() {
		a.CreateWindow()
	})
	return mygo.App.Run()
}
