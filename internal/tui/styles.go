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

	agentDoneStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b"))
	agentCompletedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd"))
	agentActiveStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f1fa8c"))
	agentErrorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555"))
	agentIdleStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

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
				Foreground(lipgloss.Color("#50fa7b"))

	tunnelDisconnectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ff5555"))

	tunnelURLStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8be9fd"))

	peerCountStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#bd93f9"))

	// Enrichment status styles (from HEAD)
	enrichmentPendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	enrichmentActiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f1fa8c"))
	enrichmentDoneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b"))
	enrichmentErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555"))

	// Board summary bar (from main)
	summaryBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8be9fd")).
			Padding(0, 1)

	// Blocked indicator (from main)
	blockedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff79c6"))
)
