package main

import (
	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func main() {
	go func() {
		// create new window
		w := app.NewWindow(
			app.Title("Example Gio UI"),
			app.Size(unit.Dp(600), unit.Dp(400)),
		)

		// Variable that stores UI operations
		var ops op.Ops

		var startButton widget.Clickable

		// Use the builting material UI theme.
		theme := material.NewTheme(gofont.Collection())

		// listen for events in the window.
		for e := range w.Events() {
			// Frame event.
			if fe, ok := e.(system.FrameEvent); ok {
				// Create graphical context that contains all the UI operations and the
				// frame event that triggered them.
				gtx := layout.NewContext(&ops, fe)

				// Style the button according to the theme to get a styled button back.
				btn := material.Button(theme, &startButton, "Start")

				// Add operations to the graphical context to draw the button.
				btn.Layout(gtx)

				// Add the list of operations to the frame event.
				fe.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}
