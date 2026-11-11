// Command fearless is the application entry point.
//
// This file is wiring only: it starts MyGo and hands control to it. All
// behaviour lives behind the app and core layers, never here. The window,
// menu and root view are UI-APP's (internal/ui/app); this main just runs the
// application, which UI-APP's acceptance requires to happen in main.
package main

import (
	"github.com/egoist/mygo"

	"fearless/internal/ui/app"
)

func main() {
	mygo.App.WhenReady(app.Open)
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
