package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7D56F4") // Purple
	ColorSecondary = lipgloss.Color("#04B575") // Green (Success/Launch)
	ColorBorder    = lipgloss.Color("#3C3C3C") // Dark Gray
	ColorSelected  = lipgloss.Color("#2E2E3E") // Subtle highlight
	ColorAccent    = lipgloss.Color("#FF5F87") // Coral/Pink
	ColorMuted     = lipgloss.Color("#626262") // Muted gray
	ColorWhite     = lipgloss.Color("#FAFAFA")
	ColorGold      = lipgloss.Color("#FFB800")
	ColorFocus     = lipgloss.Color("#7D56F4")
	ColorDim       = lipgloss.Color("#3C3C3C")

	ColorProgressFilled = lipgloss.Color("#7D56F4")
	ColorProgressEmpty  = lipgloss.Color("#2E2E2E")

	// Status Badge Styles
	StyleBadgeRunning = lipgloss.NewStyle().
				Background(ColorSecondary).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 1)

	StyleBadgeStopped = lipgloss.NewStyle().
				Background(ColorMuted).
				Foreground(ColorWhite).
				Bold(true).
				Padding(0, 1)

	StyleBadgeFailed = lipgloss.NewStyle().
				Background(lipgloss.Color("#FF3B30")).
				Foreground(ColorWhite).
				Bold(true).
				Padding(0, 1)

	StyleBadgeStarting = lipgloss.NewStyle().
				Background(lipgloss.Color("#FFB800")).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 1)

	// Tag pill style
	StyleTagPill = lipgloss.NewStyle().
			Background(lipgloss.Color("#2E2E2E")).
			Foreground(lipgloss.Color("#D0D0D0")).
			Padding(0, 1)

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

func RenderProgressBar(percent float64, width int, filledColor lipgloss.Color) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filledCount := int(percent / 100.0 * float64(width))
	if filledCount < 0 {
		filledCount = 0
	}
	if filledCount > width {
		filledCount = width
	}
	emptyCount := width - filledCount

	filledStr := ""
	if filledCount > 0 {
		filledStr = strings.Repeat("█", filledCount)
	}
	emptyStr := ""
	if emptyCount > 0 {
		emptyStr = strings.Repeat("░", emptyCount)
	}

	filledStyle := lipgloss.NewStyle().Foreground(filledColor)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorBorder)

	return filledStyle.Render(filledStr) + emptyStyle.Render(emptyStr)
}
