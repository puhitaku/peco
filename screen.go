package peco

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

// termbox always gives us some sort of warning when we run
// go run -race cmd/peco/peco.go
var termboxMutex = newMutex()

// These functions are here so that we can test
var (
	termboxClose     = termbox.Close
	termboxFlush     = termbox.Flush
	termboxInit      = termbox.Init
	termboxPollEvent = termbox.PollEvent
	termboxSetCell   = termbox.SetCell
	termboxSize      = termbox.Size
)

func (t Termbox) Init() error {
	trace("initializing termbox")
	if err := termboxInit(); err != nil {
		return err
	}

	return t.PostInit()
}

func (t Termbox) Close() error {
	termboxClose()
	return nil
}

// SendEvent is used to allow programmers generate random
// events, but it's only useful for testing purposes.
// When interactiving with termbox-go, this method is a noop
func (t Termbox) SendEvent(_ termbox.Event) {
	// no op
}

// Flush calls termbox.Flush
func (t Termbox) Flush() error {
	termboxMutex.Lock()
	defer termboxMutex.Unlock()
	return termboxFlush()
}

// PollEvent returns a channel that you can listen to for
// termbox's events. The actual polling is done in a
// separate gouroutine
func (t Termbox) PollEvent() chan termbox.Event {
	// XXX termbox.PollEvent() can get stuck on unexpected signal
	// handling cases. We still would like to wait until the user
	// (termbox) has some event for us to process, but we don't
	// want to allow termbox to control/block our input loop.
	//
	// Solution: put termbox polling in a separate goroutine,
	// and we just watch for a channel. The loop can now
	// safely be implemented in terms of select {} which is
	// safe from being stuck.
	evCh := make(chan termbox.Event)
	go func() {
		defer func() { recover() }()
		defer func() { close(evCh) }()
		for {
			evCh <- termboxPollEvent()
		}
	}()
	return evCh

}

// SetCell writes to the terminal
func (t Termbox) SetCell(x, y int, ch rune, fg, bg termbox.Attribute) {
	termboxMutex.Lock()
	defer termboxMutex.Unlock()
	termboxSetCell(x, y, ch, fg, bg)
}

// Size returns the dimensions of the current terminal
func (t Termbox) Size() (int, int) {
	termboxMutex.Lock()
	defer termboxMutex.Unlock()
	return termboxSize()
}

type PrintArgs struct {
	X       int
	XOffset int
	Y       int
	Fg      termbox.Attribute
	Bg      termbox.Attribute
	Msg     string
	Fill    bool
}

func (t Termbox) Print(args PrintArgs) int {
	var written int

	bg := args.Bg
	fg := args.Fg
	msg := args.Msg
	x := args.X
	y := args.Y
	xOffset := args.XOffset
	for len(msg) > 0 {
		c, w := utf8.DecodeRuneInString(msg)
		if c == utf8.RuneError {
			c = '?'
			w = 1
		}
		msg = msg[w:]
		if c == '\t' {
			// In case we found a tab, we draw it as 4 spaces
			n := 4 - (x+xOffset)%4
			for i := int(0); i <= n; i++ {
				t.SetCell(int(x+i), int(y), ' ', fg, bg)
			}
			written += n
			x += n
		} else {
			t.SetCell(int(x), int(y), c, fg, bg)
			n := int(runewidth.RuneWidth(c))
			x += n
			written += n
		}
	}

	if !args.Fill {
		return written
	}

	width, _ := t.Size()
	for ; x < int(width); x++ {
		t.SetCell(int(x), int(y), ' ', fg, bg)
	}
	written += int(width) - x
	return written
}
