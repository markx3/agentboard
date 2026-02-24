package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcosfelipeeipper/agentboard/internal/agent"
	"github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

type taskDetail struct {
	task         db.Task
	dependencies []string
	comments     []db.Comment
	width        int
	height       int
}

func newTaskDetail(task db.Task, svc board.Service) taskDetail {
	ctx := context.Background()
	deps, _ := svc.ListDependencies(ctx, task.ID)
	comments, _ := svc.ListComments(ctx, task.ID)
	return taskDetail{task: task, dependencies: deps, comments: comments}
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
		displayName := t.AgentName
		if r := agent.GetRunner(t.AgentName); r != nil {
			displayName = r.Name()
		}
		agentStr := fmt.Sprintf("Agent:   %s (%s)", displayName, t.AgentStatus)
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

	if t.EnrichmentStatus != "" && t.EnrichmentStatus != db.EnrichmentNone {
		enrichStr := fmt.Sprintf("Enrich:  %s", t.EnrichmentStatus)
		switch t.EnrichmentStatus {
		case db.EnrichmentEnriching:
			enrichStr = enrichmentActiveStyle.Render(enrichStr)
		case db.EnrichmentDone:
			enrichStr = enrichmentDoneStyle.Render(enrichStr)
		case db.EnrichmentError:
			enrichStr = enrichmentErrorStyle.Render(enrichStr)
		}
		lines = append(lines, enrichStr)
	}

	if len(d.dependencies) > 0 {
		depStrs := make([]string, len(d.dependencies))
		for i, dep := range d.dependencies {
			if len(dep) >= 8 {
				depStrs[i] = dep[:8]
			} else {
				depStrs[i] = dep
			}
		}
		lines = append(lines, fmt.Sprintf("Deps:    %s", strings.Join(depStrs, ", ")))
	}

	lines = append(lines, "", fmt.Sprintf("Created: %s", t.CreatedAt.Format("2006-01-02 15:04")))

	if len(d.comments) > 0 {
		lines = append(lines, "", "Comments:")
		for _, c := range d.comments {
			lines = append(lines, fmt.Sprintf("  [%s] %s: %s", c.CreatedAt.Format("15:04"), c.Author, c.Body))
		}
	}

	lines = append(lines, "", helpStyle.Render("esc: close | m/M: move | a: spawn agent | v: view | A: kill agent | x: delete"))

	content := strings.Join(lines, "\n")
	return overlayStyle.Width(d.width / 2).Render(content)
}
