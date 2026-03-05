package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	core "github.com/webforspeed/bono-core"
	"github.com/webforspeed/bono/internal/worktree"
	"github.com/webforspeed/bono/prompts"
	"github.com/webforspeed/bono/tui"
)

const systemPromptVersion = "v1.0.5"
const releaseRepo = "webforspeed/bono"
const updateCheckTimeout = 3 * time.Second

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	loadEnv()

	cwd, _ := os.Getwd()
	if cwd == "" {
		cwd = "."
	}
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("LOGNAME")
	}

	promptCtx := prompts.PromptContext{
		HostContext: prompts.HostContext{
			CWD:      cwd,
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Username: username,
			DateTime: time.Now().Format("2006-01-02 15:04:05 MST"),
		},
		Identity: prompts.AgentIdentity{
			Role:     "coding",
			Platform: "terminal",
		},
	}

	systemPrompt, err := prompts.LoadSystemPromptVersion(promptCtx, systemPromptVersion)
	if err != nil {
		fmt.Printf("Error loading system prompt version %q: %v\n", systemPromptVersion, err)
		os.Exit(1)
	}

	// Load model catalog from code.
	models := tui.DefaultModelCatalog()

	// Model priority: MODEL env var (deprecated) > openrouter/free > first model in catalog > bono-core default
	model := os.Getenv("MODEL")
	if model == "" {
		model = "openrouter/free"
	}
	if model == "" && len(models) > 0 {
		model = models[0].ID
	}
	embeddingDims := 0
	if n := os.Getenv("EMBEDDING_DIMS"); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			embeddingDims = v
		}
	}

	config := core.Config{
		APIKey:       os.Getenv("OPENROUTER_API_KEY"),
		BaseURL:      os.Getenv("BASE_URL"),
		Model:        model,
		SystemPrompt: systemPrompt,
		HTTPTimeout:  120 * time.Second,
		CodeSearch: &core.CodeSearchConfig{
			DBPath: ".bono/index.db",
			Model:  os.Getenv("EMBEDDING_MODEL"),
			Dims:   embeddingDims,
		},
		Web: &core.WebConfig{
			Model:        envOr("WEB_ANSWER_MODEL", "perplexity/sonar"),
			SearchEngine: envOr("WEB_SEARCH_ENGINE", "exa"),
		},
	}
	if n := os.Getenv("API_TIMEOUT_SEC"); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			config.HTTPTimeout = time.Duration(v) * time.Second
		}
	}
	if n := os.Getenv("MAX_TOOL_CALLS_PER_TURN"); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v >= 0 {
			config.MaxToolCallsPerTurn = v
		}
	} else {
		config.MaxToolCallsPerTurn = 50
	}

	// Validate API key early
	if config.APIKey == "" {
		fmt.Println("Error: OPENROUTER_API_KEY required")
		os.Exit(1)
	}

	// Create agent
	agent, err := core.NewAgent(config)
	if err != nil {
		fmt.Printf("Error creating agent: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := agent.Close(); err != nil {
			fmt.Printf("Warning: failed to close agent resources: %v\n", err)
		}
	}()

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := agent.CodeSearchInitError(); err != nil {
		fmt.Printf("Warning: code search unavailable: %v\n", err)
	} else if svc := agent.CodeSearchService(); svc != nil && !svc.CodeSearchSupportsVector() {
		fmt.Println("Warning: sqlite-vec unavailable; code search is running in text-only mode.")
	}
	if err := agent.WebInitError(); err != nil {
		fmt.Printf("Warning: web tools unavailable: %v\n", err)
	}

	// Create TUI model
	tuiModel := tui.NewWithOptions(agent, ctx, tui.SpinnerDot, models)
	tuiModel.SetStatusBarText(tui.StatusBarText(version))

	var watcher *tui.FileWatcher
	if w, err := tui.NewFileWatcher(cwd); err == nil {
		watcher = w
		tuiModel.SetWatcher(watcher)
	}

	// Set initial index status in sidebar
	if svc := agent.CodeSearchService(); svc != nil {
		stats, err := svc.CodeSearchStats()
		if err == nil && stats.TotalChunks > 0 {
			tuiModel.SetIndexStats(stats.TotalFiles)
		}
	}

	// Create Bubble Tea program (use alt screen for full viewport)
	p := tea.NewProgram(&tuiModel, tea.WithAltScreen())
	tuiModel.SetProgram(p)
	startUpdateCheck(ctx, p, version)

	// Warm model limits in background so context usage shows from the first response.
	go func() {
		warmCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		_ = agent.WarmModelUsageLimits(warmCtx, model)
	}()

	// Start file watcher
	if watcher != nil {
		go watcher.Start(ctx, func(count int) {
			p.Send(tui.WatcherNotifyMsg{ChangedCount: count})
		})
	}

	// Set up agent hooks to send messages to TUI
	worktreeMgr := worktree.NewManager()

	agent.OnToolCall = func(name string, args map[string]any) bool {
		if name == "read_file" || name == "compact_context" || name == "code_search" || name == "WebSearch" || name == "WebFetch" {
			// Auto-approve read-only tools
			p.Send(tui.AgentToolCallMsg{Name: name, Args: args})
			return true
		}

		if name == "write_file" || name == "edit_file" {
			originalPath, _ := args["path"].(string)
			session, err := worktreeMgr.EnsureSession(ctx, cwd)
			if err == nil {
				relPath, rewrittenAbs, err := worktreeMgr.RewritePathForWorktree(session, originalPath)
				if err == nil {
					beforeContent, wasNewFile, readErr := worktree.ReadFileOrEmpty(filepath.Join(session.RepoRoot, filepath.FromSlash(relPath)))
					if readErr == nil {
						args["path"] = rewrittenAbs
						worktreeMgr.RegisterRewrite(worktree.PathRewrite{
							ToolName:      name,
							OriginalPath:  originalPath,
							RelPath:       relPath,
							RewrittenAbs:  rewrittenAbs,
							WasNewFile:    wasNewFile,
							BeforeContent: beforeContent,
						})
						p.Send(tui.AgentToolCallMsg{Name: name, Args: args})
						return true
					}
				}
			}
			// Fallback to existing approval behavior if worktree orchestration fails.
			approved := make(chan bool, 1)
			p.Send(tui.AgentToolCallMsg{Name: name, Args: args, Approved: approved})
			select {
			case result := <-approved:
				return result
			case <-ctx.Done():
				return false
			}
		}

		if name == "run_shell" || name == "python_runtime" {
			// Sandboxed commands auto-approve
			if core.IsSandboxEnabled() {
				p.Send(tui.AgentToolCallMsg{Name: name, Args: args, Sandboxed: true})
				return true
			}
			// Non-sandboxed requires approval
			approved := make(chan bool, 1)
			p.Send(tui.AgentToolCallMsg{Name: name, Args: args, Approved: approved})
			select {
			case result := <-approved:
				return result
			case <-ctx.Done():
				return false
			}
		}

		// Other tools require approval
		approved := make(chan bool, 1)
		p.Send(tui.AgentToolCallMsg{Name: name, Args: args, Approved: approved})

		// Block until user approves (Enter) or rejects (Esc), or context cancelled
		select {
		case result := <-approved:
			return result
		case <-ctx.Done():
			return false
		}
	}

	agent.OnToolDone = func(name string, args map[string]any, result core.ToolResult) {
		sandboxed := false
		if result.ExecMeta != nil {
			sandboxed = result.ExecMeta.Sandboxed
		}
		p.Send(tui.AgentToolDoneMsg{Name: name, Args: args, Status: result.Status, Sandboxed: sandboxed})

		if !result.Success || (name != "write_file" && name != "edit_file") {
			return
		}

		rewrittenAbs, _ := args["path"].(string)
		meta, ok := worktreeMgr.ConsumeRewrite(name, rewrittenAbs)
		if !ok {
			return
		}

		afterContent, _, err := worktree.ReadFileOrEmpty(meta.RewrittenAbs)
		if err != nil {
			p.Send(tui.AgentErrorMsg{Err: fmt.Errorf("read post-write content for diff: %w", err)})
			return
		}

		p.Send(tui.AgentDiffPreviewMsg{
			RelPath:    meta.RelPath,
			OldContent: meta.BeforeContent,
			NewContent: afterContent,
		})

		approved := make(chan bool, 1)
		p.Send(tui.AgentDiffApprovalMsg{RelPath: meta.RelPath, Approved: approved})
		select {
		case ok := <-approved:
			if ok {
				session := worktreeMgr.CurrentSession()
				if session == nil {
					p.Send(tui.AgentErrorMsg{Err: fmt.Errorf("approve diff: worktree session unavailable")})
					return
				}
				if err := worktree.PromoteRewrite(meta, session.RepoRoot); err != nil {
					p.Send(tui.AgentErrorMsg{Err: fmt.Errorf("approve diff: failed to promote %s: %w", meta.RelPath, err)})
					return
				}
				p.Send(tui.AgentMessageMsg(fmt.Sprintf("Approved diff for `%s`; promoted to current branch", meta.RelPath)))
				return
			}
			if err := worktree.RevertRewrite(meta); err != nil {
				p.Send(tui.AgentErrorMsg{Err: fmt.Errorf("reject diff: failed to revert %s: %w", meta.RelPath, err)})
				return
			}
			p.Send(tui.AgentMessageMsg(fmt.Sprintf("Rejected diff for `%s`; reverted in worktree", meta.RelPath)))
		case <-ctx.Done():
			return
		}
	}

	agent.OnMessage = func(content string) {
		p.Send(tui.AgentMessageMsg(content))
	}

	agent.OnContentDelta = func(delta string) {
		p.Send(tui.AgentContentDeltaMsg(delta))
	}

	agent.OnReasoningDelta = func(delta string) {
		p.Send(tui.AgentReasoningDeltaMsg(delta))
	}

	agent.OnPreTaskStart = func(name string) {
		p.Send(tui.AgentPreTaskStartMsg(name))
	}

	agent.OnPreTaskEnd = func(name string) {
		p.Send(tui.AgentPreTaskEndMsg(name))
	}

	agent.OnContextUsage = func(pct float64, totalCost float64) {
		p.Send(tui.AgentContextUsageMsg{Pct: pct, TotalCost: totalCost})
	}

	agent.OnResponseModel = func(model string) {
		p.Send(tui.AgentResponseModelMsg{ModelID: model})
	}

	agent.OnSandboxFallback = func(command string, reason string) bool {
		approved := make(chan bool, 1)
		p.Send(tui.AgentSandboxFallbackMsg{
			Command:  command,
			Reason:   reason,
			Approved: approved,
		})

		// Block until user approves (Enter) or rejects (Esc), or context cancelled
		select {
		case result := <-approved:
			return result
		case <-ctx.Done():
			return false
		}
	}

	// Run the TUI
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			os.Setenv(strings.TrimSpace(k), strings.TrimSpace(v))
		}
	}
}

func startUpdateCheck(ctx context.Context, p *tea.Program, currentVersion string) {
	if os.Getenv("BONO_DISABLE_UPDATE_CHECK") == "1" {
		return
	}
	current := strings.TrimSpace(currentVersion)
	if current == "" || strings.EqualFold(current, "dev") {
		return
	}

	go func(currentTag string) {
		latest, err := fetchLatestReleaseTag(updateCheckTimeout)
		if err != nil || latest == "" {
			return
		}
		if !isNewerVersion(latest, currentTag) {
			return
		}
		msg := fmt.Sprintf("new version available: %s (rerun install command)", latest)
		select {
		case <-ctx.Done():
			return
		default:
			p.Send(tui.UpdateBannerMsg{Text: msg})
		}
	}(current)
}

func fetchLatestReleaseTag(timeout time.Duration) (string, error) {
	type latestRelease struct {
		TagName string `json:"tag_name"`
	}

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+releaseRepo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "bono-update-check")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var payload latestRelease
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.TagName), nil
}

func isNewerVersion(candidate, current string) bool {
	cMaj, cMin, cPatch, ok := parseSemver(candidate)
	if !ok {
		return false
	}
	vMaj, vMin, vPatch, ok := parseSemver(current)
	if !ok {
		return false
	}

	if cMaj != vMaj {
		return cMaj > vMaj
	}
	if cMin != vMin {
		return cMin > vMin
	}
	return cPatch > vPatch
}

func parseSemver(v string) (major, minor, patch int, ok bool) {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return 0, 0, 0, false
	}

	parts := strings.SplitN(s, "-", 2)
	core := parts[0]
	seg := strings.Split(core, ".")
	if len(seg) != 3 {
		return 0, 0, 0, false
	}

	maj, err := strconv.Atoi(seg[0])
	if err != nil {
		return 0, 0, 0, false
	}
	min, err := strconv.Atoi(seg[1])
	if err != nil {
		return 0, 0, 0, false
	}
	pat, err := strconv.Atoi(seg[2])
	if err != nil {
		return 0, 0, 0, false
	}
	return maj, min, pat, true
}
