package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DiffViewMode int

const (
	DiffViewInline DiffViewMode = iota
	DiffViewSideBySide
)

type diffLineType int

const (
	diffLineContext diffLineType = iota
	diffLineAdded
	diffLineDeleted
)

type diffLine struct {
	Type       diffLineType
	OldLineNum int
	NewLineNum int
	Content    string
}

type DiffViewer struct {
	viewport    viewport.Model
	diffLines   []diffLine
	viewMode    DiffViewMode
	width       int
	height      int
	ready       bool
	oldFilename string
	newFilename string
}

func NewDiffViewer() DiffViewer {
	v := DiffViewer{viewMode: DiffViewInline, width: 80, height: 20}
	v.viewport = viewport.New(v.width, v.height)
	v.ready = true
	return v
}

func (d *DiffViewer) SetSize(width, height int) {
	d.width = width
	d.height = height
	vpHeight := height - 2
	if vpHeight < 1 {
		vpHeight = 1
	}
	d.viewport.Width = width
	d.viewport.Height = vpHeight
	d.viewport.SetContent(d.renderDiff())
}

func (d *DiffViewer) SetContent(oldContent, newContent, oldFilename, newFilename string) {
	d.oldFilename = oldFilename
	d.newFilename = newFilename
	d.diffLines = computeDiffLines(oldContent, newContent)
	d.viewport.SetContent(d.renderDiff())
	d.viewport.GotoTop()
}

func (d *DiffViewer) ToggleMode() {
	if d.viewMode == DiffViewInline {
		d.viewMode = DiffViewSideBySide
	} else {
		d.viewMode = DiffViewInline
	}
	d.viewport.SetContent(d.renderDiff())
}

func (d DiffViewer) Update(msg tea.Msg) (DiffViewer, tea.Cmd) {
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return d, cmd
}

func (d DiffViewer) View() string {
	if !d.ready {
		return "Loading diff..."
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(
		fmt.Sprintf("📄 %s → %s", d.oldFilename, d.newFilename),
	)
	mode := "inline"
	if d.viewMode == DiffViewSideBySide {
		mode = "side-by-side"
	}
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		fmt.Sprintf("↑/↓ scroll • tab: toggle view (%s)", mode),
	)
	return header + "\n" + d.viewport.View() + "\n" + footer
}

func computeDiffLines(oldContent, newContent string) []diffLine {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)
	ops := diffOps(oldLines, newLines)

	result := make([]diffLine, 0, len(ops))
	oldN, newN := 1, 1
	for _, op := range ops {
		switch op.kind {
		case opEqual:
			result = append(result, diffLine{Type: diffLineContext, OldLineNum: oldN, NewLineNum: newN, Content: op.text})
			oldN++
			newN++
		case opDelete:
			result = append(result, diffLine{Type: diffLineDeleted, OldLineNum: oldN, Content: op.text})
			oldN++
		case opInsert:
			result = append(result, diffLine{Type: diffLineAdded, NewLineNum: newN, Content: op.text})
			newN++
		}
	}
	return result
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type opKind int

const (
	opEqual opKind = iota
	opDelete
	opInsert
)

type lineOp struct {
	kind opKind
	text string
}

// diffOps computes a line-level diff via LCS dynamic programming.
func diffOps(a, b []string) []lineOp {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	i, j := 0, 0
	ops := make([]lineOp, 0, n+m)
	for i < n && j < m {
		if a[i] == b[j] {
			ops = append(ops, lineOp{kind: opEqual, text: a[i]})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, lineOp{kind: opDelete, text: a[i]})
			i++
		} else {
			ops = append(ops, lineOp{kind: opInsert, text: b[j]})
			j++
		}
	}
	for i < n {
		ops = append(ops, lineOp{kind: opDelete, text: a[i]})
		i++
	}
	for j < m {
		ops = append(ops, lineOp{kind: opInsert, text: b[j]})
		j++
	}
	return ops
}

func (d DiffViewer) renderDiff() string {
	if d.viewMode == DiffViewSideBySide {
		return d.renderSideBySide()
	}
	return d.renderInline()
}

func (d DiffViewer) renderInline() string {
	var sb strings.Builder
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	addStyle := lipgloss.NewStyle().Background(lipgloss.Color("22")).Foreground(lipgloss.Color("120"))
	delStyle := lipgloss.NewStyle().Background(lipgloss.Color("52")).Foreground(lipgloss.Color("203"))

	for _, line := range d.diffLines {
		oldNum := "    "
		newNum := "    "
		if line.OldLineNum > 0 {
			oldNum = fmt.Sprintf("%4d", line.OldLineNum)
		}
		if line.NewLineNum > 0 {
			newNum = fmt.Sprintf("%4d", line.NewLineNum)
		}
		lineNums := lineNumStyle.Render(oldNum) + " " + lineNumStyle.Render(newNum) + " "

		switch line.Type {
		case diffLineAdded:
			sb.WriteString(lineNums + addStyle.Render("+ "+line.Content) + "\n")
		case diffLineDeleted:
			sb.WriteString(lineNums + delStyle.Render("- "+line.Content) + "\n")
		default:
			sb.WriteString(lineNums + contextStyle.Render("  "+line.Content) + "\n")
		}
	}
	return sb.String()
}

func (d DiffViewer) renderSideBySide() string {
	var sb strings.Builder
	halfWidth := (d.width - 3) / 2
	if halfWidth < 20 {
		halfWidth = 20
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	addStyle := lipgloss.NewStyle().Background(lipgloss.Color("22")).Foreground(lipgloss.Color("120"))
	delStyle := lipgloss.NewStyle().Background(lipgloss.Color("52")).Foreground(lipgloss.Color("203"))

	for _, l := range d.diffLines {
		var left, right string
		switch l.Type {
		case diffLineContext:
			left = formatSide(lineNumStyle, contextStyle, l.OldLineNum, l.Content, halfWidth)
			right = formatSide(lineNumStyle, contextStyle, l.NewLineNum, l.Content, halfWidth)
		case diffLineDeleted:
			left = formatSide(lineNumStyle, delStyle, l.OldLineNum, l.Content, halfWidth)
			right = strings.Repeat(" ", halfWidth)
		case diffLineAdded:
			left = strings.Repeat(" ", halfWidth)
			right = formatSide(lineNumStyle, addStyle, l.NewLineNum, l.Content, halfWidth)
		}
		sb.WriteString(left + " │ " + right + "\n")
	}
	return sb.String()
}

func formatSide(lineNumStyle, contentStyle lipgloss.Style, lineNum int, content string, width int) string {
	n := "    "
	if lineNum > 0 {
		n = fmt.Sprintf("%4d", lineNum)
	}
	maxContentWidth := width - 6
	if maxContentWidth < 1 {
		maxContentWidth = 1
	}
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-1] + "…"
	}
	formatted := lineNumStyle.Render(n) + " " + contentStyle.Render(content)
	w := lipgloss.Width(formatted)
	if w < width {
		formatted += strings.Repeat(" ", width-w)
	}
	return formatted
}
