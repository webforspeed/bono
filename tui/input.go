package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputBox wraps a textinput with border styling.
type InputBox struct {
	textInput textinput.Model
	width     int
}

// NewInputBox creates a new InputBox with placeholder text.
func NewInputBox() InputBox {
	ti := textinput.New()
	ti.Placeholder = "Type a message or ask a question..."
	ti.Focus()
	ti.CharLimit = 0 // No limit
	return InputBox{textInput: ti}
}

// Value returns the current input value.
func (i InputBox) Value() string {
	return i.textInput.Value()
}

// SetValue sets the input value.
func (i *InputBox) SetValue(s string) {
	i.textInput.SetValue(s)
}

// Reset clears the input.
func (i *InputBox) Reset() {
	i.textInput.Reset()
}

// SetWidth sets the width of the input box.
// The style is used to calculate horizontal padding for the scrolling viewport.
func (i *InputBox) SetWidth(w int, style lipgloss.Style) {
	i.width = w

	// Calculate the available width for the text input's scrolling viewport.
	// Subtract: horizontal padding + prompt width
	padding := style.GetHorizontalFrameSize()
	promptWidth := lipgloss.Width(i.textInput.Prompt)
	viewportWidth := w - padding - promptWidth
	if viewportWidth > 0 {
		i.textInput.Width = viewportWidth
	}
}

// Focus focuses the input.
func (i *InputBox) Focus() tea.Cmd {
	return i.textInput.Focus()
}

// Blur removes focus from the input.
func (i *InputBox) Blur() {
	i.textInput.Blur()
}

// Focused returns whether the input is focused.
func (i InputBox) Focused() bool {
	return i.textInput.Focused()
}

// Update handles input events.
func (i InputBox) Update(msg tea.Msg) (InputBox, tea.Cmd) {
	var cmd tea.Cmd
	i.textInput, cmd = i.textInput.Update(msg)
	return i, cmd
}

// View renders the input box with horizontal border lines and padding.
func (i InputBox) View(styles Styles) string {
	return styles.InputBox.Render(i.textInput.View())
}
