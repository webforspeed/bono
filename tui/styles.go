// Package tui provides the terminal user interface components for Bono.
package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all the lipgloss styles used by the TUI components.
type Styles struct {
	// Input box
	InputBox lipgloss.Style

	// Spinner bar (above input)
	SpinnerBar lipgloss.Style

	// Status bar (bottom)
	StatusBar lipgloss.Style

	// Slash modal
	SlashModal        lipgloss.Style
	SlashItem         lipgloss.Style
	SlashItemSelected lipgloss.Style
	SlashCommand      lipgloss.Style
	SlashDescription  lipgloss.Style
}

// DefaultStyles returns the default style configuration.
func DefaultStyles() Styles {
	return Styles{
		InputBox: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("62")).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(false).
			BorderRight(false).
			Padding(0, 1),

		SpinnerBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1),

		SlashModal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),

		SlashItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),

		SlashItemSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true),

		SlashCommand: lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")),

		SlashDescription: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
	}
}
