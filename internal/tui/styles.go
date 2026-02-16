package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colours
	accentColor  = lipgloss.Color("#7C3AED") // violet
	successColor = lipgloss.Color("#10B981") // green
	warnColor    = lipgloss.Color("#F59E0B") // amber
	mutedColor   = lipgloss.Color("#6B7280") // grey
	errorColor   = lipgloss.Color("#EF4444") // red

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	checkStyle = lipgloss.NewStyle().
			Foreground(successColor)

	uncheckStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	dryRunBadge = lipgloss.NewStyle().
			Background(warnColor).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	configKeyStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Width(18)

	configValStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))
)
