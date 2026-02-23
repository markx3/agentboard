package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

type column struct {
	title   string
	status  db.TaskStatus
	list    list.Model
	focused bool
	width   int
	height  int
}

func newColumn(title string, status db.TaskStatus) column {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.SetSpacing(0)

	l := list.New(nil, delegate, 0, 0)
	l.Title = title
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	return column{
		title:  title,
		status: status,
		list:   l,
	}
}

func (c *column) SetSize(w, h int) {
	c.width = w
	c.height = h
	c.list.SetSize(w-2, h-4) // Account for padding (border handled by board)
}

func (c *column) SetItems(tasks []db.Task) {
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		items[i] = taskItem{task: t}
	}
	c.list.SetItems(items)
}

func (c *column) SelectedTask() *db.Task {
	item := c.list.SelectedItem()
	if item == nil {
		return nil
	}
	ti := item.(taskItem)
	return &ti.task
}

// SelectTaskByID moves the list cursor to the task with the given ID.
func (c *column) SelectTaskByID(taskID string) {
	for i, item := range c.list.Items() {
		if ti, ok := item.(taskItem); ok && ti.task.ID == taskID {
			c.list.Select(i)
			return
		}
	}
}

func (c column) Update(msg tea.Msg) (column, tea.Cmd) {
	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)
	return c, cmd
}

func (c column) View() string {
	header := headerStyle.Width(c.width - 2).Render(c.title)

	count := countStyle.Width(c.width - 2).
		Render(countLabel(len(c.list.Items())))

	content := header + "\n" + count + "\n" + c.list.View()

	style := columnStyle
	if c.focused {
		style = focusedColumnStyle
	}

	return style.Width(c.width).Height(c.height).Render(content)
}

func countLabel(n int) string {
	if n == 0 {
		return "empty"
	}
	if n == 1 {
		return "1 task"
	}
	return fmt.Sprintf("%d tasks", n)
}
