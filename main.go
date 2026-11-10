// Command fearless is the application entry point.
//
// This file is wiring only: it creates the window and hands control to MyGo.
// All behaviour lives behind the app and core layers, never here.
//
// See docs/progress/packages.md (package UI-APP) for the intended shape.
package main

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "fearless",
			Width:   1280,
			Height:  800,
			Content: ui.View(rootView),
		})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
}

// rootView is a placeholder until UI-APP lands. It exists so the module builds
// and so the MyGo + Lobe MyGo stack is proven end to end from main.
func rootView(c *ui.Context) {
	theme.Apply(c)
	tokens := theme.TokensOf(c)
	ui.Column(c).Fill().Center().Gap(8).Background(tokens.BgLayout).Children(func() {
		ui.Text(c, "fearless").TextColor(tokens.TextHeading)
		ui.Text(c, fmt.Sprintf("host: %s", "not implemented yet")).TextColor(tokens.TextSecondary)
	})
}
