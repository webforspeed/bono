package tui

// StatusBar displays status information at the bottom of the TUI.
type StatusBar struct {
	text  string
	width int
}

// NewStatusBar creates a new StatusBar with default text.
func NewStatusBar() StatusBar {
	return StatusBar{
		text: "Bono - An agent by Webforspeed • /help for commands • Ctrl+C to exit",
	}
}

// SetText updates the status bar text.
func (s *StatusBar) SetText(text string) {
	s.text = text
}

// SetWidth sets the width of the status bar.
func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

// Text returns the current status bar text.
func (s StatusBar) Text() string {
	return s.text
}

// View renders the status bar.
func (s StatusBar) View(styles Styles) string {
	style := styles.StatusBar
	if s.width > 0 {
		style = style.Width(s.width)
	}
	return style.Render(s.text)
}
