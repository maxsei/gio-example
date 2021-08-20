package main

import (
	"image/color"
	"log"
	"math"
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

var progress float32

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
	// var progress float32

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
							eggWidget = CreateEggWidget(gtx.Constraints)
							return layout.Dimensions{Size: gtx.Constraints.Max}
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
			if progress < 1 {
				progress += (float32(boilTickerDuration) / float32(boilDuration))
			}
			w.Invalidate()
		}
	}
}

func CreateEggWidget(constraints layout.Constraints) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = constraints

		center := gtx.Constraints.Max.Div(2)
		centerF32 := f32.Pt(float32(center.X), float32(center.Y))
		op.Offset(centerF32).Add(gtx.Ops)

		// Calculate the center and the radius of the circle.
		a := float64(center.Y)
		if center.X < center.Y {
			a = float64(center.X)
		}
		a /= (17.0 / 11.0)

		// Draw egg path.
		var eggPath clip.Path
		// eggPath.Move(centerF32)
		func() {
			// Begin the path and close it when function exits.
			eggPath.Begin(gtx.Ops)
			defer eggPath.Close()

			// Egg constants.
			var (
				b = a * (15.0 / 11.0)
				d = a * (2.0 / 11.0)
			)

			// Rotate from 0 to 360 degrees.
			for deg := 0; deg < 360; deg++ {
				rad := (float64(deg) / 360) * 2 * math.Pi
				// Trig gives the distance in X and Y direction
				cosT := math.Cos(rad)
				sinT := math.Sin(rad)
				// The x/y coordinates
				x := a * cosT
				y := -(math.Sqrt(b*b-d*d*cosT*cosT) + d*sinT) * sinT
				y += d
				// Finally the point on the outline
				p := f32.Pt(float32(x), float32(y))

				// If its the first time drawing move to the point else draw line.
				if deg == 0 {
					eggPath.MoveTo(p)
					continue
				}
				// Draw the line to this point
				eggPath.LineTo(p)
			}
		}()

		eggArea := clip.Outline{Path: eggPath.End()}.Op()
		// Fill the shape
		color := color.NRGBA{
			R: 255,
			G: uint8(239 * (1 - progress)),
			B: uint8(174 * (1 - progress)),
			A: 255,
		}

		paint.FillShape(gtx.Ops, color, eggArea)
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}
