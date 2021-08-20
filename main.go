package main

import (
	"image/color"
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
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

	// Variables for incrementing progress bar.
	var boiling bool
	var progress float32

	// Create ticker for boiler that is created stopped.
	const boilDuration time.Duration = time.Second * 5
	const boilTickerDuration time.Duration = time.Second / 25
	boilTicker := time.NewTicker(boilTickerDuration)
	boilTicker.Stop()

	// If we click the start button we toggle boiling the egg.
	var startButton widget.Clickable

	// Use the builting material UI theme.
	theme := material.NewTheme(gofont.Collection())

	// listen for events in the window.
	for {
		select {
		case e := <-w.Events():
			// Frame event.
			if fe, ok := e.(system.FrameEvent); ok {
				// Handle start button clicks here.
				if startButton.Clicked() {
					if !boiling {
						boilTicker.Reset(boilTickerDuration)
					} else {
						boilTicker.Stop()
					}
					boiling = !boiling
				}

				// Define flex layout with the following options.
				flex := layout.Flex{
					Axis:    layout.Vertical,
					Spacing: layout.SpaceStart,
				}

				progressBar := func(gtx layout.Context) layout.Dimensions {
					// Get a progress bar from the theme.
					bar := material.ProgressBar(theme, progress)
					// Return layout of bar after drawing.
					return bar.Layout(gtx)
				}

				startButtonStyled := func(gtx layout.Context) layout.Dimensions {
					// Create a margin inside the flex layout.
					margin := layout.Inset{
						Top:    unit.Dp(25),
						Bottom: unit.Dp(25),
						Right:  unit.Dp(35),
						Left:   unit.Dp(35),
					}

					// Add button to the flex layout.
					return margin.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							// Default state is to start boil else try to stop the boiling.
							btnState := "Start"
							if boiling {
								btnState = "Stop"
							}

							// Style the button according to the theme to get a styled button back.
							btn := material.Button(theme, &startButton, btnState)

							// Add operations to the graphical context to draw the button.
							return btn.Layout(gtx)
						},
					)
				}

				// Reverse rendering order to figure out the size of the egg widget.
				var eggWidget layout.Widget
				{
					var ops op.Ops
					gtx := layout.NewContext(&ops, fe)
					flex.Layout(gtx,
						// Add a ProgressBar.
						layout.Rigid(progressBar),
						// Add a button with margins.
						layout.Rigid(startButtonStyled),
						// Add an egg.
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							// Get the size of the egg widget and create the egg widget.
							eggDims := CreateEggWidget(nil)(gtx)
							eggWidget = CreateEggWidget(&layout.Constraints{
								Max: eggDims.Size,
							})
							return eggDims
						}),
					)
				}

				// Create graphical context that contains all the UI operations and the
				// frame event that triggered them.
				gtx := layout.NewContext(&ops, fe)

				flex.Layout(gtx,
					// Add an egg.
					layout.Rigid(eggWidget),
					// Add a ProgressBar.
					layout.Rigid(progressBar),
					// Add a button with margins.
					layout.Rigid(startButtonStyled),
				)

				// Add the list of operations to the frame event.
				fe.Frame(gtx.Ops)
			}

			// Window is destroyed.
			if de, ok := e.(system.DestroyEvent); ok {
				return de.Err
			}

		case <-boilTicker.C:
			// Increment progress by the total boil time divided by the tick duration.
			progress += (float32(boilTickerDuration) / float32(boilDuration))
			w.Invalidate()
		}
	}
}

func CreateEggWidget(constraints *layout.Constraints) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		// Set the constraints to the graphical context constrains if nil.
		if constraints == nil {
			constraints = &gtx.Constraints
		}

		// Calculate the center and the radius of the circle.
		center := constraints.Max.Div(2)
		r := center.Y
		if center.X < center.Y {
			r = center.X
		}

		circle := clip.Circle{
			// // Have circle start at bottom.
			// Center: f32.Point{X: float32(center.X), Y: float32(2*center.Y - r)},

			// Have circle centered.
			Center: f32.Point{X: float32(center.X), Y: float32(center.Y)},
			Radius: float32(r),
		}

		// Paint the circle red with the current graphical context.
		circleColor := color.NRGBA{R: 255, A: 255}
		paint.FillShape(gtx.Ops, circleColor, circle.Op(gtx.Ops))

		// Return the dimensions that were drawn.
		return layout.Dimensions{Size: constraints.Max}
	}

}
