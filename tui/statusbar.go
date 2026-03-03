package tui

import "strings"

// StatusBar displays status information at the bottom of the TUI.
type StatusBar struct {
	text  string
	width int
}

const statusBarSuffix = "An agent by Webforspeed • /help for commands • Ctrl+C to exit"

// StatusBarText returns the default footer string with version information.
func StatusBarText(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || strings.EqualFold(version, "dev") {
		return "Bono (dev) - " + statusBarSuffix
	}
	return "Bono " + version + " - " + statusBarSuffix
}

// NewStatusBar creates a new StatusBar with default text.
func NewStatusBar() StatusBar {
	return StatusBar{
		text: StatusBarText("dev"),
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
