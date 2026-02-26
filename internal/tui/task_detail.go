package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/markx3/agentboard/internal/agent"
	"github.com/markx3/agentboard/internal/board"
	"github.com/markx3/agentboard/internal/db"
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
	task         db.Task
	dependencies []string
	comments     []db.Comment
	width        int
	height       int
	vp           viewport.Model

	// Edit mode
	editing    bool
	editField  int
	inputs     [4]textinput.Model
	descInput  textarea.Model
	titleEmpty bool
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
	vpW := w/2 - 6
	vpH := h - 10
	if vpH < 5 {
		vpH = 5
	}
	d.vp.Width = vpW
	d.vp.Height = vpH
}

func (d *taskDetail) enterEditMode() {
	d.editing = true
	d.editField = editFieldTitle

	d.inputs[0] = textinput.New()
	d.inputs[0].Placeholder = "Title..."
	d.inputs[0].CharLimit = 500
	d.inputs[0].SetValue(d.task.Title)
	d.inputs[0].Focus()

	d.inputs[1] = textinput.New()
	d.inputs[1].Placeholder = "Assignee..."
	d.inputs[1].SetValue(d.task.Assignee)

	d.inputs[2] = textinput.New()
	d.inputs[2].Placeholder = "Branch name..."
	d.inputs[2].SetValue(d.task.BranchName)

	d.inputs[3] = textinput.New()
	d.inputs[3].Placeholder = "PR URL..."
	d.inputs[3].SetValue(d.task.PRUrl)

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

func (d *taskDetail) applyEdits() (bool, tea.Cmd) {
	title := strings.TrimSpace(d.inputs[0].Value())
	if title == "" {
		return false, nil
	}
	d.task.Title = title
	d.task.Description = strings.TrimSpace(d.descInput.Value())
	d.task.Assignee = strings.TrimSpace(d.inputs[1].Value())
	d.task.BranchName = strings.TrimSpace(d.inputs[2].Value())
	d.task.PRUrl = strings.TrimSpace(d.inputs[3].Value())
	t := d.task
	return true, func() tea.Msg {
		return taskSaveRequestedMsg{task: t}
	}
}

func (d taskDetail) Update(msg tea.Msg) (taskDetail, tea.Cmd) {
	if !d.editing {
		// Populate vp.lines so scroll guards (len == 0, AtBottom) work correctly.
		// View() sets content on a local copy that doesn't persist to a.detail.
		d.vp.SetContent(d.buildReadContent())
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "up", "k":
				d.vp.LineUp(1)
			case "down", "j":
				d.vp.LineDown(1)
			case "pgup", "ctrl+b":
				d.vp.HalfViewUp()
			case "pgdown", "ctrl+f":
				d.vp.HalfViewDown()
			case "g":
				d.vp.GotoTop()
			case "G":
				d.vp.GotoBottom()
			}
		}
		return d, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			d.nextField()
			return d, nil
		case "ctrl+s":
			ok, cmd := d.applyEdits()
			if !ok {
				d.titleEmpty = true
				return d, nil
			}
			d.editing = false
			return d, cmd
		}
	}

	var cmd tea.Cmd
	switch d.editField {
	case editFieldTitle:
		d.titleEmpty = false
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

// buildReadContent assembles the scrollable body text for the read view.
// It is called from both Update (to populate vp.lines so scroll works) and
// readView (to render the current frame).
func (d taskDetail) buildReadContent() string {
	t := d.task
	w := d.vp.Width

	// wrap word-wraps plain text to the viewport width, preserving existing newlines.
	wrap := func(s string) string {
		if w <= 0 {
			return s
		}
		return lipgloss.NewStyle().Width(w).Render(s)
	}

	title := detailTitleStyle.Render(t.Title)

	var lines []string
	lines = append(lines, title, "")

	if t.Description != "" {
		lines = append(lines, wrap(t.Description), "")
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

	if len(t.BlockedBy) > 0 {
		var shortIDs []string
		for _, id := range t.BlockedBy {
			if len(id) > 8 {
				shortIDs = append(shortIDs, id[:8])
			} else {
				shortIDs = append(shortIDs, id)
			}
		}
		lines = append(lines, blockedStyle.Render(fmt.Sprintf("Blocked: %s", strings.Join(shortIDs, ", "))))
	}

	lines = append(lines, "", fmt.Sprintf("Created: %s", t.CreatedAt.Format("2006-01-02 15:04")))

	if len(d.comments) > 0 {
		lines = append(lines, "", "Comments:")
		for _, c := range d.comments {
			header := fmt.Sprintf("  [%s] %s:", c.CreatedAt.Format("15:04"), c.Author)
			lines = append(lines, header)
			lines = append(lines, wrap(c.Body))
		}
	}

	return strings.Join(lines, "\n")
}

func (d taskDetail) readView() string {
	d.vp.SetContent(d.buildReadContent())
	help := helpStyle.Render("esc:close  e:edit  j/k:scroll  g/G:top/btm  m/M:move  a:agent  v:view  A:kill  x:del  E:enrich")
	inner := d.vp.View() + "\n" + help
	return overlayStyle.Width(d.width / 2).Render(inner)
}

func (d taskDetail) editView() string {
	title := formTitleStyle.Render("Edit Task")

	fieldNames := []string{"Title", "Description", "Assignee", "Branch", "PR URL"}
	fieldName := fieldNames[d.editField]

	var lines []string
	lines = append(lines, title, "")

	lines = append(lines, "Title:")
	lines = append(lines, d.inputs[0].View())
	if d.titleEmpty {
		lines = append(lines, agentErrorStyle.Render("  Title cannot be empty"))
	}
	lines = append(lines, "")

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
