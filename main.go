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

	const boilTickerFreq time.Duration = time.Second / 25
	boilTimer := NewBoilTimer(boilTickerFreq, 0)
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

				eggWidget.Layout(gtx)

				// Add the list of operations to the frame event.
				fe.Frame(gtx.Ops)
			}

			// Window is destroyed.
			if de, ok := e.(system.DestroyEvent); ok {
				return de.Err
			}

		case state := <-eggWidget.boilTicker.C():
			eggWidget.boilTimerState = state
			w.Invalidate()
		}
	}
}

func NewEggWidget(boilTimer *BoilTimer, precision int) *EggWidget {
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

type EggWidget struct {
	boilTicker        *BoilTimer
	boilTimerState    BoilTimerState
	startButton       *widget.Clickable
	boilDurationInput widget.Editor
	boilPrecision     int
	theme             *material.Theme
}

func (e *EggWidget) Close() {
	e.boilTicker.Close()
}

// Widget for inputing the boil duration.
const boilDurationPrecision int = 1

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

	// Only used for when predrawing the ui.
	var preDraw bool = false

	// Alias boiler state progress.
	progress := e.boilTimerState.progress

	// Define flex layout with the following options.
	flex := layout.Flex{
		Axis:    layout.Vertical,
		Spacing: layout.SpaceStart,
	}

	progressBar := func(gtx layout.Context) D {
		// Get a progress bar from the theme.
		bar := material.ProgressBar(e.theme, float32(progress))
		// Return layout of bar after drawing.
		return bar.Layout(gtx)
	}

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
		// Calculate the margins for a centered button with size of 50.
		minBtnSize := unit.Dp(50)
		marginHz := unit.Add(gtx.Metric,
			minBtnSize.Scale(-1), unit.Px(float32(gtx.Constraints.Max.X))).Scale(0.5)

		margins := layout.Inset{
			Top:    unit.Dp(20),
			Bottom: unit.Dp(20),
			Right:  marginHz,
			Left:   marginHz,
		}
		border := widget.Border{
			Color:        color.NRGBA{R: 0, G: 200, B: 125, A: 200},
			CornerRadius: unit.Dp(4),
			Width:        unit.Dp(2),
		}

		ed := material.Editor(e.theme, &widget.Editor{}, "sec")
		if !preDraw {
			ed = material.Editor(e.theme, &e.boilDurationInput, "sec")

			// If boiling out how far along in the boiling process we are.
			if e.boilTimerState.boiling && progress < 1 {
				boilRemain := float64(e.boilTicker.BoilRemain(e.boilTimerState)) / float64(time.Second)
				// Format to 1 decimal.
				precisionStr := fmt.Sprintf("%%.%df", boilDurationPrecision)
				boilRemainStr := fmt.Sprintf(precisionStr, boilRemain)
				// Update the text in the inputbox
				e.boilDurationInput.SetText(boilRemainStr)
			}
		}

		layout := margins.Layout(gtx,
			func(gtx layout.Context) D {
				return border.Layout(gtx, ed.Layout)
			},
		)

		return layout
	}

	CreateEggWidget := func(constraints C) layout.Widget {
		return func(gtx layout.Context) D {
			gtx.Constraints = constraints

			center := gtx.Constraints.Max.Div(2)
			centerF32 := f32.Pt(float32(center.X), float32(center.Y))
			op.Offset(centerF32).Add(gtx.Ops)

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

			// Draw egg path.
			var eggPath clip.Path
			// eggPath.Move(centerF32)
			func() {
				// Begin the path and close it when function exits.
				eggPath.Begin(gtx.Ops)
				defer eggPath.Close()

				// Egg paramters.
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

			eggArea := clip.Outline{Path: eggPath.End()}.Op()
			// Fill the shape
			color := color.NRGBA{
				R: 255,
				G: uint8(239 * (1 - progress)),
				B: uint8(174 * (1 - progress)),
				A: 255,
			}

			paint.FillShape(gtx.Ops, color, eggArea)
			return D{Size: gtx.Constraints.Max}
		}
	}

	// Reverse rendering order to figure out the size of the egg widget.
	var eggWidget layout.Widget
	{
		preDraw = true
		gtxRev := gtx
		gtxRev.Ops = &op.Ops{}

		flex.Layout(gtxRev,
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
		preDraw = false
	}

	return flex.Layout(gtx,
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

type BoilTimerState struct {
	boiling  bool
	duration time.Duration
	progress float64
}

func NewBoilTimer(freq, duration time.Duration) *BoilTimer {
	bt := BoilTimer{
		freq:   freq,
		ticker: time.NewTicker(freq),
		state: BoilTimerState{
			boiling:  false,
			duration: duration,
			progress: 0.0,
		},
		c:      make(chan BoilTimerState),
		action: make(chan BoilTimerStateSignal),
		closer: make(chan struct{}),
	}
	bt.ticker.Stop()

	go func() {
		var done bool
		for !done {
			select {
			case <-bt.ticker.C:
				// Increment progress by the total boil time divided by the tick duration.
				if bt.state.progress < 1 {
					bt.state.progress += (float64(bt.freq) / float64(bt.state.duration))
					if bt.state.duration == 0 {
						bt.state.progress = 0
					}
					bt.c <- bt.state
				}
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

			case <-bt.closer:
				done = true
			}
		}
		close(bt.c)
		close(bt.action)
	}()

	return &bt
}

type BoilTimer struct {
	freq   time.Duration
	ticker *time.Ticker
	state  BoilTimerState
	action chan BoilTimerStateSignal
	c      chan BoilTimerState
	closer chan struct{}
}

func (bt *BoilTimer) BoilRemain(state BoilTimerState) time.Duration {
	return time.Duration((1 - state.progress) * float64(state.duration))
}

func (bt *BoilTimer) Do(signal BoilTimerSignal, state BoilTimerState) BoilTimerState {
	go func() { bt.action <- BoilTimerStateSignal{signal, &state} }()
	return <-bt.c
}

func (bt *BoilTimer) C() <-chan BoilTimerState { return bt.c }
func (bt *BoilTimer) Close()                   { bt.closer <- struct{}{} }

type BoilTimerStateSignal struct {
	Signal BoilTimerSignal
	State  *BoilTimerState
}

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

const (
	BoilTimerSignalGet BoilTimerSignal = iota
	BoilTimerSignalStop
	BoilTimerSignalStart
	BoilTimerSignalReset
	BoilTimerSignalRestart
)
