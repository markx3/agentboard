package tui

import (
	"fmt"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

type taskItem struct {
	task db.Task
}

func (t taskItem) Title() string {
	prefix := ""
	switch t.task.AgentStatus {
	case db.AgentActive:
		prefix = agentActiveStyle.Render("● ")
	case db.AgentError:
		prefix = agentErrorStyle.Render("✗ ")
	default:
		prefix = agentIdleStyle.Render("○ ")
	}
	return prefix + t.task.Title
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
