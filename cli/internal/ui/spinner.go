package ui

import (
	"fmt"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner is a single-line animated progress indicator. When styling is
// disabled it degrades to plain status lines with no animation.
type Spinner struct {
	mu       sync.Mutex
	msg      string
	frame    int
	active   bool
	stop     chan struct{}
	done     chan struct{}
	lastLine string
}

// NewSpinner creates a spinner with an initial message (not yet started).
func NewSpinner(msg string) *Spinner {
	return &Spinner{msg: msg}
}

// Start begins the animation. On non-TTY it waits silently for Success/Fail.
func (s *Spinner) Start() *Spinner {
	if !Enabled {
		return s
	}
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return s
	}
	s.active = true
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.loop()
	return s
}

func (s *Spinner) loop() {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			close(s.done)
			return
		case <-ticker.C:
			s.render()
		}
	}
}

func (s *Spinner) render() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return
	}
	frame := spinnerFrames[s.frame%len(spinnerFrames)]
	s.frame++
	fmt.Fprintf(Out, "\r\033[K  %s %s", Brand(frame), s.msg)
}

// Update changes the spinner message.
func (s *Spinner) Update(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
	if !Enabled {
		fmt.Fprintf(Out, "  %s %s\n", arrow(), msg)
	}
}

// Println prints a persistent line above the spinner without disrupting it.
func (s *Spinner) Println(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if Enabled && s.active {
		fmt.Fprint(Out, "\r\033[K")
	}
	fmt.Fprintln(Out, line)
}

func (s *Spinner) halt() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	stop := s.stop
	done := s.done
	s.mu.Unlock()

	close(stop)
	<-done
	fmt.Fprint(Out, "\r\033[K")
}

// Success stops the spinner and prints a green check line.
func (s *Spinner) Success(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	s.halt()
	fmt.Fprintf(Out, "  %s %s\n", Green(tick()), msg)
}

// Fail stops the spinner and prints a red cross line.
func (s *Spinner) Fail(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	s.halt()
	fmt.Fprintf(Out, "  %s %s\n", Red(cross()), msg)
}

// Stop clears the spinner with no trailing line.
func (s *Spinner) Stop() {
	s.halt()
}
