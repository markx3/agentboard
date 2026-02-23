package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type notification struct {
	text    string
	expires time.Time
}

type clearNotificationMsg struct{}

func scheduleNotificationClear(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}
