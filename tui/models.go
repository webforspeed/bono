package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ModelInfo describes a model from the catalog.
type ModelInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Capabilities []string `json:"capabilities"`
	Context      string   `json:"context"`
	Tier         string   `json:"tier"`
}

// ModelSelectedMsg is sent when a model is selected from the picker.
type ModelSelectedMsg struct {
	Model ModelInfo
}

// ModelModal is a picker that displays available models for selection.
type ModelModal struct {
	models   []ModelInfo
	selected int
	active   bool
	width    int
}

// NewModelModal creates a new model picker.
func NewModelModal(models []ModelInfo) ModelModal {
	return ModelModal{
		models: models,
	}
}

// IsActive returns whether the modal is visible.
func (mm ModelModal) IsActive() bool {
	return mm.active
}

// Show activates the modal.
func (mm *ModelModal) Show() {
	mm.active = true
	mm.selected = 0
}

// Hide deactivates the modal.
func (mm *ModelModal) Hide() {
	mm.active = false
}

// SetWidth sets the width of the modal.
func (mm *ModelModal) SetWidth(w int) {
	mm.width = w
}

// Height returns the height of the modal when active.
func (mm ModelModal) Height() int {
	if !mm.active || len(mm.models) == 0 {
		return 0
	}
	h := len(mm.models) + 2 // items + border
	if h > 14 {
		h = 14
	}
	return h
}

// HandleKey handles keyboard input when the modal is active.
func (mm *ModelModal) HandleKey(msg tea.KeyMsg) (cmd tea.Cmd, handled bool) {
	if !mm.active {
		return nil, false
	}

	switch msg.Type {
	case tea.KeyUp:
		if mm.selected > 0 {
			mm.selected--
		}
		return nil, true

	case tea.KeyDown:
		if mm.selected < len(mm.models)-1 {
			mm.selected++
		}
		return nil, true

	case tea.KeyEnter:
		if mm.selected < len(mm.models) {
			model := mm.models[mm.selected]
			mm.active = false
			return func() tea.Msg { return ModelSelectedMsg{Model: model} }, true
		}
		return nil, true

	case tea.KeyEsc:
		mm.active = false
		return nil, true
	}

	return nil, false
}

// View renders the model picker modal.
func (mm ModelModal) View(styles Styles) string {
	if !mm.active || len(mm.models) == 0 {
		return ""
	}

	tierStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	capsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var items []string
	for i, m := range mm.models {
		caps := strings.Join(m.Capabilities, ", ")
		line := fmt.Sprintf("%-22s  %-10s  ctx:%s  [%s]", m.Name, m.Provider, m.Context, caps)

		if i == mm.selected {
			item := styles.SlashItemSelected.Render("▸ " + line)
			items = append(items, item)
		} else {
			name := styles.SlashCommand.Render("  " + m.Name)
			rest := fmt.Sprintf("  %-10s  ctx:%s  [%s]", m.Provider, m.Context, caps)
			_ = capsStyle
			item := name + tierStyle.Render(rest)
			items = append(items, item)
		}
	}

	content := strings.Join(items, "\n")
	style := styles.SlashModal
	if mm.width > 0 {
		style = style.Width(mm.width)
	}
	return style.Render(content)
}
