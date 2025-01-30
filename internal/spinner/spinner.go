package spinner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Spinner struct {
	message  string
	frames   []string
	delay    time.Duration
	active   bool
	mu       sync.Mutex
	stopChan chan struct{}
}

func New(message string) *Spinner {
	return &Spinner{
		message:  message,
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		delay:    100 * time.Millisecond,
		stopChan: make(chan struct{}),
	}
}

func (s *Spinner) Start(ctx context.Context) {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		frameIndex := 0
		for {
			select {
			case <-ctx.Done():
				s.Stop()
				return
			case <-s.stopChan:
				return
			default:
				frame := s.frames[frameIndex%len(s.frames)]
				fmt.Printf("\r%s %s", frame, s.message)
				frameIndex++
				time.Sleep(s.delay)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	s.stopChan <- struct{}{}
	fmt.Print("\r") // limpiar la línea
}

func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}
