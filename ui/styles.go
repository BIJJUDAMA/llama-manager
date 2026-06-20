package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	ColorPrimary   lipgloss.Color
	ColorSecondary lipgloss.Color
	ColorBorder    lipgloss.Color
	ColorSelected  lipgloss.Color
	ColorAccent    lipgloss.Color
	ColorMuted     lipgloss.Color
	ColorWhite     lipgloss.Color
	ColorGold      lipgloss.Color
	ColorFocus     lipgloss.Color
	ColorDim       lipgloss.Color

	ColorProgressFilled lipgloss.Color
	ColorProgressEmpty  lipgloss.Color

	ThemeGradientStart string
	ThemeGradientEnd   string

	// Status Badge Styles
	StyleBadgeRunning  lipgloss.Style
	StyleBadgeStopped  lipgloss.Style
	StyleBadgeFailed   lipgloss.Style
	StyleBadgeStarting lipgloss.Style
	StyleTagPill       lipgloss.Style

	// Box Styles
	StyleHeader           lipgloss.Style
	StyleTitle            lipgloss.Style
	StyleLeftPanel        lipgloss.Style
	StyleRightPanel       lipgloss.Style
	StyleSelectedListItem lipgloss.Style
	StyleListItem         lipgloss.Style
	StyleHelp             lipgloss.Style
	StyleHelpKey          lipgloss.Style
	StyleSearchActive     lipgloss.Style

	// Hardware compatibility styling
	ColorSuccess lipgloss.Color
	ColorWarning lipgloss.Color
	ColorDanger  lipgloss.Color

	StyleSuccess lipgloss.Style
	StyleWarning lipgloss.Style
	StyleDanger  lipgloss.Style
)

func init() {
	// Initialize default theme
	ApplyTheme("dracula")
}

func ApplyTheme(themeName string) {
	switch strings.ToLower(themeName) {
	case "sunset":
		ColorPrimary = lipgloss.Color("#FF5F6D")     // Coral Pink
		ColorSecondary = lipgloss.Color("#04B575")   // Keep green for success indicators
		ColorBorder = lipgloss.Color("#4A2E2E")      // Dark red-gray
		ColorSelected = lipgloss.Color("#3D2222")    // Subtle highlight
		ColorAccent = lipgloss.Color("#FFC371")      // Apricot Orange
		ColorMuted = lipgloss.Color("#8A6F6F")       // Muted gray-red
		ColorWhite = lipgloss.Color("#FAFAFA")
		ColorGold = lipgloss.Color("#FFB800")
		ColorFocus = lipgloss.Color("#FF5F6D")
		ColorDim = lipgloss.Color("#4A2E2E")
		ColorProgressFilled = lipgloss.Color("#FF5F6D")
		ColorProgressEmpty = lipgloss.Color("#2E1E1E")
		ThemeGradientStart = "#FF5F6D"
		ThemeGradientEnd = "#FFC371"

	case "nord":
		ColorPrimary = lipgloss.Color("#88C0D0")     // Ice Blue
		ColorSecondary = lipgloss.Color("#A3BE8C")   // Sage Green
		ColorBorder = lipgloss.Color("#3B4252")      // Slate
		ColorSelected = lipgloss.Color("#2E3440")    // Dark Slate
		ColorAccent = lipgloss.Color("#EBCB8B")      // Soft Yellow
		ColorMuted = lipgloss.Color("#65728A")       // Gray-blue
		ColorWhite = lipgloss.Color("#ECEFF4")
		ColorGold = lipgloss.Color("#EBCB8B")
		ColorFocus = lipgloss.Color("#88C0D0")
		ColorDim = lipgloss.Color("#3B4252")
		ColorProgressFilled = lipgloss.Color("#88C0D0")
		ColorProgressEmpty = lipgloss.Color("#242933")
		ThemeGradientStart = "#88C0D0"
		ThemeGradientEnd = "#5E81AC"

	case "cyberpunk":
		ColorPrimary = lipgloss.Color("#FF007F")     // Neon Pink
		ColorSecondary = lipgloss.Color("#00F3FF")   // Neon Cyan
		ColorBorder = lipgloss.Color("#2A0845")      // Deep Purple-blue
		ColorSelected = lipgloss.Color("#2A0033")    // Very dark purple
		ColorAccent = lipgloss.Color("#FFE600")      // Neon Yellow
		ColorMuted = lipgloss.Color("#7F5A83")       // Muted magenta
		ColorWhite = lipgloss.Color("#FFFFFF")
		ColorGold = lipgloss.Color("#FFE600")
		ColorFocus = lipgloss.Color("#FFE600")
		ColorDim = lipgloss.Color("#2A0845")
		ColorProgressFilled = lipgloss.Color("#FF007F")
		ColorProgressEmpty = lipgloss.Color("#1A001A")
		ThemeGradientStart = "#FF007F"
		ThemeGradientEnd = "#00F3FF"

	case "forest":
		ColorPrimary = lipgloss.Color("#2E7D32")     // Forest Green
		ColorSecondary = lipgloss.Color("#81C784")   // Sage
		ColorBorder = lipgloss.Color("#2E3B2E")      // Dark Olive
		ColorSelected = lipgloss.Color("#1B2E1B")    // Deep Green
		ColorAccent = lipgloss.Color("#FFB74D")      // Orange
		ColorMuted = lipgloss.Color("#6B8E23")       // Olive Drab
		ColorWhite = lipgloss.Color("#F1F8E9")
		ColorGold = lipgloss.Color("#FFB74D")
		ColorFocus = lipgloss.Color("#81C784")
		ColorDim = lipgloss.Color("#2E3B2E")
		ColorProgressFilled = lipgloss.Color("#2E7D32")
		ColorProgressEmpty = lipgloss.Color("#121A12")
		ThemeGradientStart = "#2E7D32"
		ThemeGradientEnd = "#81C784"

	case "monochrome":
		ColorPrimary = lipgloss.Color("#FAFAFA")     // White
		ColorSecondary = lipgloss.Color("#A0A0A0")   // Medium Gray
		ColorBorder = lipgloss.Color("#404040")      // Dark Gray
		ColorSelected = lipgloss.Color("#2A2A2A")    // Selected Gray
		ColorAccent = lipgloss.Color("#FAFAFA")      // White
		ColorMuted = lipgloss.Color("#707070")       // Muted Gray
		ColorWhite = lipgloss.Color("#FFFFFF")
		ColorGold = lipgloss.Color("#FFFFFF")
		ColorFocus = lipgloss.Color("#FAFAFA")
		ColorDim = lipgloss.Color("#404040")
		ColorProgressFilled = lipgloss.Color("#FAFAFA")
		ColorProgressEmpty = lipgloss.Color("#181818")
		ThemeGradientStart = "#FAFAFA"
		ThemeGradientEnd = "#606060"

	default: // "dracula" / "dark" / "purple"
		ColorPrimary = lipgloss.Color("#7D56F4")     // Purple
		ColorSecondary = lipgloss.Color("#04B575")   // Green (Success/Launch)
		ColorBorder = lipgloss.Color("#3C3C3C")      // Dark Gray
		ColorSelected = lipgloss.Color("#2E2E3E")    // Subtle highlight
		ColorAccent = lipgloss.Color("#FF5F87")      // Coral/Pink
		ColorMuted = lipgloss.Color("#626262")       // Muted gray
		ColorWhite = lipgloss.Color("#FAFAFA")
		ColorGold = lipgloss.Color("#FFB800")
		ColorFocus = lipgloss.Color("#7D56F4")
		ColorDim = lipgloss.Color("#3C3C3C")
		ColorProgressFilled = lipgloss.Color("#7D56F4")
		ColorProgressEmpty = lipgloss.Color("#2E2E2E")
		ThemeGradientStart = "#7D56F4"
		ThemeGradientEnd = "#FF5F87"
	}

	// Dynamic Hardware Compatibility Colors
	ColorSuccess = lipgloss.Color("#04B575") // Green (Fits)
	ColorWarning = lipgloss.Color("#FFB800") // Yellow (Partial)
	ColorDanger = lipgloss.Color("#FF3B30")  // Red (Exceeds)

	if themeName == "monochrome" {
		ColorSuccess = lipgloss.Color("#FAFAFA")
		ColorWarning = lipgloss.Color("#A0A0A0")
		ColorDanger = lipgloss.Color("#707070")
	}

	// Rebuild Lip Gloss style structs with updated colors
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
		Background(ColorDanger).
		Foreground(ColorWhite).
		Bold(true).
		Padding(0, 1)

	StyleBadgeStarting = lipgloss.NewStyle().
		Background(ColorWarning).
		Foreground(lipgloss.Color("#000000")).
		Bold(true).
		Padding(0, 1)

	StyleTagPill = lipgloss.NewStyle().
		Background(lipgloss.Color("#2E2E2E")).
		Foreground(lipgloss.Color("#D0D0D0")).
		Padding(0, 1)

	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
	StyleDanger = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true)
}

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

func parseHexColor(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		fmt.Sscanf(string(hex[0])+string(hex[0])+string(hex[1])+string(hex[1])+string(hex[2])+string(hex[2]), "%02x%02x%02x", &r, &g, &b)
	} else if len(hex) == 6 {
		fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	}
	return
}

func interpolateColor(startHex, endHex string, fraction float64) lipgloss.Color {
	r1, g1, b1 := parseHexColor(startHex)
	r2, g2, b2 := parseHexColor(endHex)

	r := int(float64(r1) + float64(r2-r1)*fraction)
	g := int(float64(g1) + float64(g2-g1)*fraction)
	b := int(float64(b1) + float64(b2-b1)*fraction)

	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))
}

func RenderGradient(text string, startHex, endHex string) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	if n == 1 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(startHex)).Bold(true).Render(text)
	}

	var sb strings.Builder
	for i, char := range runes {
		fraction := float64(i) / float64(n-1)
		c := interpolateColor(startHex, endHex, fraction)
		sb.WriteString(lipgloss.NewStyle().Foreground(c).Bold(true).Render(string(char)))
	}
	return sb.String()
}
