package tui

import "github.com/charmbracelet/lipgloss"

var (
	columnStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4a9a8a"))

	focusedColumnStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#e6b450"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#e6b450")).
			Align(lipgloss.Center)

	taskStyle = lipgloss.NewStyle().
			Padding(0, 1)

	selectedTaskStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Background(lipgloss.Color("#3a3a5a")).
				Foreground(lipgloss.Color("#ffffff"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1)

	notificationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#e6b450")).
				Padding(0, 1)

	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#e6b450")).
			Padding(1, 2)

	agentActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50fa7b"))

	agentErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5555"))

	agentIdleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
)
