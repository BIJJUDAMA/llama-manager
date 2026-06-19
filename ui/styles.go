package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7D56F4") // Purple
	ColorSecondary = lipgloss.Color("#04B575") // Green (Success/Launch)
	ColorBorder    = lipgloss.Color("#3C3C3C") // Dark Gray
	ColorSelected  = lipgloss.Color("#2E2E3E") // Subtle highlight
	ColorAccent    = lipgloss.Color("#FF5F87") // Coral/Pink
	ColorMuted     = lipgloss.Color("#626262") // Muted gray
	ColorWhite     = lipgloss.Color("#FAFAFA")

	// Box Styles
	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginBottom(1)

	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true).
			Padding(0, 1)

	StyleLeftPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	StyleRightPanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	StyleSelectedListItem = lipgloss.NewStyle().
				Background(ColorSelected).
				Foreground(ColorSecondary).
				Bold(true)

	StyleListItem = lipgloss.NewStyle().
			Foreground(ColorWhite)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	StyleHelpKey = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	StyleSearchActive = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	// Hardware compatibility styling
	ColorSuccess = lipgloss.Color("#04B575") // Green (Fits)
	ColorWarning = lipgloss.Color("#FFB800") // Yellow (Partial)
	ColorDanger  = lipgloss.Color("#FF3B30") // Red (Exceeds)

	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
	StyleDanger  = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true)
)
