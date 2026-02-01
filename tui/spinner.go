package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerType represents the available spinner styles from charmbracelet/bubbles.
type SpinnerType int

const (
	SpinnerLine SpinnerType = iota
	SpinnerDot
	SpinnerMiniDot
	SpinnerJump
	SpinnerPulse
	SpinnerPoints
	SpinnerGlobe
	SpinnerMoon
	SpinnerMonkey
	SpinnerMeter
	SpinnerHamburger
	SpinnerEllipsis
)

// SpinnerTypeNames maps spinner types to their display names.
var SpinnerTypeNames = map[SpinnerType]string{
	SpinnerLine:      "line",
	SpinnerDot:       "dot",
	SpinnerMiniDot:   "minidot",
	SpinnerJump:      "jump",
	SpinnerPulse:     "pulse",
	SpinnerPoints:    "points",
	SpinnerGlobe:     "globe",
	SpinnerMoon:      "moon",
	SpinnerMonkey:    "monkey",
	SpinnerMeter:     "meter",
	SpinnerHamburger: "hamburger",
	SpinnerEllipsis:  "ellipsis",
}

// ParseSpinnerType converts a string name to SpinnerType.
func ParseSpinnerType(name string) SpinnerType {
	for t, n := range SpinnerTypeNames {
		if n == name {
			return t
		}
	}
	return SpinnerDot // default
}

// toSpinnerModel converts our SpinnerType to the bubbles spinner.Spinner type.
func (t SpinnerType) toSpinnerModel() spinner.Spinner {
	switch t {
	case SpinnerLine:
		return spinner.Line
	case SpinnerDot:
		return spinner.Dot
	case SpinnerMiniDot:
		return spinner.MiniDot
	case SpinnerJump:
		return spinner.Jump
	case SpinnerPulse:
		return spinner.Pulse
	case SpinnerPoints:
		return spinner.Points
	case SpinnerGlobe:
		return spinner.Globe
	case SpinnerMoon:
		return spinner.Moon
	case SpinnerMonkey:
		return spinner.Monkey
	case SpinnerMeter:
		return spinner.Meter
	case SpinnerHamburger:
		return spinner.Hamburger
	case SpinnerEllipsis:
		return spinner.Ellipsis
	default:
		return spinner.Dot
	}
}

// SpinnerBar displays a spinner with status text above the input box.
type SpinnerBar struct {
	spinner     spinner.Model
	text        string // text shown when spinner is active (e.g., "Thinking...")
	idleText    string // text shown when spinner is inactive (e.g., working directory)
	width       int
	active      bool
	spinnerType SpinnerType
}

// NewSpinnerBar creates a new SpinnerBar with the specified spinner type.
func NewSpinnerBar(spinnerType SpinnerType) SpinnerBar {
	s := spinner.New()
	s.Spinner = spinnerType.toSpinnerModel()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return SpinnerBar{
		spinner:     s,
		text:        "Thinking...",
		idleText:    "",
		active:      false,
		spinnerType: spinnerType,
	}
}

// SetText updates the status text shown when spinner is active.
func (s *SpinnerBar) SetText(text string) {
	s.text = text
}

// SetIdleText updates the text shown when spinner is inactive (e.g., working directory).
func (s *SpinnerBar) SetIdleText(text string) {
	s.idleText = text
}

// SetWidth sets the width of the spinner bar.
func (s *SpinnerBar) SetWidth(w int) {
	s.width = w
}

// SetActive sets whether the spinner is actively spinning.
func (s *SpinnerBar) SetActive(active bool) {
	s.active = active
}

// IsActive returns whether the spinner is actively spinning.
func (s SpinnerBar) IsActive() bool {
	return s.active
}

// SetSpinnerType changes the spinner style.
func (s *SpinnerBar) SetSpinnerType(t SpinnerType) {
	s.spinnerType = t
	s.spinner.Spinner = t.toSpinnerModel()
}

// GetSpinnerType returns the current spinner type.
func (s SpinnerBar) GetSpinnerType() SpinnerType {
	return s.spinnerType
}

// NextSpinnerType cycles to the next spinner type.
func (s *SpinnerBar) NextSpinnerType() {
	next := (int(s.spinnerType) + 1) % len(SpinnerTypeNames)
	s.SetSpinnerType(SpinnerType(next))
}

// Text returns the current status text.
func (s SpinnerBar) Text() string {
	return s.text
}

// Update handles spinner animation updates.
func (s SpinnerBar) Update(msg tea.Msg) (SpinnerBar, tea.Cmd) {
	if !s.active {
		return s, nil
	}
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// Tick returns the spinner tick command (call this to start/continue animation).
func (s SpinnerBar) Tick() tea.Cmd {
	return s.spinner.Tick
}

// View renders the spinner bar.
func (s SpinnerBar) View(styles Styles) string {
	var content string
	if s.active {
		content = s.spinner.View() + " " + s.text
	} else {
		content = s.idleText
	}

	style := styles.SpinnerBar
	if s.width > 0 {
		style = style.Width(s.width)
	}
	return style.Render(content)
}
