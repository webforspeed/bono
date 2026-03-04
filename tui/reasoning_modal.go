package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ReasoningLevel describes a reasoning effort option.
type ReasoningLevel struct {
	Value       string // value sent to API (e.g., "high")
	Label       string // display name
	Description string // short description
}

// DefaultReasoningLevels returns the available reasoning effort levels.
func DefaultReasoningLevels() []ReasoningLevel {
	return []ReasoningLevel{
		{Value: "", Label: "Disabled", Description: "No reasoning effort specified"},
		{Value: "minimal", Label: "Minimal", Description: "~10% of token budget"},
		{Value: "low", Label: "Low", Description: "~20% of token budget"},
		{Value: "medium", Label: "Medium", Description: "~50% of token budget"},
		{Value: "high", Label: "High", Description: "~80% of token budget"},
		{Value: "xhigh", Label: "Extra High", Description: "~95% of token budget"},
	}
}

// ReasoningSelectedMsg is sent when a reasoning level is selected from the picker.
type ReasoningSelectedMsg struct {
	Level ReasoningLevel
}

// ReasoningModal is a picker that displays reasoning effort levels for selection.
type ReasoningModal struct {
	levels   []ReasoningLevel
	selected int
	active   bool
	width    int
}

// NewReasoningModal creates a new reasoning effort picker.
func NewReasoningModal() ReasoningModal {
	return ReasoningModal{
		levels: DefaultReasoningLevels(),
	}
}

// IsActive returns whether the modal is visible.
func (rm ReasoningModal) IsActive() bool {
	return rm.active
}

// Show activates the modal and highlights the current level.
func (rm *ReasoningModal) Show(currentEffort string) {
	rm.active = true
	rm.selected = 0
	for i, l := range rm.levels {
		if l.Value == currentEffort {
			rm.selected = i
			break
		}
	}
}

// Hide deactivates the modal.
func (rm *ReasoningModal) Hide() {
	rm.active = false
}

// SetWidth sets the width of the modal.
func (rm *ReasoningModal) SetWidth(w int) {
	rm.width = w
}

// Height returns the height of the modal when active.
func (rm ReasoningModal) Height() int {
	if !rm.active {
		return 0
	}
	return len(rm.levels) + 2 // items + border
}

// HandleKey handles keyboard input when the modal is active.
func (rm *ReasoningModal) HandleKey(msg tea.KeyMsg) (cmd tea.Cmd, handled bool) {
	if !rm.active {
		return nil, false
	}

	switch msg.Type {
	case tea.KeyUp:
		if rm.selected > 0 {
			rm.selected--
		}
		return nil, true

	case tea.KeyDown:
		if rm.selected < len(rm.levels)-1 {
			rm.selected++
		}
		return nil, true

	case tea.KeyEnter:
		if rm.selected < len(rm.levels) {
			level := rm.levels[rm.selected]
			rm.active = false
			return func() tea.Msg { return ReasoningSelectedMsg{Level: level} }, true
		}
		return nil, true

	case tea.KeyEsc:
		rm.active = false
		return nil, true
	}

	return nil, false
}

// View renders the reasoning modal.
func (rm ReasoningModal) View(styles Styles) string {
	if !rm.active {
		return ""
	}

	descStyle := styles.SlashDescription

	var items []string
	for i, l := range rm.levels {
		line := fmt.Sprintf("%-12s  %s", l.Label, l.Description)

		if i == rm.selected {
			items = append(items, styles.SlashItemSelected.Render("▸ "+line))
		} else {
			name := styles.SlashCommand.Render("  " + l.Label)
			rest := descStyle.Render(fmt.Sprintf("  %s", l.Description))
			items = append(items, name+rest)
		}
	}

	content := strings.Join(items, "\n")
	style := styles.SlashModal
	if rm.width > 0 {
		style = style.Width(rm.width)
	}
	return style.Render(content)
}
