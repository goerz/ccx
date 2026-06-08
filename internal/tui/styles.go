package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Adaptive palette. Each entry carries a dark-background value (the original
// hardcoded color) and a light-background counterpart tuned for readability on
// a light terminal. lipgloss picks the variant based on the detected (or
// configured, see ApplyTheme) terminal background.
var (
	// Foreground grayscale ramp (bright -> faint on dark; inverted on light).
	fgBright    = lipgloss.AdaptiveColor{Dark: "#FFFFFF", Light: "#000000"}
	fgVeryLight = lipgloss.AdaptiveColor{Dark: "#E2E8F0", Light: "#0F172A"}
	fgLight     = lipgloss.AdaptiveColor{Dark: "#D1D5DB", Light: "#1F2937"}
	fgMutedZinc = lipgloss.AdaptiveColor{Dark: "#A1A1AA", Light: "#52525B"}
	fgMuted     = lipgloss.AdaptiveColor{Dark: "#9CA3AF", Light: "#4B5563"}
	fgDim       = lipgloss.AdaptiveColor{Dark: "#6B7280", Light: "#6B7280"}
	fgDimmer    = lipgloss.AdaptiveColor{Dark: "#4B5563", Light: "#9CA3AF"}
	fgFaint     = lipgloss.AdaptiveColor{Dark: "#374151", Light: "#D1D5DB"}

	// Hue accents (saturated colors darkened for contrast on light backgrounds).
	huePrimary     = lipgloss.AdaptiveColor{Dark: "#7C3AED", Light: "#6D28D9"}
	hueIndigo      = lipgloss.AdaptiveColor{Dark: "#6366F1", Light: "#4F46E5"}
	hueViolet      = lipgloss.AdaptiveColor{Dark: "#8B5CF6", Light: "#6D28D9"}
	hueVioletLight = lipgloss.AdaptiveColor{Dark: "#A78BFA", Light: "#7C3AED"}
	hueSky         = lipgloss.AdaptiveColor{Dark: "#38BDF8", Light: "#0284C7"}
	hueSkyLight    = lipgloss.AdaptiveColor{Dark: "#7DD3FC", Light: "#0369A1"}
	hueSkyBlue     = lipgloss.AdaptiveColor{Dark: "#87CEEB", Light: "#0284C7"}
	hueBlue        = lipgloss.AdaptiveColor{Dark: "#3B82F6", Light: "#2563EB"}
	hueCyan        = lipgloss.AdaptiveColor{Dark: "#06B6D4", Light: "#0891B2"}
	hueCyanLight   = lipgloss.AdaptiveColor{Dark: "#22D3EE", Light: "#0891B2"}
	hueEmerald     = lipgloss.AdaptiveColor{Dark: "#10B981", Light: "#059669"}
	hueGreen       = lipgloss.AdaptiveColor{Dark: "#22C55E", Light: "#16A34A"}
	hueGreenLight  = lipgloss.AdaptiveColor{Dark: "#4ADE80", Light: "#16A34A"}
	hueLime        = lipgloss.AdaptiveColor{Dark: "#84CC16", Light: "#4D7C0F"}
	hueAmber       = lipgloss.AdaptiveColor{Dark: "#F59E0B", Light: "#B45309"}
	hueYellow      = lipgloss.AdaptiveColor{Dark: "#FBBF24", Light: "#B45309"}
	hueOrange      = lipgloss.AdaptiveColor{Dark: "#FB923C", Light: "#C2410C"}
	hueRed         = lipgloss.AdaptiveColor{Dark: "#EF4444", Light: "#DC2626"}
	hueRedLight    = lipgloss.AdaptiveColor{Dark: "#F87171", Light: "#DC2626"}
	huePink        = lipgloss.AdaptiveColor{Dark: "#EC4899", Light: "#DB2777"}
	huePinkLight   = lipgloss.AdaptiveColor{Dark: "#F472B6", Light: "#DB2777"}
	huePinkPale    = lipgloss.AdaptiveColor{Dark: "#F9A8D4", Light: "#DB2777"}

	// Panel / row backgrounds (dark slate on dark; light tints on light).
	bgPanel      = lipgloss.AdaptiveColor{Dark: "#1E293B", Light: "#E2E8F0"}
	bgBlueDark   = lipgloss.AdaptiveColor{Dark: "#2A3A5C", Light: "#DBEAFE"}
	bgBlueDarker = lipgloss.AdaptiveColor{Dark: "#3B4D7A", Light: "#BFDBFE"}

	// ANSI-indexed accents used in a few spots, given adaptive hex on light.
	ansiCyan   = lipgloss.AdaptiveColor{Dark: "6", Light: "#0E7490"}
	ansiGray   = lipgloss.AdaptiveColor{Dark: "240", Light: "#9CA3AF"}
	ansiOrange = lipgloss.AdaptiveColor{Dark: "214", Light: "#B45309"}
)

// resolveHex returns the hex string for a color under the active theme. It is
// used where colors must be emitted as raw ANSI rather than through lipgloss
// styling (e.g. manually drawn split borders).
func resolveHex(c lipgloss.TerminalColor) string {
	switch v := c.(type) {
	case lipgloss.Color:
		return string(v)
	case lipgloss.AdaptiveColor:
		if lipgloss.HasDarkBackground() {
			return v.Dark
		}
		return v.Light
	}
	return ""
}

// ApplyTheme selects which variant of the adaptive palette is used.
//
//	"light" -> force the light-background palette
//	"dark"  -> force the dark-background palette
//	"" / "auto" / anything else -> detect, defaulting to light unless we are
//	                               confident the background is dark
//
// Auto-detection can fail over some SSH/tmux setups, so an explicit value is the
// reliable way to pin the color scheme.
func ApplyTheme(theme string) {
	switch theme {
	case "light":
		lipgloss.SetHasDarkBackground(false)
	case "dark":
		lipgloss.SetHasDarkBackground(true)
	default:
		lipgloss.SetHasDarkBackground(autoDarkBackground())
	}
}

// autoDarkBackground reports whether the terminal background is dark, biased
// toward light: it only returns true when there is positive evidence of a dark
// background. The terminal's OSC 11 background query is reliable in a direct
// terminal but silently fails (and assumes dark) when multiplexed by tmux or
// screen, so a dark result is only trusted outside a multiplexer.
func autoDarkBackground() bool {
	// COLORFGBG is an explicit signal set by several terminals (konsole, rxvt,
	// etc.) and survives tmux, so it wins when present and unambiguous.
	if dark, ok := colorfgbgDark(); ok {
		return dark
	}
	if os.Getenv("TMUX") != "" || os.Getenv("STY") != "" {
		return false
	}
	return lipgloss.HasDarkBackground()
}

// colorfgbgDark interprets the COLORFGBG environment variable ("fg;bg" or
// "fg;default;bg", where bg is an ANSI color index). It returns (dark, true)
// when the value is present and unambiguous, or (_, false) otherwise.
func colorfgbgDark() (bool, bool) {
	v := os.Getenv("COLORFGBG")
	if v == "" {
		return false, false
	}
	parts := strings.Split(v, ";")
	switch parts[len(parts)-1] {
	case "0", "1", "2", "3", "4", "5", "6", "8":
		return true, true
	case "7", "9", "10", "11", "12", "13", "14", "15":
		return false, true
	}
	return false, false
}

var (
	colorPrimary       = huePrimary
	colorTitleBg       = bgPanel // subtle dark bg for title bar
	colorDim           = fgDim
	colorAccent        = hueEmerald
	colorUser          = hueBlue
	colorAssistant     = hueAmber
	colorError         = hueRed
	colorWorktree      = hueViolet
	colorFilter        = huePink
	colorBorderFocused = hueSky
	colorBorderDim     = fgFaint

	helpStyle = lipgloss.NewStyle().Foreground(colorDim)

	userLabelStyle      = lipgloss.NewStyle().Foreground(colorUser).Bold(true)
	assistantLabelStyle = lipgloss.NewStyle().Foreground(colorAssistant).Bold(true)
	toolStyle           = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
	toolBlockStyle      = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	errorStyle          = lipgloss.NewStyle().Foreground(colorError)
	dimStyle            = lipgloss.NewStyle().Foreground(colorDim)
	selectedStyle       = lipgloss.NewStyle().Foreground(fgLight)
	selectedRowStyle    = lipgloss.NewStyle().Background(bgPanel)
	filterBadge         = lipgloss.NewStyle().Foreground(colorFilter).Bold(true)
	teamBadge           = lipgloss.NewStyle().Foreground(hueCyan).Bold(true)
	agentBadgeStyle     = lipgloss.NewStyle().Foreground(hueCyan).Bold(true)
	compactBadgeStyle   = lipgloss.NewStyle().Foreground(hueVioletLight).Bold(true)
	taskBadgeStyle      = lipgloss.NewStyle().Foreground(hueOrange).Bold(true)
	memoryBadge         = lipgloss.NewStyle().Foreground(hueYellow).Bold(true)
	planBadge           = lipgloss.NewStyle().Foreground(hueVioletLight).Bold(true)
	liveBadge           = lipgloss.NewStyle().Foreground(hueGreen).Bold(true)
	busyBadge           = lipgloss.NewStyle().Foreground(hueAmber).Bold(true)
	forkBadge           = lipgloss.NewStyle().Foreground(hueAmber).Bold(true)
	hereBadge           = lipgloss.NewStyle().Foreground(huePinkLight).Bold(true)
	bgBadgeStyle        = lipgloss.NewStyle().Foreground(hueCyanLight).Bold(true)
	waitBadgeStyle      = lipgloss.NewStyle().Foreground(hueYellow).Bold(true)
	doneBadgeStyle      = lipgloss.NewStyle().Foreground(hueEmerald).Bold(true)
	stuckBadgeStyle     = lipgloss.NewStyle().Foreground(hueRed).Bold(true)
	customBadgeStyle    = lipgloss.NewStyle().Foreground(hueLime).Bold(true).Italic(true)
	blockCursorStyle    = lipgloss.NewStyle().Foreground(hueSky).Bold(true)
	blockSelectedBg     = lipgloss.NewStyle().Background(bgPanel)
	previewBorder       = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), true, false, false, false).
				BorderForeground(colorDim)

	// Message list: tool-only continuation rows
	toolOnlyLabelStyle = lipgloss.NewStyle().Foreground(colorDim)
	toolOnlySepStyle   = lipgloss.NewStyle().Foreground(fgDimmer)
	acDimStyle         = lipgloss.NewStyle().Foreground(colorDim).Italic(true)

	// Conversation preview
	convCursorStyle = lipgloss.NewStyle().Foreground(hueSky).Bold(true)
	convSepStyle    = lipgloss.NewStyle().Foreground(fgFaint)

	// Search match highlight
	matchHighlight = lipgloss.NewStyle().Foreground(huePinkPale).Bold(true)

	// Help line: shortcut keys vs description text
	helpKeyStyle = lipgloss.NewStyle().Foreground(fgMuted)

	// Stats rendering (shared across renderSessionStats, renderGlobalStats, timelines)
	statTitleStyle  = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	statNumStyle    = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	statAccentStyle = lipgloss.NewStyle().Foreground(colorAccent)
	statInputStyle  = lipgloss.NewStyle().Foreground(colorUser)
	statOutputStyle = lipgloss.NewStyle().Foreground(colorAssistant)
	statCostStyle   = lipgloss.NewStyle().Foreground(ansiOrange)

	// Multi-select checkmark
	selectMarkStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)

	// Task status icons in conversation
	taskDoneStyle       = lipgloss.NewStyle().Foreground(colorAccent)
	taskInProgressStyle = lipgloss.NewStyle().Foreground(colorAssistant)

	// Skill and hook styles for message detail
	skillBlockStyle = lipgloss.NewStyle().Foreground(hueVioletLight).Bold(true)
	hookBadgeStyle  = lipgloss.NewStyle().Foreground(hueOrange)
	hookDetailStyle = lipgloss.NewStyle().Foreground(fgMuted).Italic(true)
)
