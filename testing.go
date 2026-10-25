// Copyright 2021 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocui

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var simulationScreen tcell.SimulationScreen

// TestingScreen is used to create tests using a simulated screen
type TestingScreen struct {
	screen  tcell.SimulationScreen
	gui     *Gui
	started bool
}

func isSimScreen() {
	if simulationScreen == nil {
		panic("Cannot use testing methods with a real screen use ")
	}
}

// Creates an instance of TestingScreen for the current Gui
func (g *Gui) GetTestingScreen() TestingScreen {
	isSimScreen()

	return TestingScreen{
		screen: simulationScreen,
		gui:    g,
	}
}

// StartGui starts the Gui using the test screen
// it returns a func which stops the Gui to cleanup
// after your test finishes:
//
// cleanup := testingScreen.StartGui()
// defer cleanup()
func (t *TestingScreen) StartGui() func() {
	t.gui.testNotify = make(chan struct{})
	go func() {
		if err := t.gui.MainLoop(); err != nil && !errors.Is(err, ErrQuit) {
			log.Panic(err)
		}
	}()
	t.WaitSync()

	t.started = true

	// Return a func that will stop the main loop
	return func() {
		t.gui.stop <- struct{}{}
	}
}

// SendStringAsKeys sends a string of text to gocui
func (t *TestingScreen) SendStringAsKeys(input string) {
	if !t.started {
		panic("TestingScreen must be started using 'StartGui' before injecting keys")
	}
	t.injectString(input)
}

// SendsKey sends a key to gocui
func (t *TestingScreen) SendKey(key Key) {
	if !t.started {
		panic("TestingScreen must be started using 'StartGui' before injecting keys")
	}
	t.screen.InjectKey(tcell.Key(key), rune(key), tcell.ModNone)
}

// SendsKeySync sends a key to gocui and wait until MainLoop process it.
func (t *TestingScreen) SendKeySync(key Key) {
	if !t.started {
		panic("TestingScreen must be started using 'StartGui' before injecting keys")
	}
	t.screen.InjectKey(tcell.Key(key), rune(key), tcell.ModNone)
	t.WaitSync()
}

// WaitSync sends time event to gocui and awaits notification that it was received.
//
// Notification is sent from gocui at the end of MainLoop, so after this function returns,
// user has confirmation that all the keys sent to gocui before time event were processed.
func (t *TestingScreen) WaitSync() {
	ev := &tcell.EventTime{}
	t.screen.PostEventWait(ev)
	<-t.gui.testNotify
}

// GetViewContent gets the current conent of a view from the simulated screen
func (t *TestingScreen) GetViewContent(viewName string) (string, error) {
	if !t.started {
		panic("TestingScreen must be started using 'StartGui' before contents can be retrieve")
	}
	view, err := t.gui.View(viewName)
	if err != nil {
		return "", fmt.Errorf("failed to retreive view: %w", err)
	}

	x0, y0, x1, y1 := view.Dimensions()

	// Account for the border
	x0++
	y0++
	x1--
	y1--

	// Walk each line in the view
	var result strings.Builder
	Xcurrent := x0
	Ycurrent := y0
	for y0 < y1 || y0 == y1 {
		// Did we reach the end of the line?
		if Xcurrent > x1 {
			Xcurrent = x0
			Ycurrent++
			result.WriteString("\n")
		}

		// Did we reach the bottom of the view?
		if Ycurrent > y1 {
			break
		}

		// Get the content (without formatting) at that position
		content, err := t.gui.Rune(Xcurrent, Ycurrent)
		if err != nil {
			return "", fmt.Errorf("failed reading rune from simulation screen: %w", err)
		}
		result.WriteRune(content)
		Xcurrent++
	}

	return result.String(), nil
}

// injectString preserves Unicode scalar values and ASCII control-key mappings.
// Waiting for queue space avoids losing events when the simulation queue is full.
func (t *TestingScreen) injectString(str string) {
	for _, ch := range str {
		key, mod := tcell.KeyRune, tcell.ModNone
		if ch < 0x20 {
			key = tcell.Key(ch)
			ch = 0
			if key >= tcell.KeyCtrlA && key <= tcell.KeyCtrlZ {
				mod = tcell.ModCtrl
			}
		}
		simulationScreen.PostEventWait(tcell.NewEventKey(key, ch, mod))
	}
}
