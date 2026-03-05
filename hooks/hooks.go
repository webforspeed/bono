package hooks

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// Handler processes a hook event. Implementations should be fast and non-blocking.
type Handler interface {
	Handle(ctx context.Context, event Event, payload any)
}

// HandlerFunc adapts a plain function to the Handler interface.
type HandlerFunc func(ctx context.Context, event Event, payload any)

func (f HandlerFunc) Handle(ctx context.Context, event Event, payload any) {
	f(ctx, event, payload)
}

// Dispatcher manages handler registration and event dispatch.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[Event][]Handler
}

// NewDispatcher creates an empty dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{handlers: make(map[Event][]Handler)}
}

// On registers one or more handlers for an event.
func (d *Dispatcher) On(event Event, h ...Handler) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[event] = append(d.handlers[event], h...)
}

// Fire dispatches an event to all registered handlers.
// Panics in handlers are recovered and printed to stderr.
func (d *Dispatcher) Fire(ctx context.Context, event Event, payload any) {
	d.mu.RLock()
	hs := d.handlers[event]
	d.mu.RUnlock()

	for _, h := range hs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Fprintf(os.Stderr, "hook panic [%s]: %v\n", event, r)
				}
			}()
			h.Handle(ctx, event, payload)
		}()
	}
}
