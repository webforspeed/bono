package session

import (
	"context"
	"testing"

	core "github.com/webforspeed/bono-core"
)

type recordingFrontend struct {
	order []string
}

func (f *recordingFrontend) HandleEvent(_ context.Context, _ Event) {
	f.order = append(f.order, "base-event")
}

func (f *recordingFrontend) RequestApproval(_ context.Context, _ ApprovalRequest) bool {
	f.order = append(f.order, "base-approval")
	return true
}

func (f *recordingFrontend) RequestSubAgentApproval(_ context.Context, _ core.SubAgentResult) core.SubAgentApprovalResponse {
	return core.SubAgentApprovalResponse{Action: core.SubAgentApprove}
}

func TestChainMiddlewareOrder(t *testing.T) {
	base := &recordingFrontend{}
	wrapped := Chain(base,
		func(next SessionFrontend) SessionFrontend {
			return middlewareFunc{
				handleEvent: func(ctx context.Context, event Event) {
					base.order = append(base.order, "mw1-before-event")
					next.HandleEvent(ctx, event)
					base.order = append(base.order, "mw1-after-event")
				},
				requestApproval: func(ctx context.Context, req ApprovalRequest) bool {
					base.order = append(base.order, "mw1-before-approval")
					ok := next.RequestApproval(ctx, req)
					base.order = append(base.order, "mw1-after-approval")
					return ok
				},
			}
		},
		func(next SessionFrontend) SessionFrontend {
			return middlewareFunc{
				handleEvent: func(ctx context.Context, event Event) {
					base.order = append(base.order, "mw2-before-event")
					next.HandleEvent(ctx, event)
					base.order = append(base.order, "mw2-after-event")
				},
				requestApproval: func(ctx context.Context, req ApprovalRequest) bool {
					base.order = append(base.order, "mw2-before-approval")
					ok := next.RequestApproval(ctx, req)
					base.order = append(base.order, "mw2-after-approval")
					return ok
				},
			}
		},
	)

	wrapped.HandleEvent(context.Background(), MessageEvent{Content: "hi"})
	wrapped.RequestApproval(context.Background(), ApprovalRequest{Kind: ApprovalChangeBatch, ChangeCount: 1})

	want := []string{
		"mw1-before-event",
		"mw2-before-event",
		"base-event",
		"mw2-after-event",
		"mw1-after-event",
		"mw1-before-approval",
		"mw2-before-approval",
		"base-approval",
		"mw2-after-approval",
		"mw1-after-approval",
	}
	if len(base.order) != len(want) {
		t.Fatalf("len(order) = %d, want %d (%v)", len(base.order), len(want), base.order)
	}
	for i := range want {
		if base.order[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q (full=%v)", i, base.order[i], want[i], base.order)
		}
	}
}

type middlewareFunc struct {
	handleEvent     func(ctx context.Context, event Event)
	requestApproval func(ctx context.Context, req ApprovalRequest) bool
}

func (m middlewareFunc) HandleEvent(ctx context.Context, event Event) {
	m.handleEvent(ctx, event)
}

func (m middlewareFunc) RequestApproval(ctx context.Context, req ApprovalRequest) bool {
	return m.requestApproval(ctx, req)
}

func (m middlewareFunc) RequestSubAgentApproval(_ context.Context, _ core.SubAgentResult) core.SubAgentApprovalResponse {
	return core.SubAgentApprovalResponse{Action: core.SubAgentApprove}
}
