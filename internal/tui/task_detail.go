package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcosfelipeeipper/agentboard/internal/agent"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

const (
	editFieldTitle = iota
	editFieldDesc
	editFieldAssignee
	editFieldBranch
	editFieldPRUrl
	editFieldCount
)

type taskDetail struct {
	task   db.Task
	width  int
	height int

	// Edit mode
	editing    bool
	editField  int
	inputs     [4]textinput.Model // title, assignee, branch, pr_url
	descInput  textarea.Model
	saveNeeded bool
}

func newTaskDetail(task db.Task) taskDetail {
	return taskDetail{task: task}
}

func (d *taskDetail) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d *taskDetail) enterEditMode() {
	d.editing = true
	d.editField = editFieldTitle

	// Title
	d.inputs[0] = textinput.New()
	d.inputs[0].Placeholder = "Title..."
	d.inputs[0].CharLimit = 500
	d.inputs[0].SetValue(d.task.Title)
	d.inputs[0].Focus()

	// Assignee
	d.inputs[1] = textinput.New()
	d.inputs[1].Placeholder = "Assignee..."
	d.inputs[1].SetValue(d.task.Assignee)

	// Branch
	d.inputs[2] = textinput.New()
	d.inputs[2].Placeholder = "Branch name..."
	d.inputs[2].SetValue(d.task.BranchName)

	// PR URL
	d.inputs[3] = textinput.New()
	d.inputs[3].Placeholder = "PR URL..."
	d.inputs[3].SetValue(d.task.PRUrl)

	// Description
	d.descInput = textarea.New()
	d.descInput.Placeholder = "Description..."
	d.descInput.SetHeight(5)
	d.descInput.SetValue(d.task.Description)

	fieldWidth := d.width/2 - 4
	for i := range d.inputs {
		d.inputs[i].Width = fieldWidth
	}
	d.descInput.SetWidth(fieldWidth)
}

func (d *taskDetail) focusField(field int) {
	// Blur all
	for i := range d.inputs {
		d.inputs[i].Blur()
	}
	d.descInput.Blur()

	d.editField = field
	switch field {
	case editFieldTitle:
		d.inputs[0].Focus()
	case editFieldDesc:
		d.descInput.Focus()
	case editFieldAssignee:
		d.inputs[1].Focus()
	case editFieldBranch:
		d.inputs[2].Focus()
	case editFieldPRUrl:
		d.inputs[3].Focus()
	}
}

func (d *taskDetail) nextField() {
	next := (d.editField + 1) % editFieldCount
	d.focusField(next)
}

func (d *taskDetail) applyEdits() {
	d.task.Title = strings.TrimSpace(d.inputs[0].Value())
	d.task.Description = strings.TrimSpace(d.descInput.Value())
	d.task.Assignee = strings.TrimSpace(d.inputs[1].Value())
	d.task.BranchName = strings.TrimSpace(d.inputs[2].Value())
	d.task.PRUrl = strings.TrimSpace(d.inputs[3].Value())
	d.saveNeeded = true
}

func (d taskDetail) Update(msg tea.Msg) (taskDetail, tea.Cmd) {
	if !d.editing {
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			d.nextField()
			return d, nil
		case "ctrl+s":
			d.applyEdits()
			d.editing = false
			return d, nil
		}
	}

	// Update the focused field
	var cmd tea.Cmd
	switch d.editField {
	case editFieldTitle:
		d.inputs[0], cmd = d.inputs[0].Update(msg)
	case editFieldDesc:
		d.descInput, cmd = d.descInput.Update(msg)
	case editFieldAssignee:
		d.inputs[1], cmd = d.inputs[1].Update(msg)
	case editFieldBranch:
		d.inputs[2], cmd = d.inputs[2].Update(msg)
	case editFieldPRUrl:
		d.inputs[3], cmd = d.inputs[3].Update(msg)
	}
	return d, cmd
}

func (d taskDetail) View() string {
	if d.editing {
		return d.editView()
	}
	return d.readView()
}

func (d taskDetail) readView() string {
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
		if t.AgentActivity != "" {
			lines = append(lines, fmt.Sprintf("Activity: %s", t.AgentActivity))
		}
	}

	if t.BranchName != "" {
		lines = append(lines, fmt.Sprintf("Branch:  %s", t.BranchName))
	}

	if t.PRUrl != "" {
		lines = append(lines, fmt.Sprintf("PR:      %s", t.PRUrl))
	}

	lines = append(lines, "", fmt.Sprintf("Created: %s", t.CreatedAt.Format("2006-01-02 15:04")))
	lines = append(lines, "", helpStyle.Render("esc: close | e: edit | m/M: move | a: spawn agent | v: view | A: kill agent | x: delete"))

	content := strings.Join(lines, "\n")
	return overlayStyle.Width(d.width / 2).Render(content)
}

func (d taskDetail) editView() string {
	title := formTitleStyle.Render("Edit Task")

	fieldNames := []string{"Title", "Description", "Assignee", "Branch", "PR URL"}
	fieldName := fieldNames[d.editField]

	var lines []string
	lines = append(lines, title, "")

	lines = append(lines, "Title:")
	lines = append(lines, d.inputs[0].View(), "")

	lines = append(lines, "Description:")
	lines = append(lines, d.descInput.View(), "")

	lines = append(lines, "Assignee:")
	lines = append(lines, d.inputs[1].View(), "")

	lines = append(lines, "Branch:")
	lines = append(lines, d.inputs[2].View(), "")

	lines = append(lines, "PR URL:")
	lines = append(lines, d.inputs[3].View(), "")

	lines = append(lines, editingHintStyle.Render("Editing: "+fieldName))
	lines = append(lines, helpStyle.Render("tab: next field | ctrl+s: save | esc: cancel"))

	content := strings.Join(lines, "\n")
	return overlayStyle.Width(d.width / 2).Render(content)
}
