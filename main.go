// Command fearless is the application entry point.
//
// This file is wiring only: it starts MyGo and hands control to it. All
// behaviour lives behind the app and core layers, never here. The window,
// menu and root view are UI-APP's (internal/ui/app); this main just runs the
// application, which UI-APP's acceptance requires to happen in main.
package main

import (
	"github.com/egoist/mygo"

	"fearless/internal/app/bus"
	"fearless/internal/app/commands"
	"fearless/internal/ui/app"
)

func main() {
	mygo.App.WhenReady(open)

	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}

// open builds the one service the window needs before it can persist anything —
// preferences — and hands it to the app.
//
// Preferences are the app's memory of where work is saved. Without them the
// first-run prompt would ask for a folder, take it, and forget it on the next
// launch, so they are wired here rather than left to a default.
func open() {
	b := bus.New()
	defer b.Close()
	store := bus.NewStore(nil, b)
	defer store.Close()

	prefs := commands.NewPreferencesService(store, b, "", nil)
	app.Open(app.WithPreferencesService(prefs))
}
