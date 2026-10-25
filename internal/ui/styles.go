package ui

import "github.com/charmbracelet/lipgloss"

var (
	colBase     = lipgloss.AdaptiveColor{Light: "#1a1a2e", Dark: "#cdd6f4"}
	colMuted    = lipgloss.AdaptiveColor{Light: "#6c7086", Dark: "#6c7086"}
	colAccent   = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"}
	colDir      = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89dceb"}
	colExec     = lipgloss.AdaptiveColor{Light: "#e64553", Dark: "#f38ba8"}
	colLink     = lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	colSel      = lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"}
	colBorder   = lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#313244"}
	colProgress = lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
)

var (
	styleActiveBorder   lipgloss.Style
	styleInactiveBorder lipgloss.Style
	styleStatusBar      lipgloss.Style
	styleHeader         lipgloss.Style
)

func init() {
	styleActiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colAccent)

	styleInactiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorder)

	styleStatusBar = lipgloss.NewStyle().
		Foreground(colBase).
		Background(colAccent)

	styleHeader = lipgloss.NewStyle().
		Foreground(colAccent).
		Bold(true)
}
