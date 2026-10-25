package gocui

import (
	"errors"
	"strings"
	"testing"
)

// TestTestingScreenUnicode covers issue #141 with one native simulator.
func TestTestingScreenUnicode(t *testing.T) {
	g, err := NewGui(OutputSimulator, true)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	g.SetManagerFunc(func(g *Gui) error {
		v, err := g.SetView("input", 0, 0, 70, 5, 0)
		if errors.Is(err, ErrUnknownView) {
			v.Editable = true
			v.Wrap = true
			_, err = g.SetCurrentView("input")
		}
		return err
	})
	screen := g.GetTestingScreen()
	stop := screen.StartGui()
	defer stop()
	cases := []struct{ input, want string }{
		{"界", "界"}, {"A界", "A界"}, {"123456789界", "123456789界"}, {"🙂", "🙂"},
		{strings.Repeat("ab界", 30), strings.Repeat("ab界", 30)},
		{"abc\bX", "abX"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			reset := make(chan error, 1)
			g.Update(func(g *Gui) error {
				v, err := g.View("input")
				if err == nil {
					v.Clear()
				}
				reset <- err
				return err
			})
			if err := <-reset; err != nil {
				t.Fatal(err)
			}
			screen.WaitSync()
			screen.SendStringAsKeys(tc.input)
			screen.WaitSync()
			v, err := g.View("input")
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSuffix(v.Buffer(), "\n"); got != tc.want {
				t.Fatalf("input lost: got %q, want %q", got, tc.want)
			}
		})
	}
}
