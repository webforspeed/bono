package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const sidebarWidth = 42

// SidebarSection is a named group of items rendered with a header.
type SidebarSection struct {
	Header string
	Items  []SidebarItem
}

// SidebarItem is a single line within a section.
type SidebarItem struct {
	Text  string
	Color lipgloss.TerminalColor // nil = use default SidebarItem style
}

// GitStatus holds parsed git state for the sidebar.
type GitStatus struct {
	Branch   string
	Staged   []string // filenames
	Unstaged []string // filenames
}

// Sidebar displays session metadata on the right side of the TUI.
type Sidebar struct {
	modelName       string
	contextUsagePct float64
	totalCost       float64
	cwd             string
	indexedFiles     int
	changedFiles     int // files changed since last index
	indexReady       bool
	reasoningEffort  string // current reasoning effort value (e.g. "high", "" = disabled)
	git              GitStatus
	width, height    int
}

// NewSidebar creates a new Sidebar.
func NewSidebar() Sidebar {
	return Sidebar{}
}

func (s *Sidebar) SetModelName(name string)    { s.modelName = name }
func (s *Sidebar) SetContextUsage(pct float64) { s.contextUsagePct = pct }
func (s *Sidebar) SetTotalCost(cost float64)   { s.totalCost = cost }
func (s *Sidebar) SetCwd(cwd string)           { s.cwd = cwd }
func (s *Sidebar) SetWidth(w int)              { s.width = w }
func (s *Sidebar) SetHeight(h int)             { s.height = h }
func (s *Sidebar) SetGitStatus(g GitStatus)        { s.git = g }
func (s *Sidebar) SetReasoningEffort(effort string) { s.reasoningEffort = effort }

// SetIndexStats updates the workspace index information.
func (s *Sidebar) SetIndexStats(files int) {
	s.indexedFiles = files
	s.indexReady = true
	s.changedFiles = 0
}

// SetChangedFiles updates the count of files changed since last index.
func (s *Sidebar) SetChangedFiles(n int) {
	s.changedFiles = n
}

// ClearIndex resets index state (e.g., no index available).
func (s *Sidebar) ClearIndex() {
	s.indexedFiles = 0
	s.indexReady = false
	s.changedFiles = 0
}

// FetchGitStatus runs git commands and returns the current status.
// Safe to call from any goroutine.
func FetchGitStatus() GitStatus {
	var gs GitStatus

	// Branch name
	if out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		gs.Branch = strings.TrimSpace(string(out))
	}

	// Staged and unstaged files via porcelain format
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return gs
	}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n\r"), "\n") {
		if len(line) < 4 {
			continue
		}
		x, y := line[0], line[1]
		name := strings.TrimSpace(line[3:])

		if x != ' ' && x != '?' {
			gs.Staged = append(gs.Staged, name)
		}
		if y != ' ' && y != '?' {
			gs.Unstaged = append(gs.Unstaged, name)
		}
		// Untracked files (??): show as unstaged
		if x == '?' && y == '?' {
			gs.Unstaged = append(gs.Unstaged, name)
		}
	}

	return gs
}

// sections builds the sidebar content from current data.
// This is the single configuration point — extend here to add new sections/items.
func (s Sidebar) sections() []SidebarSection {
	var sections []SidebarSection

	// MODEL
	session := SidebarSection{Header: "MODEL (/model)"}
	if s.modelName != "" {
		session.Items = append(session.Items, SidebarItem{
			Text:  s.modelName,
			Color: lipgloss.Color("86"),
		})
	}
	sections = append(sections, session)

	// REASONING
	reasoning := SidebarSection{Header: "REASONING (/reasoning)"}
	for _, level := range DefaultReasoningLevels() {
		item := SidebarItem{Text: level.Label}
		if level.Value == s.reasoningEffort {
			item.Color = lipgloss.Color("86") // highlighted = selected
		} else {
			item.Color = lipgloss.Color("241") // dimmed = not selected
		}
		reasoning.Items = append(reasoning.Items, item)
	}
	sections = append(sections, reasoning)

	// CONTEXT
	context := SidebarSection{Header: "CONTEXT (/clear)"}
	if s.contextUsagePct > 0 {
		context.Items = append(context.Items, SidebarItem{
			Text:  fmt.Sprintf("%.0f%% used", s.contextUsagePct),
			Color: contextUsageColor(s.contextUsagePct),
		})
	}
	if s.totalCost > 0 {
		context.Items = append(context.Items, SidebarItem{
			Text: formatCost(s.totalCost),
		})
	}
	sections = append(sections, context)

	// STAGED CHANGES
	staged := SidebarSection{Header: fmt.Sprintf("STAGED CHANGES (%d)", len(s.git.Staged))}
	for _, f := range s.git.Staged {
		staged.Items = append(staged.Items, SidebarItem{
			Text:  f,
			Color: lipgloss.Color("78"),
		})
	}
	sections = append(sections, staged)

	// UNSTAGED CHANGES
	unstaged := SidebarSection{Header: fmt.Sprintf("UNSTAGED CHANGES (%d)", len(s.git.Unstaged))}
	for _, f := range s.git.Unstaged {
		unstaged.Items = append(unstaged.Items, SidebarItem{
			Text:  f,
			Color: lipgloss.Color("214"),
		})
	}
	sections = append(sections, unstaged)

	// INDEX
	idx := SidebarSection{Header: "INDEX (/index)"}
	if s.indexReady {
		idx.Items = append(idx.Items, SidebarItem{
			Text: fmt.Sprintf("%d files indexed", s.indexedFiles),
		})
		if s.changedFiles > 0 {
			idx.Items = append(idx.Items, SidebarItem{
				Text:  fmt.Sprintf("%d files pending reindex", s.changedFiles),
				Color: lipgloss.Color("214"),
			})
		}
	} else {
		idx.Items = append(idx.Items, SidebarItem{
			Text:  "No index",
			Color: lipgloss.Color("241"),
		})
	}
	sections = append(sections, idx)

	return sections
}

// View renders sections generically. Returns "" when width <= 0.
func (s Sidebar) View(styles Styles) string {
	if s.width <= 0 {
		return ""
	}

	headerStyle := styles.SidebarHeader
	itemStyle := styles.SidebarItem

	var lines []string
	for i, sec := range s.sections() {
		if i > 0 {
			lines = append(lines, "") // blank line between sections
		}
		lines = append(lines, headerStyle.Render(sec.Header))
		for _, item := range sec.Items {
			style := itemStyle
			if item.Color != nil {
				style = lipgloss.NewStyle().Foreground(item.Color)
			}
			lines = append(lines, "  "+style.Render(item.Text))
		}
	}

	// CWD and branch pinned near the bottom, aligned with the input text.
	contentLines := len(lines)
	if s.cwd != "" {
		maxLen := s.width - 6 // account for border, padding, indent

		// CWD line — replace $HOME with ~ so it's copy-pasteable into a terminal
		cwd := s.cwd
		if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(cwd, home) {
			cwd = "~" + cwd[len(home):]
		}
		if maxLen > 0 && len(cwd) > maxLen {
			cwd = "..." + cwd[len(cwd)-maxLen+3:]
		}
		cwdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
		cwdLine := cwdStyle.Render(cwd)

		// Branch line
		var branchLine string
		bottomLines := 1 // just the cwd line
		if s.git.Branch != "" {
			branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
			branchLine = branchStyle.Render("main <- " + s.git.Branch)
			bottomLines = 2
		}

		// Position so bottom aligns with input text (2 lines from terminal bottom)
		padding := s.height - contentLines - 2 - bottomLines
		if padding < 1 {
			padding = 1
		}
		lines = append(lines, strings.Repeat("\n", padding)+cwdLine)
		if branchLine != "" {
			lines = append(lines, branchLine)
		}
	}

	content := strings.Join(lines, "\n")

	style := styles.Sidebar.
		Width(s.width).
		Height(s.height)

	return style.Render(content)
}
