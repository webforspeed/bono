package session

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// HeadlessFrontend renders Bono session events as an append-only terminal transcript.
type HeadlessFrontend struct {
	out io.Writer
	in  *bufio.Reader

	streamingContent   bool
	streamingReasoning bool
	sawContentDelta    bool
}

func NewHeadlessFrontend(out io.Writer, in io.Reader) *HeadlessFrontend {
	return &HeadlessFrontend{
		out: out,
		in:  bufio.NewReader(in),
	}
}

func (f *HeadlessFrontend) HandleEvent(_ context.Context, event Event) {
	switch event := event.(type) {
	case UserPromptEvent:
		f.finishStreaming()
		fmt.Fprintf(f.out, "> %s\n\n", event.Prompt)
	case ContentDeltaEvent:
		f.startContent()
		_, _ = io.WriteString(f.out, event.Delta)
		f.sawContentDelta = true
	case ReasoningDeltaEvent:
		f.startReasoning()
		_, _ = io.WriteString(f.out, event.Delta)
	case MessageEvent:
		f.finishStreaming()
		if f.sawContentDelta {
			f.sawContentDelta = false
			return
		}
		if strings.TrimSpace(event.Content) != "" {
			fmt.Fprintln(f.out, event.Content)
			fmt.Fprintln(f.out)
		}
	case ToolCallEvent:
		f.finishStreaming()
		line := "● " + FormatTool(event.Name, event.Args)
		if event.Sandboxed {
			line += " [Running in sandbox]"
		}
		fmt.Fprintln(f.out, line)
	case ToolDoneEvent:
		f.finishStreaming()
		line := "● " + FormatTool(event.Name, event.Args)
		if event.Sandboxed {
			line += " [Ran in sandbox]"
		}
		line += " => " + event.Status
		fmt.Fprintln(f.out, line)
		fmt.Fprintln(f.out)
	case DiffPreviewEvent:
		f.finishStreaming()
		fmt.Fprintln(f.out, RenderDiffPreview(event))
	case PreTaskStartEvent:
		f.finishStreaming()
		fmt.Fprintf(f.out, "● Running %s agent...\n", event.Name)
	case PreTaskEndEvent:
		f.finishStreaming()
		fmt.Fprintf(f.out, "● Completed %s agent\n", event.Name)
	case SubAgentStartEvent:
		f.finishStreaming()
		fmt.Fprintf(f.out, "● Running %s agent...\n", event.Name)
	case SubAgentEndEvent:
		f.finishStreaming()
		fmt.Fprintf(f.out, "● Completed %s agent\n", event.Name)
	case ErrorEvent:
		f.finishStreaming()
		if event.Err != nil {
			fmt.Fprintf(f.out, "Error: %v\n", event.Err)
		}
	case ContextUsageEvent:
	case ResponseModelEvent:
	case RefreshGitStatusEvent:
	default:
		f.finishStreaming()
	}
}

func (f *HeadlessFrontend) RequestApproval(ctx context.Context, req ApprovalRequest) bool {
	f.finishStreaming()

	switch req.Kind {
	case ApprovalTool:
		line := "● " + FormatTool(req.ToolName, req.ToolArgs)
		if req.ExecutionReason != "" {
			line += " [Outside sandbox: " + req.ExecutionReason + "]"
		}
		fmt.Fprintln(f.out, line)
		ok := f.readApproval(ctx, "  ↳ Approve? [y/N]: ")
		if !ok {
			fmt.Fprintf(f.out, "● %s => cancelled\n\n", FormatTool(req.ToolName, req.ToolArgs))
		}
		return ok
	case ApprovalSandboxFallback:
		fmt.Fprintf(f.out, "  ↳ %s [Sandbox blocked: %s]\n", DisplaySandboxCommand(req.Command), fallbackReason(req.Reason))
		ok := f.readApproval(ctx, "  ↳ Run outside sandbox? [y/N]: ")
		if !ok {
			fmt.Fprintf(f.out, "  ↳ %s => cancelled\n\n", DisplaySandboxCommand(req.Command))
		}
		return ok
	case ApprovalChangeBatch:
		prompt := BatchReviewPrompt(req.ChangeCount)
		fmt.Fprintln(f.out, "● "+prompt)
		ok := f.readApproval(ctx, "  ↳ Approve? [y/N]: ")
		status := "approved"
		if !ok {
			status = "undone"
		}
		fmt.Fprintf(f.out, "● %s => %s\n\n", prompt, status)
		return ok
	default:
		return false
	}
}

func (f *HeadlessFrontend) startReasoning() {
	if f.streamingContent {
		f.finishStreaming()
	}
	if !f.streamingReasoning {
		fmt.Fprint(f.out, "Thinking: ")
		f.streamingReasoning = true
	}
}

func (f *HeadlessFrontend) startContent() {
	if f.streamingReasoning {
		fmt.Fprintln(f.out)
		fmt.Fprintln(f.out)
		f.streamingReasoning = false
	}
	if !f.streamingContent {
		f.streamingContent = true
	}
}

func (f *HeadlessFrontend) finishStreaming() {
	if f.streamingReasoning || f.streamingContent {
		fmt.Fprintln(f.out)
		fmt.Fprintln(f.out)
	}
	f.streamingReasoning = false
	f.streamingContent = false
}

func (f *HeadlessFrontend) readApproval(ctx context.Context, prompt string) bool {
	fmt.Fprint(f.out, prompt)
	answerCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		line, err := f.in.ReadString('\n')
		if err != nil {
			errCh <- err
			return
		}
		answerCh <- line
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(f.out)
		return false
	case err := <-errCh:
		if err == io.EOF {
			fmt.Fprintln(f.out)
			return false
		}
		fmt.Fprintln(f.out)
		return false
	case line := <-answerCh:
		answer := strings.TrimSpace(strings.ToLower(line))
		return answer == "y" || answer == "yes"
	}
}

func fallbackReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "policy violation"
	}
	return reason
}
