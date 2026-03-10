package main

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Semantic color variables — set by initColors() in theme.go
var (
	green     color.Color // connected, active states
	red       color.Color // errors
	yellow    color.Color // warnings
	accent    color.Color // titles, active borders, highlights, shortcuts
	textCol   color.Color // primary text
	dimCol    color.Color // labels, inactive text, dim elements
	borderCol color.Color // panel borders, dividers
	base      color.Color // background
)

// Style variables — set by initStyles()
var (
	titleStyle               lipgloss.Style
	titleAccentStyle         lipgloss.Style
	activeBorderStyle        lipgloss.Style
	connectedBorderStyle     lipgloss.Style
	inactiveBorderStyle      lipgloss.Style
	panelTitleStyle          lipgloss.Style
	panelTitleConnectedStyle lipgloss.Style
	panelTitleDimStyle       lipgloss.Style
	itemStyle                lipgloss.Style
	selectedItemStyle        lipgloss.Style
	activeMarkerStyle        lipgloss.Style
	labelStyle               lipgloss.Style
	valueStyle               lipgloss.Style
	connectedStyle           lipgloss.Style
	errorStyle               lipgloss.Style
	warnStyle                lipgloss.Style
	dimStyle                 lipgloss.Style
	shortcutKeyStyle         lipgloss.Style
	helpOverlayStyle         lipgloss.Style
	helpTitleStyle           lipgloss.Style
	inputPromptStyle         lipgloss.Style
	spinnerStyle             lipgloss.Style
	connectedIndicator       string
	disconnectedIndicator    string
)

func initStyles() {
	// Title bar
	titleStyle = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)

	titleAccentStyle = lipgloss.NewStyle().
		Foreground(accent)

	// Panel borders
	activeBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent)

	connectedBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(green)

	inactiveBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderCol)

	// Panel titles (rendered inside top border)
	panelTitleStyle = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true).
		Background(base).
		Padding(0, 1)

	panelTitleConnectedStyle = lipgloss.NewStyle().
		Foreground(green).
		Bold(true).
		Background(base).
		Padding(0, 1)

	panelTitleDimStyle = lipgloss.NewStyle().
		Foreground(dimCol).
		Background(base).
		Padding(0, 1)

	// List items
	itemStyle = lipgloss.NewStyle().
		Foreground(textCol)

	selectedItemStyle = lipgloss.NewStyle().
		Foreground(green).
		Bold(true)

	activeMarkerStyle = lipgloss.NewStyle().
		Foreground(green)

	// Status field icons and labels
	labelStyle = lipgloss.NewStyle().
		Foreground(dimCol).
		Width(14)

	valueStyle = lipgloss.NewStyle().
		Foreground(textCol)

	// Feedback
	connectedStyle = lipgloss.NewStyle().Foreground(green)
	errorStyle = lipgloss.NewStyle().Foreground(red)
	warnStyle = lipgloss.NewStyle().Foreground(yellow)
	dimStyle = lipgloss.NewStyle().Foreground(dimCol)

	// Bottom bar
	shortcutKeyStyle = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)

	// Help overlay
	helpOverlayStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, 3)

	helpTitleStyle = lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)

	// Inline input
	inputPromptStyle = lipgloss.NewStyle().
		Foreground(accent)

	// Spinner
	spinnerStyle = lipgloss.NewStyle().
		Foreground(accent)

	// Connection status indicator
	connectedIndicator = lipgloss.NewStyle().Foreground(green).Bold(true).Render("●")
	disconnectedIndicator = lipgloss.NewStyle().Foreground(dimCol).Render("○")
}
