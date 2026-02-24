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

	// Five-color agent status indicators
	agentDoneStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")) // Green
	agentCompletedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd")) // Cyan/Teal
	agentActiveStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f1fa8c")) // Yellow
	agentErrorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")) // Red
	agentIdleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")) // Gray

	// Subtle background tints for card items
	cardDoneBg      = lipgloss.NewStyle().Background(lipgloss.Color("#1a3a2a"))
	cardCompletedBg = lipgloss.NewStyle().Background(lipgloss.Color("#1a2a3a"))
	cardActiveBg    = lipgloss.NewStyle().Background(lipgloss.Color("#3a3a1a"))
	cardErrorBg     = lipgloss.NewStyle().Background(lipgloss.Color("#3a1a1a"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	countStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Align(lipgloss.Center)

	formTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#e6b450"))

	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#e6b450"))

	editingHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666"))

	tunnelConnectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#50fa7b")) // Green

	tunnelDisconnectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ff5555")) // Red

	tunnelURLStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8be9fd")) // Cyan

	peerCountStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#bd93f9")) // Purple

	// Enrichment status styles
	enrichmentPendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")) // Gray
	enrichmentActiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f1fa8c")) // Yellow
	enrichmentDoneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")) // Green
	enrichmentErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")) // Red
)
