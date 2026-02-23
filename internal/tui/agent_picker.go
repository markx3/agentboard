package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcosfelipeeipper/agentboard/internal/agent"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

type agentPicker struct {
	runners  []agent.AgentRunner
	selected int
	task     db.Task
	width    int
	height   int
}

// agentSelectedMsg is emitted when the user picks an agent from the picker.
type agentSelectedMsg struct {
	task   db.Task
	runner agent.AgentRunner
}

func newAgentPicker(runners []agent.AgentRunner, task db.Task, w, h int) agentPicker {
	// Pre-select the task's previous agent if available
	selected := 0
	if task.AgentName != "" {
		for i, r := range runners {
			if r.ID() == task.AgentName {
				selected = i
				break
			}
		}
	}
	return agentPicker{
		runners:  runners,
		selected: selected,
		task:     task,
		width:    w,
		height:   h,
	}
}

func (p agentPicker) Update(msg tea.Msg) (agentPicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Down) || msg.String() == "j":
			if p.selected < len(p.runners)-1 {
				p.selected++
			}
		case key.Matches(msg, keys.Up) || msg.String() == "k":
			if p.selected > 0 {
				p.selected--
			}
		case msg.String() == "enter":
			return p, func() tea.Msg {
				return agentSelectedMsg{
					task:   p.task,
					runner: p.runners[p.selected],
				}
			}
		}
	}
	return p, nil
}

func (p agentPicker) View() string {
	title := formTitleStyle.Render("Select Agent")

	var items []string
	for i, r := range p.runners {
		cursor := "  "
		if i == p.selected {
			cursor = "▸ "
		}

		label := r.Name()
		if i == p.selected {
			label = agentActiveStyle.Render(label)
		}

		items = append(items, fmt.Sprintf("%s%s", cursor, label))
	}

	help := helpStyle.Render("j/k: navigate | enter: select | esc: cancel")

	content := strings.Join(append(
		[]string{title, ""},
		append(items, "", help)...,
	), "\n")

	return overlayStyle.Width(p.width / 3).Render(content)
}
