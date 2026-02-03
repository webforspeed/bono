package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SlashCommand represents a slash command with its name and description.
type SlashCommand struct {
	Name        string
	Description string
}

// DefaultSlashCommands returns the default set of slash commands.
func DefaultSlashCommands() []SlashCommand {
	return slashCommandList(DefaultSlashCommandSpecs())
}

// SlashModal is a component that displays a list of slash commands.
type SlashModal struct {
	commands []SlashCommand
	filtered []SlashCommand
	selected int
	active   bool
	width    int
}

// NewSlashModal creates a new SlashModal with default commands.
func NewSlashModal() SlashModal {
	commands := DefaultSlashCommands()
	return SlashModal{
		commands: commands,
		filtered: commands,
		selected: 0,
		active:   false,
	}
}

// IsActive returns whether the modal is currently visible.
func (s SlashModal) IsActive() bool {
	return s.active
}

// Selected returns the currently selected command index.
func (s SlashModal) Selected() int {
	return s.selected
}

// SelectedCommand returns the currently selected command, or nil if none.
func (s SlashModal) SelectedCommand() *SlashCommand {
	if len(s.filtered) == 0 || s.selected >= len(s.filtered) {
		return nil
	}
	return &s.filtered[s.selected]
}

// SetWidth sets the width of the modal.
func (s *SlashModal) SetWidth(w int) {
	s.width = w
}

// Height returns the height of the modal when active.
func (s SlashModal) Height() int {
	if !s.active || len(s.filtered) == 0 {
		return 0
	}
	// Modal height: number of items + 2 for border
	h := len(s.filtered) + 2
	if h > 8 {
		h = 8 // Max height
	}
	return h
}

// Update updates the modal state based on the current input value.
// Call this after the input value changes.
func (s *SlashModal) Update(inputValue string) {
	if strings.HasPrefix(inputValue, "/") {
		s.active = true
		query := strings.TrimPrefix(inputValue, "/")
		s.filtered = s.filterCommands(query)
		// Reset selection if out of bounds
		if s.selected >= len(s.filtered) {
			s.selected = 0
		}
	} else {
		s.active = false
		s.selected = 0
		s.filtered = s.commands
	}
}

// filterCommands filters commands based on the query string.
func (s SlashModal) filterCommands(query string) []SlashCommand {
	if query == "" {
		return s.commands
	}
	query = strings.ToLower(query)
	var filtered []SlashCommand
	for _, cmd := range s.commands {
		if strings.HasPrefix(strings.ToLower(cmd.Name), query) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

// HandleKey handles keyboard input when the modal is active.
// Returns the selected command value (with /) if selected, and whether the key was handled.
func (s *SlashModal) HandleKey(msg tea.KeyMsg) (selected string, handled bool) {
	if !s.active {
		return "", false
	}

	switch msg.Type {
	case tea.KeyUp:
		if s.selected > 0 {
			s.selected--
		}
		return "", true

	case tea.KeyDown:
		if s.selected < len(s.filtered)-1 {
			s.selected++
		}
		return "", true

	case tea.KeyTab:
		if len(s.filtered) > 0 && s.selected < len(s.filtered) {
			cmd := s.filtered[s.selected]
			s.active = false
			return "/" + cmd.Name, true
		}
		return "", true

	case tea.KeyEsc:
		s.active = false
		s.selected = 0
		return "", true
	}

	return "", false
}

// View renders the slash modal.
func (s SlashModal) View(styles Styles) string {
	if !s.active || len(s.filtered) == 0 {
		return ""
	}

	var items []string
	for i, cmd := range s.filtered {
		// Format: "/command - description"
		cmdPart := styles.SlashCommand.Render("/" + cmd.Name)
		descPart := styles.SlashDescription.Render(" - " + cmd.Description)
		item := cmdPart + descPart

		if i == s.selected {
			item = styles.SlashItemSelected.Render(fmt.Sprintf("  %s", item))
		} else {
			item = styles.SlashItem.Render(fmt.Sprintf("  %s", item))
		}
		items = append(items, item)
	}

	content := strings.Join(items, "\n")
	style := styles.SlashModal
	if s.width > 0 {
		style = style.Width(s.width)
	}
	return style.Render(content)
}
