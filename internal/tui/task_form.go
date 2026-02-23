package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type taskForm struct {
	titleInput textinput.Model
	descInput  textarea.Model
	focusTitle bool
	width      int
	height     int
}

func newTaskForm() taskForm {
	ti := textinput.New()
	ti.Placeholder = "Task title..."
	ti.CharLimit = 500
	ti.Focus()

	ta := textarea.New()
	ta.Placeholder = "Description (optional)..."
	ta.SetHeight(5)

	return taskForm{
		titleInput: ti,
		descInput:  ta,
		focusTitle: true,
	}
}

func (f *taskForm) SetSize(w, h int) {
	f.width = w
	f.height = h
	f.titleInput.Width = w - 8
	f.descInput.SetWidth(w - 8)
}

func (f taskForm) Update(msg tea.Msg) (taskForm, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if f.focusTitle {
				f.titleInput.Blur()
				f.descInput.Focus()
				f.focusTitle = false
			} else {
				f.descInput.Blur()
				f.titleInput.Focus()
				f.focusTitle = true
			}
			return f, nil
		}
	}

	if f.focusTitle {
		var cmd tea.Cmd
		f.titleInput, cmd = f.titleInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		f.descInput, cmd = f.descInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

func (f taskForm) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e6b450")).Render("New Task")

	focusHint := "Title"
	if !f.focusTitle {
		focusHint = "Description"
	}

	help := helpStyle.Render("tab: switch field | enter: create | esc: cancel")

	content := strings.Join([]string{
		title,
		"",
		"Title:",
		f.titleInput.View(),
		"",
		"Description:",
		f.descInput.View(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("Editing: " + focusHint),
		help,
	}, "\n")

	return overlayStyle.Width(f.width/2).Render(content)
}

func (f taskForm) Title() string {
	return strings.TrimSpace(f.titleInput.Value())
}

func (f taskForm) Desc() string {
	return strings.TrimSpace(f.descInput.Value())
}

func (f *taskForm) Reset() {
	f.titleInput.SetValue("")
	f.descInput.SetValue("")
	f.titleInput.Focus()
	f.descInput.Blur()
	f.focusTitle = true
}
