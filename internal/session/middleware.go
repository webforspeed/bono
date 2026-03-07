package session

import (
	"context"
	"sync"
)

// Middleware decorates a frontend with cross-cutting behavior.
type Middleware func(SessionFrontend) SessionFrontend

func Chain(frontend SessionFrontend, middleware ...Middleware) SessionFrontend {
	wrapped := frontend
	for i := len(middleware) - 1; i >= 0; i-- {
		wrapped = middleware[i](wrapped)
	}
	return wrapped
}

// SynchronizedMiddleware serializes event delivery and approval prompts.
func SynchronizedMiddleware() Middleware {
	return func(next SessionFrontend) SessionFrontend {
		return &synchronizedFrontend{next: next}
	}
}

type synchronizedFrontend struct {
	next SessionFrontend
	mu   sync.Mutex
}

func (f *synchronizedFrontend) HandleEvent(ctx context.Context, event Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next.HandleEvent(ctx, event)
}

func (f *synchronizedFrontend) RequestApproval(ctx context.Context, req ApprovalRequest) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.next.RequestApproval(ctx, req)
}
