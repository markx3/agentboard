package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

type taskItem struct {
	task db.Task
}

func (t taskItem) Title() string {
	return t.statusPrefix() + t.task.Title
}

func (t taskItem) Description() string {
	if t.task.Assignee != "" {
		return fmt.Sprintf("@%s", t.task.Assignee)
	}
	return ""
}

func (t taskItem) FilterValue() string {
	return t.task.Title
}

// statusPrefix returns the color-coded status indicator dot.
// Priority: Active > Done > Completed > Error > Idle.
func (t taskItem) statusPrefix() string {
	switch {
	case t.task.AgentStatus == db.AgentActive:
		elapsed := formatElapsed(t.task.AgentStartedAt)
		if elapsed != "" {
			return agentActiveStyle.Render("● "+elapsed+" ")
		}
		return agentActiveStyle.Render("● ")
	case t.task.Status == db.StatusDone:
		return agentDoneStyle.Render("● ")
	case t.task.AgentStatus == db.AgentCompleted:
		return agentCompletedStyle.Render("● ")
	case t.task.AgentStatus == db.AgentError:
		return agentErrorStyle.Render("✖ ")
	default:
		return agentIdleStyle.Render("○ ")
	}
}

// cardTintStyle returns the background tint style for the task card.
func (t taskItem) cardTintStyle() lipgloss.Style {
	switch {
	case t.task.AgentStatus == db.AgentActive:
		return cardActiveBg
	case t.task.Status == db.StatusDone:
		return cardDoneBg
	case t.task.AgentStatus == db.AgentCompleted:
		return cardCompletedBg
	case t.task.AgentStatus == db.AgentError:
		return cardErrorBg
	default:
		return lipgloss.NewStyle()
	}
}

func formatElapsed(startedAt string) string {
	if startedAt == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}
