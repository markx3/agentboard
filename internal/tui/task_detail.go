package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

type taskDetail struct {
	task   db.Task
	width  int
	height int
}

func newTaskDetail(task db.Task) taskDetail {
	return taskDetail{task: task}
}

func (d *taskDetail) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d taskDetail) Update(msg tea.Msg) (taskDetail, tea.Cmd) {
	return d, nil
}

func (d taskDetail) View() string {
	t := d.task

	title := detailTitleStyle.Render(t.Title)

	var lines []string
	lines = append(lines, title, "")

	if t.Description != "" {
		lines = append(lines, t.Description, "")
	}

	lines = append(lines, fmt.Sprintf("Status:  %s", t.Status))

	if t.Assignee != "" {
		lines = append(lines, fmt.Sprintf("Assignee: @%s", t.Assignee))
	}

	if t.AgentName != "" {
		agentStr := fmt.Sprintf("Agent:   %s (%s)", t.AgentName, t.AgentStatus)
		switch t.AgentStatus {
		case db.AgentActive:
			agentStr = agentActiveStyle.Render(agentStr)
		case db.AgentError:
			agentStr = agentErrorStyle.Render(agentStr)
		}
		lines = append(lines, agentStr)
		if t.SkipPermissions && t.AgentStatus == db.AgentActive {
			lines = append(lines, agentActiveStyle.Render("Perms:   skipped"))
		}
	}

	if t.BranchName != "" {
		lines = append(lines, fmt.Sprintf("Branch:  %s", t.BranchName))
	}

	if t.PRUrl != "" {
		lines = append(lines, fmt.Sprintf("PR:      %s", t.PRUrl))
	}

	lines = append(lines, "", fmt.Sprintf("Created: %s", t.CreatedAt.Format("2006-01-02 15:04")))
	lines = append(lines, "", helpStyle.Render("esc: close | m/M: move | a: spawn agent | v: view | A: kill agent | x: delete"))

	content := strings.Join(lines, "\n")
	return overlayStyle.Width(d.width / 2).Render(content)
}
