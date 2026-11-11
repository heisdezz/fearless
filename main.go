// Command fearless is the application entry point.
//
// This file is wiring only: it creates the window and hands control to MyGo.
// All behaviour lives behind the app and core layers, never here.
//
// See docs/progress/packages.md (package UI-APP) for the architecture.
package main

import (
	"github.com/egoist/mygo"

	"fearless/internal/ui/app"
)

func main() {
	a := app.New()
	mygo.App.WhenReady(func() {
		a.CreateWindow()
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}
