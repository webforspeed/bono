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

	// Build left column (vertical stack)
	var leftColumn string
	if m.modelModal.IsActive() {
		modalView := m.modelModal.View(m.styles)
		leftColumn = lipgloss.JoinVertical(lipgloss.Left,
			viewportView,
			modalView,
			spinnerView,
			inputView,
			statusView,
		)
	} else if m.reasoningModal.IsActive() {
		modalView := m.reasoningModal.View(m.styles)
		leftColumn = lipgloss.JoinVertical(lipgloss.Left,
			viewportView,
			modalView,
			spinnerView,
			inputView,
			statusView,
		)
	} else if m.slashModal.IsActive() {
		slashView := m.slashModal.View(m.styles)
		leftColumn = lipgloss.JoinVertical(lipgloss.Left,
			viewportView,
			slashView,
			spinnerView,
			inputView,
			statusView,
		)
	} else {
		leftColumn = lipgloss.JoinVertical(lipgloss.Left,
			viewportView,
			spinnerView,
			inputView,
			statusView,
		)
	}

	// Sidebar (right column, spans full height)
	sidebarView := m.sidebar.View(m.styles)
	if sidebarView == "" {
		return leftColumn
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, sidebarView)
}
