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
	eggWidget := NewEggWidget(boilTimer)
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

		case <-eggWidget.Tick():
			w.Invalidate()
		}
	}
}

func NewEggWidget(boilTimer *BoilTimer) *EggWidget {
	e := EggWidget{
		boilTicker:  boilTimer,
		startButton: &widget.Clickable{},
		boilDurationInput: widget.Editor{
			Alignment:  text.Middle,
			SingleLine: true,
		},
		theme:   material.NewTheme(gofont.Collection()),
		updates: make(chan struct{}),
	}

	go func() {
		for progressNew := range e.boilTicker.ProgressCh() {
			e.progress = progressNew
			e.updates <- struct{}{}
		}
	}()

	return &e
}

type EggWidget struct {
	boilTicker        *BoilTimer
	startButton       *widget.Clickable
	boilDuration      time.Duration
	boilDurationInput widget.Editor
	theme             *material.Theme
	updates           chan struct{}
	progress          float64
}

func (e *EggWidget) Tick() <-chan struct{} { return e.updates }
func (e *EggWidget) Close() {
	close(e.updates)
	e.boilTicker.Close()
}

func (e *EggWidget) Layout(gtx layout.Context) D {
	// Only used for when predrawing the ui.
	var preDraw bool = false

	// Widget for inputing the boil duration.
	const boilDurationPrecision int = 1

	// Define flex layout with the following options.
	flex := layout.Flex{
		Axis:    layout.Vertical,
		Spacing: layout.SpaceStart,
	}

	progressBar := func(gtx layout.Context) D {
		// Get a progress bar from the theme.
		bar := material.ProgressBar(e.theme, float32(e.progress))
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
				if e.boilTicker.Boiling() {
					btnState = "Stop"
				}
				if e.progress >= 1 {
					btnState = "Finished"
					e.boilTicker.Stop(nil)
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

		ed := material.Editor(e.theme, &widget.Editor{}, "sec")
		if !preDraw {
			ed = material.Editor(e.theme, &e.boilDurationInput, "sec")

			// If boiling out how far along in the boiling process we are.
			if e.boilTicker.Boiling() && e.progress < 1 {
				boilRemain := float64(e.boilTicker.BoilRemain(e.progress)) / float64(time.Second)
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
				G: uint8(239 * (1 - e.progress)),
				B: uint8(174 * (1 - e.progress)),
				A: 255,
			}

			paint.FillShape(gtx.Ops, color, eggArea)
			return D{Size: gtx.Constraints.Max}
		}
	}

	// Handle start button clicks here.
	if e.startButton.Clicked() {
		// Read from the input box
		inputString := e.boilDurationInput.Text()
		inputString = strings.TrimSpace(inputString)
		inputFloat, _ := strconv.ParseFloat(inputString, 32)

		// Check if the output of the ticker has changed significantly
		boilRemain := float64(e.boilTicker.BoilRemain(e.progress))
		if math.Abs(boilRemain-inputFloat) > math.Pow10(-boilDurationPrecision) {
			e.boilDuration = time.Duration(inputFloat * float64(time.Second))
			e.boilTicker.Reset(&e.boilDuration)
		}

		// Handle ticker.
		if !e.boilTicker.Boiling() {
			e.boilTicker.Start(nil)
		} else {
			e.boilTicker.Stop(nil)
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

func NewBoilTimer(freq, duration time.Duration) *BoilTimer {
	bt := BoilTimer{
		freq:       freq,
		duration:   duration,
		ticker:     time.NewTicker(freq),
		boiling:    false,
		progress:   0.0,
		progressCh: make(chan float64),
		closer:     make(chan struct{}),
	}

	go func() {
		var done bool
		for !done {
			select {
			case <-bt.ticker.C:
				// Increment progress by the total boil time divided by the tick duration.
				if bt.progress < 1 {
					bt.progress += (float64(bt.freq) / float64(bt.duration))
					if bt.duration == 0 {
						bt.progressCh <- 0
						continue
					}
					bt.progressCh <- bt.progress
				}
			case <-bt.closer:
				done = true
			}
		}
		close(bt.progressCh)
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
	closer     chan struct{}
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
func (bt *BoilTimer) Close()                     { bt.closer <- struct{}{} }
func (bt *BoilTimer) BoilRemain(progress float64) time.Duration {
	return time.Duration((1 - progress) * float64(bt.duration))
}
