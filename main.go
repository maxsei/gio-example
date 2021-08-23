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

var progress float64

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
	//////////////////////////////////////////////////////////////////////////////
	//                              Draw Variables                              //
	//////////////////////////////////////////////////////////////////////////////

	// Variable that stores UI operations
	var ops op.Ops
	// If we click the start button we toggle boiling the egg.
	var startButton widget.Clickable

	// Only used for when predrawing the ui.
	var preDraw bool = false

	// Use the builting material UI theme.
	theme := material.NewTheme(gofont.Collection())

	// Create ticker for boiler that is created stopped.
	var boilDuration time.Duration
	const boilTickerFreq time.Duration = time.Second / 25
	boilTicker := NewBoilTimer(boilTickerFreq, boilDuration)
	boilTicker.Stop(nil)

	// Widget for inputing the boil duration.
	const boilDurationPrecision int = 1
	var boilDurationInput widget.Editor
	boilDurationInput.SingleLine = true
	boilDurationInput.Alignment = text.Middle

	//////////////////////////////////////////////////////////////////////////////
	//                              Define Widgets                              //
	//////////////////////////////////////////////////////////////////////////////

	// Define flex layout with the following options.
	flex := layout.Flex{
		Axis:    layout.Vertical,
		Spacing: layout.SpaceStart,
	}

	progressBar := func(gtx layout.Context) D {
		// Get a progress bar from the theme.
		bar := material.ProgressBar(theme, float32(progress))
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
				if boilTicker.Boiling() {
					btnState = "Stop"
				}
				if progress >= 1 {
					btnState = "Finished"
					boilTicker.Stop(nil)
				}

				// Style the button according to the theme to get a styled button back.
				btn := material.Button(theme, &startButton, btnState)

				// Add operations to the graphical context to draw the button.
				return btn.Layout(gtx)
			},
		)
	}

	// Boil duration input widget.
	boilDurationInputWidget := func(gtx layout.Context) D {
		hzMarginPct := float32(0.95)
		hzMargin := float32(gtx.Constraints.Max.X) * hzMarginPct / 2

		margins := layout.Inset{
			Top:    unit.Dp(20),
			Bottom: unit.Dp(20),
			Right:  unit.Dp(hzMargin),
			Left:   unit.Dp(hzMargin),
		}
		border := widget.Border{
			Color:        color.NRGBA{R: 0, G: 200, B: 125, A: 200},
			CornerRadius: unit.Dp(4),
			Width:        unit.Dp(2),
		}

		ed := material.Editor(theme, &widget.Editor{}, "sec")

		if !preDraw {
			ed = material.Editor(theme, &boilDurationInput, "sec")

			// If boiling out how far along in the boiling process we are.
			if boilTicker.Boiling() && progress < 1 {
				boilRemain := float64(boilTicker.BoilRemain(progress)) / float64(time.Second)
				// Format to 1 decimal.
				precisionStr := fmt.Sprintf("%%.%df", boilDurationPrecision)
				boilRemainStr := fmt.Sprintf(precisionStr, boilRemain)
				// Update the text in the inputbox
				boilDurationInput.SetText(boilRemainStr)
			}
		}

		layout := margins.Layout(gtx,
			func(gtx layout.Context) D {
				return border.Layout(gtx, ed.Layout)
			},
		)

		return layout
	}

	//////////////////////////////////////////////////////////////////////////////
	//                               Program Loop                               //
	//////////////////////////////////////////////////////////////////////////////

	for {
		select {
		case e := <-w.Events():
			// Frame event.
			if fe, ok := e.(system.FrameEvent); ok {
				// Handle start button clicks here.
				if startButton.Clicked() {
					// Read from the input box
					inputString := boilDurationInput.Text()
					inputString = strings.TrimSpace(inputString)
					inputFloat, _ := strconv.ParseFloat(inputString, 32)

					// Check if the output of the ticker has changed significantly
					boilRemain := float64(boilTicker.BoilRemain(progress))
					if math.Abs(boilRemain-inputFloat) > math.Pow10(-boilDurationPrecision) {
						boilDuration = time.Duration(inputFloat * float64(time.Second))
						boilTicker.Reset(&boilDuration)
					}

					// Handle ticker.
					if !boilTicker.Boiling() {
						boilTicker.Start(nil)
					} else {
						boilTicker.Stop(nil)
					}

				}

				// Reverse rendering order to figure out the size of the egg widget.
				var eggWidget layout.Widget
				{
					preDraw = true
					var ops op.Ops
					gtx := layout.NewContext(&ops, fe)
					flex.Layout(gtx,
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

				// Create graphical context that contains all the UI operations and the
				// frame event that triggered them.
				gtx := layout.NewContext(&ops, fe)

				flex.Layout(gtx,
					// Add an egg.
					layout.Rigid(eggWidget),
					// Add a boil duration input widget.
					layout.Rigid(boilDurationInputWidget),
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

		case progress = <-boilTicker.ProgressCh():
			// fmt.Printf("progress = %+v\n", progress)
			w.Invalidate()
		}
	}
}

func CreateEggWidget(constraints C) layout.Widget {
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

func NewBoilTimer(freq, duration time.Duration) *BoilTimer {
	bt := BoilTimer{
		freq:       freq,
		duration:   duration,
		ticker:     time.NewTicker(freq),
		boiling:    false,
		progress:   0.0,
		progressCh: make(chan float64),
	}

	go func() {
		fmt.Println("init")

		for _ = range bt.ticker.C {
			// Increment progress by the total boil time divided by the tick duration.
			fmt.Printf("progress = %+v\n", progress)
			if bt.progress < 1 {
				bt.progress += (float64(bt.freq) / float64(bt.duration))
				bt.progressCh <- bt.progress
			}
		}
	}()

	return &bt
}

type BoilTimer struct {
	freq       time.Duration
	duration   time.Duration
	boiling    bool
	ticker     *time.Ticker
	progress   float64
	progressCh chan float64
}

func (bt *BoilTimer) setDuration(duration *time.Duration) {
	if duration != nil {
		bt.duration = *duration
	}
}

func (bt *BoilTimer) Start(duration *time.Duration) {
	bt.ticker.Stop()
	bt.setDuration(duration)
	bt.boiling = true
	bt.ticker.Reset(bt.freq)
}

func (bt *BoilTimer) Stop(duration *time.Duration) {
	bt.ticker.Stop()
	bt.setDuration(duration)
	bt.boiling = false
}

func (bt *BoilTimer) Reset(duration *time.Duration) {
	bt.progress = 0
	go func() { bt.progressCh <- 0 }()
	bt.Stop(duration)
}

func (bt *BoilTimer) Restart(duration *time.Duration) {
	bt.progress = 0
	go func() { bt.progressCh <- 0 }()
	bt.Start(duration)
}

func (bt *BoilTimer) Boiling() bool              { return bt.boiling }
func (bt *BoilTimer) ProgressCh() <-chan float64 { return bt.progressCh }
func (bt *BoilTimer) BoilRemain(progress float64) time.Duration {
	return time.Duration((1 - progress) * float64(bt.duration))
}
