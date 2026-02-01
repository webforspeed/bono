package tui

import "github.com/charmbracelet/lipgloss"

// View renders the complete TUI.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render each component
	viewportView := m.viewport.View()
	spinnerView := m.spinnerBar.View(m.styles)
	inputView := m.input.View(m.styles)
	statusView := m.statusBar.View(m.styles)

	// Conditionally include slash modal
	if m.slashModal.IsActive() {
		slashView := m.slashModal.View(m.styles)
		return lipgloss.JoinVertical(lipgloss.Left,
			viewportView,
			slashView,
			spinnerView,
			inputView,
			statusView,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		viewportView,
		spinnerView,
		inputView,
		statusView,
	)
}
