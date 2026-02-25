package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/markx3/agentboard/internal/agent"
	"github.com/markx3/agentboard/internal/db"
)

type taskItem struct {
	task     db.Task
	depCount int
}

func (t taskItem) Title() string {
	prefix := t.statusPrefix()
	if len(t.task.BlockedBy) > 0 {
		prefix = blockedStyle.Render("🔒 ") + prefix
	}
	return prefix + t.task.Title
}

func (t taskItem) Description() string {
	// Prefer activity when agent is active
	if t.task.AgentActivity != "" {
		activity := t.task.AgentActivity
		if len(activity) > 30 {
			activity = activity[:27] + "..."
		}
		return "▸ " + activity
	}

	var parts []string
	if t.task.Assignee != "" {
		parts = append(parts, fmt.Sprintf("@%s", t.task.Assignee))
	}
	if badge := t.enrichmentBadge(); badge != "" {
		parts = append(parts, badge)
	}
	if t.depCount > 0 {
		parts = append(parts, fmt.Sprintf("[%d deps]", t.depCount))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

func (t taskItem) enrichmentBadge() string {
	switch t.task.EnrichmentStatus {
	case db.EnrichmentPending:
		return enrichmentPendingStyle.Render("[enrich:pending]")
	case db.EnrichmentEnriching:
		return enrichmentActiveStyle.Render("[enriching...]")
	case db.EnrichmentDone:
		return enrichmentDoneStyle.Render("[enriched]")
	case db.EnrichmentError:
		return enrichmentErrorStyle.Render("[enrich:error]")
	default:
		return ""
	}
}

func (t taskItem) FilterValue() string {
	return t.task.Title
}

func (t taskItem) statusPrefix() string {
	switch {
	case t.task.AgentStatus == db.AgentActive:
		prefix := "● "
		if t.task.SkipPermissions {
			prefix = "●! "
		}
		label := agentAbbrev(t.task.AgentName)
		elapsed := formatElapsed(t.task.AgentStartedAt)
		if elapsed != "" {
			return agentActiveStyle.Render(prefix + label + " " + elapsed + " ")
		}
		return agentActiveStyle.Render(prefix + label + " ")
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

func agentAbbrev(agentName string) string {
	if r := agent.GetRunner(agentName); r != nil {
		name := r.Name()
		words := []rune(name)
		if len(words) >= 2 {
			for i, ch := range name {
				if ch == ' ' && i+1 < len(name) {
					return string([]rune(name)[0:1]) + string([]rune(name)[i+1:i+2])
				}
			}
			return string(words[:2])
		}
		return name
	}
	if len(agentName) >= 2 {
		return agentName[:2]
	}
	return agentName
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
