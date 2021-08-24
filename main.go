package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// D is a type alias for layout.Dimensions
type D = layout.Dimensions

// C is a type alias for layout.Constraints
type C = layout.Constraints

func main() {
	go func() {
		// create new window.
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

const boilTickerFreq time.Duration = time.Second / 25

func draw(w *app.Window) error {
	// Variable that stores UI operations
	var ops op.Ops

	// Create an egg widget with a boilTimer.
	boilTimer := NewBoilTicker(boilTickerFreq, 0)
	eggWidget := NewEggWidget(boilTimer, 1)
	defer eggWidget.Close()

	for {
		select {
		case e := <-w.Events():
			// Frame event.
			if fe, ok := e.(system.FrameEvent); ok {
				// Create graphical context that contains all the UI operations and the
				// frame event that triggered them.
				gtx := layout.NewContext(&ops, fe)

				// Draw and handle updates to egg widget.
				eggWidget.Layout(gtx)

				// Add the list of operations to the frame event.
				fe.Frame(gtx.Ops)
			}

			// Window is destroyed.
			if de, ok := e.(system.DestroyEvent); ok {
				return de.Err
			}

		// Capture state events and update.
		case state := <-eggWidget.boilTicker.C():
			eggWidget.boilTimerState = state
			w.Invalidate()
		}
	}
}

// NewEggWidget with a boil ticker and specified boil ticker.
func NewEggWidget(boilTimer *BoilTicker, precision int) *EggWidget {
	return &EggWidget{
		boilTicker:  boilTimer,
		startButton: &widget.Clickable{},
		boilDurationInput: widget.Editor{
			Alignment:  text.Middle,
			SingleLine: true,
		},
		theme: material.NewTheme(gofont.Collection()),
	}
}

// EggWidget is a ui widget that has a button and a time that can be set to boil
// an egg.
type EggWidget struct {
	boilTicker        *BoilTicker
	boilTimerState    BoilTickerState
	startButton       *widget.Clickable
	boilDurationInput widget.Editor
	boilPrecision     int
	theme             *material.Theme
}

// Close EggWidget.
func (e *EggWidget) Close() {
	e.boilTicker.Close()
}

// Widget for inputing the boil duration.
const boilDurationPrecision int = 1

// Layout draw and handles events on the egg widget.
func (e *EggWidget) Layout(gtx layout.Context) D {
	//////////////////////////////////////////////////////////////////////////////
	//                                  State                                   //
	//////////////////////////////////////////////////////////////////////////////

	// Handle start button clicks here.
	if e.startButton.Clicked() {
		// Read from the input box
		inputString := e.boilDurationInput.Text()
		inputString = strings.TrimSpace(inputString)
		inputFloat, _ := strconv.ParseFloat(inputString, 32)

		// Check if the output of the ticker has changed significantly
		boilRemain := float64(e.boilTicker.BoilRemain(e.boilTimerState)) / float64(time.Second)
		if math.Abs(boilRemain-inputFloat) > math.Pow10(-boilDurationPrecision) {
			e.boilTimerState.duration = time.Duration(inputFloat * float64(time.Second))
			e.boilTimerState = e.boilTicker.Do(BoilTimerSignalReset, e.boilTimerState)
		} else {
			// Handle ticker.
			signal := BoilTimerSignalStart
			if e.boilTimerState.boiling {
				signal = BoilTimerSignalStop
			}
			if e.boilTimerState.duration > 0 {
				e.boilTimerState = e.boilTicker.Do(signal, e.boilTimerState)
			}
		}
	}

	//////////////////////////////////////////////////////////////////////////////
	//                                  Layout                                  //
	//////////////////////////////////////////////////////////////////////////////

	// Alias boiler state progress.
	progress := e.boilTimerState.progress

	// Define flex layout for egg widget with the following options.
	var EggWidgetFlex = layout.Flex{
		Axis:    layout.Vertical,
		Spacing: layout.SpaceStart,
	}

	// ProgressBar widget.
	progressBar := func(gtx layout.Context) D {
		// Get a progress bar from the theme.
		bar := material.ProgressBar(e.theme, float32(progress))
		// Return layout of bar after drawing.
		return bar.Layout(gtx)
	}

	// Start button widget.
	startButtonStyled := func(gtx layout.Context) D {
		// Create a margin inside the flex layout.
		margin := layout.Inset{
			Top:    unit.Dp(25),
			Bottom: unit.Dp(25),
			Right:  unit.Dp(35),
			Left:   unit.Dp(35),
		}

		// Add button to the flex layout.
		return margin.Layout(gtx,
			func(gtx layout.Context) D {
				// Default state is to start boil else try to stop the boiling.
				btnState := "Start"
				if e.boilTimerState.boiling {
					btnState = "Stop"
				}
				if (progress >= 1) && (e.boilTimerState.duration != 0) {
					btnState = "Finished"
				}

				// Style the button according to the theme to get a styled button back.
				btn := material.Button(e.theme, e.startButton, btnState)

				// Add operations to the graphical context to draw the button.
				return btn.Layout(gtx)
			},
		)
	}

	// Boil duration input widget.
	boilDurationInputWidget := func(gtx layout.Context) D {
		// Create a styled editor.
		ed := material.Editor(e.theme, &e.boilDurationInput, "sec")

		// If boiling set the text of the editor to the remaining time in seconds.
		if e.boilTimerState.boiling && progress < 1 {
			boilRemain := float64(e.boilTicker.BoilRemain(e.boilTimerState)) / float64(time.Second)
			// Format to 1 decimal.
			precisionStr := fmt.Sprintf("%%.%df", boilDurationPrecision)
			boilRemainStr := fmt.Sprintf(precisionStr, boilRemain)
			// Update the text in the inputbox
			e.boilDurationInput.SetText(boilRemainStr)
		}

		// Calculate the margins for a centered button with size of 50.
		minBtnSize := unit.Dp(50)
		marginHz := unit.Add(gtx.Metric,
			minBtnSize.Scale(-1), unit.Px(float32(gtx.Constraints.Max.X))).Scale(0.5)

		// Create the margins for the text input.
		margins := layout.Inset{
			Top:    unit.Dp(20),
			Bottom: unit.Dp(20),
			Right:  marginHz,
			Left:   marginHz,
		}
		return margins.Layout(gtx,
			func(gtx layout.Context) D {

				// Create a border around the text input.
				border := widget.Border{
					Color:        color.NRGBA{R: 0, G: 200, B: 125, A: 200},
					CornerRadius: unit.Dp(4),
					Width:        unit.Dp(2),
				}
				return border.Layout(gtx, ed.Layout)
			},
		)
	}

	// Function that creates the egg widget given intial constraints.
	CreateEggWidget := func(constraints C) layout.Widget {
		return func(gtx layout.Context) D {
			// Set the constraints of the graphical context.
			gtx.Constraints = constraints
			dims := D{Size: gtx.Constraints.Max}

			// If the context queue is empty just return the size of the element.
			if gtx.Queue == nil {
				return dims
			}

			// Draw egg path.
			var eggPath clip.Path
			func() {
				// Find the center of the layout and start drawing the egg from there.
				center := gtx.Constraints.Max.Div(2)
				centerF32 := f32.Pt(float32(center.X), float32(center.Y))
				op.Offset(centerF32).Add(gtx.Ops)

				// Begin the path and close it when function exits.
				eggPath.Begin(gtx.Ops)
				defer eggPath.Close()

				// Calculate the center and the radius of the circle.
				r := float64(center.Y)
				if center.X < center.Y {
					r = float64(center.X)
				}

				// Constants that relate a to the other variables
				const (
					bDivA = (15.0 / 11.0)
					dDivA = (2.0 / 11.0)
				)

				// 'a' radius is related in this way to the radius of the circle.
				a := r / (bDivA + dDivA)

				// Egg paramters: 'b' radius and 'd' offset.
				var (
					b = a * bDivA
					d = a * dDivA
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

			// Complete the area for the egg path.
			eggArea := clip.Outline{Path: eggPath.End()}.Op()

			// Fill the shape
			color := color.NRGBA{
				R: 255,
				G: uint8(239 * (1 - progress)),
				B: uint8(174 * (1 - progress)),
				A: 255,
			}
			paint.FillShape(gtx.Ops, color, eggArea)

			return dims
		}
	}

	// Reverse rendering order to figure out the size of the egg widget.
	var eggWidget layout.Widget
	{
		// Copy graphical context but remove the queue so that element aren't
		// actually drawn.
		gtxRev := gtx
		gtxRev.Queue = nil
		gtxRev.Ops = &op.Ops{}

		// Layout in order of widget dependence e.g. progressBar depends on
		// startButtonStyled etc.
		EggWidgetFlex.Layout(gtxRev,
			// Add a ProgressBar.
			layout.Rigid(progressBar),
			// Add a button with margins.
			layout.Rigid(startButtonStyled),
			// Add a boil duration input widget.
			layout.Rigid(boilDurationInputWidget),
			// Add an egg.
			layout.Rigid(func(gtx layout.Context) D {
				eggWidget = CreateEggWidget(gtx.Constraints)
				return D{Size: gtx.Constraints.Max}
			}),
		)
	}

	// Layout in order of widgets from top to bottom.
	return EggWidgetFlex.Layout(gtx,
		// Add an egg.
		layout.Rigid(eggWidget),
		// Add a boil duration input widget.
		layout.Rigid(boilDurationInputWidget),
		// Add a ProgressBar.
		layout.Rigid(progressBar),
		// Add a button with margins.
		layout.Rigid(startButtonStyled),
	)
}

// BoilTickerState is a struct for keeping storing data relavent to the the state
// of a BoilTimer
type BoilTickerState struct {
	boiling  bool
	duration time.Duration
	progress float64
}

// NewBoilTicker creates a boil ticker with the given duration and freqency.
// The timer starts off in a stopped state.
func NewBoilTicker(freq, duration time.Duration) *BoilTicker {
	bt := BoilTicker{
		freq:   freq,
		ticker: time.NewTicker(freq),
		state: BoilTickerState{
			boiling:  false,
			duration: duration,
			progress: 0.0,
		},
		c:      make(chan BoilTickerState),
		action: make(chan BoilTimerStateSignal),
		closer: make(chan struct{}),
	}

	bt.ticker.Stop()

	// Main ticker loop.
	go func() {
		// Proccess events from timer and sets on state until closed.
		var done bool
		for !done {
			select {
			// When internal ticker ticks increment progress and report state.
			case <-bt.ticker.C:
				// Increment progress by the total boil time divided by the tick duration.
				if bt.state.progress < 1 {
					bt.state.progress += (float64(bt.freq) / float64(bt.state.duration))
					if bt.state.duration == 0 {
						bt.state.progress = 0
					}
					bt.c <- bt.state
				}

			// Update and report back state.
			case action := <-bt.action:
				signal := action.Signal

				// Set the duration.
				bt.state = *action.State

				// Boiling false if stopping else true
				bt.state.boiling = (signal == BoilTimerSignalStart) || (signal == BoilTimerSignalRestart)

				// Stop the timer if boiling.
				if bt.state.boiling {
					bt.ticker.Reset(bt.freq)
				} else {
					bt.ticker.Stop()
				}

				// If restarting or resetting then progress goes to zero.
				if (signal == BoilTimerSignalReset) || (signal == BoilTimerSignalRestart) {
					bt.state.progress = 0.0
				}

				bt.c <- bt.state

			// Close boil ticker
			case <-bt.closer:
				done = true
			}
		}
		close(bt.c)
		close(bt.action)
	}()

	return &bt
}

// BoilTicker sends reports the state of the egg boil ticker at a specified
// frequency and duration.
type BoilTicker struct {
	freq   time.Duration
	ticker *time.Ticker
	state  BoilTickerState
	action chan BoilTimerStateSignal
	c      chan BoilTickerState
	closer chan struct{}
}

// BoilRemain calculates the time remaining boiling.
func (bt *BoilTicker) BoilRemain(state BoilTickerState) time.Duration {
	return time.Duration((1 - state.progress) * float64(state.duration))
}

// Do sends a particular signal with the attempt to update the ticker with the
// provided state.
func (bt *BoilTicker) Do(signal BoilTimerSignal, state BoilTickerState) BoilTickerState {
	go func() { bt.action <- BoilTimerStateSignal{signal, &state} }()
	return <-bt.c
}

// Get a receiver on the boil ticker's state.
func (bt *BoilTicker) C() <-chan BoilTickerState { return bt.c }

// Close the boil ticker.
func (bt *BoilTicker) Close() { bt.closer <- struct{}{} }

// BoilTimerStateSignal is a structure that is used to send messages from other
// go routines to the BoilTicker tick go routine.
type BoilTimerStateSignal struct {
	Signal BoilTimerSignal
	State  *BoilTickerState
}

// BoilTimerSignal is an enum for various type of updates to the BoilTicker's
// internal state.
type BoilTimerSignal int

func (b BoilTimerSignal) String() string {
	switch b {
	case BoilTimerSignalGet:
		return "Get"
	case BoilTimerSignalStop:
		return "Stop"
	case BoilTimerSignalStart:
		return "Start"
	case BoilTimerSignalReset:
		return "Reset"
	case BoilTimerSignalRestart:
		return "Restart"
	}
	panic("unreachable")
}

// Types of BoilTimerSignal's
const (
	BoilTimerSignalGet     BoilTimerSignal = iota // Get
	BoilTimerSignalStop                           // Stop
	BoilTimerSignalStart                          // Start
	BoilTimerSignalReset                          // Reset
	BoilTimerSignalRestart                        // Restart
)
