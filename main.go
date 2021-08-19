package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// D is a type alias for layout.Dimensions
type D layout.Dimensions

// C is a type alias for layout.Constraints
type C layout.Constraints

func main() {
	go func() {
		// create new window
		w := app.NewWindow(
			app.Title("Example Gio UI"),
			app.Size(unit.Dp(600), unit.Dp(400)),
		)

		if err := draw(w); err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
	}()
	app.Main()
}

func draw(w *app.Window) error {

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

			// Define flex layout with the following options.
			layout.Flex{
				Axis:    layout.Vertical,
				Spacing: layout.SpaceStart,
			}.Layout(gtx,
				// Define the children of the flex layout which in this case is just
				// button widget. Widget's are just functions of a graphical context
				// that return their dimensions.
				layout.Rigid(
					func(gtx layout.Context) layout.Dimensions {
						// Style the button according to the theme to get a styled button back.
						btn := material.Button(theme, &startButton, "Start")

						// Add operations to the graphical context to draw the button.
						return btn.Layout(gtx)
					},
				),
				// Add an empty space the bottom of the screen.
				layout.Rigid(
					layout.Spacer{Height: unit.Dp(25)}.Layout,
				),
			)

			// Add the list of operations to the frame event.
			fe.Frame(gtx.Ops)
		}

		// Window is destroyed.
		if de, ok := e.(system.DestroyEvent); ok {
			return de.Err
		}
	}

	// This code shouln't be reached.
	panic("not reachable")
}
