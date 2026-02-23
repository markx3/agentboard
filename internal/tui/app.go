package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

type overlayType int

const (
	overlayNone overlayType = iota
	overlayForm
	overlayDetail
	overlayHelp
)

type App struct {
	board        kanban
	service      board.Service
	overlay      overlayType
	form         taskForm
	detail       taskDetail
	notification *notification
	width        int
	height       int
	ready        bool
}

func NewApp(svc board.Service) App {
	return App{
		board:   newKanban(),
		service: svc,
		form:    newTaskForm(),
	}
}

func (a App) Init() tea.Cmd {
	return a.loadTasks()
}

func (a App) loadTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.service.ListTasks(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return tasksLoadedMsg{tasks: tasks}
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.board.SetSize(msg.Width, msg.Height)
		a.form.SetSize(msg.Width, msg.Height)
		if a.overlay == overlayDetail {
			a.detail.SetSize(msg.Width, msg.Height)
		}
		a.ready = true
		return a, nil

	case tasksLoadedMsg:
		a.board.LoadTasks(msg.tasks)
		return a, nil

	case taskCreatedMsg:
		a.overlay = overlayNone
		a.form.Reset()
		return a, tea.Batch(
			a.loadTasks(),
			a.notify(fmt.Sprintf("Created: %s", msg.task.Title)),
		)

	case taskMovedMsg:
		return a, tea.Batch(
			a.loadTasks(),
			a.notify(fmt.Sprintf("Moved to %s", msg.newStatus)),
		)

	case taskDeletedMsg:
		return a, tea.Batch(
			a.loadTasks(),
			a.notify("Task deleted"),
		)

	case errMsg:
		return a, a.notify(fmt.Sprintf("Error: %s", msg.err))

	case notifyMsg:
		a.notification = &notification{
			text:    msg.text,
			expires: time.Now().Add(3 * time.Second),
		}
		return a, scheduleNotificationClear(3 * time.Second)

	case clearNotificationMsg:
		if a.notification != nil && time.Now().After(a.notification.expires) {
			a.notification = nil
		}
		return a, nil
	}

	// Route to overlay if active
	if a.overlay != overlayNone {
		return a.updateOverlay(msg)
	}

	return a.updateBoard(msg)
}

func (a App) updateOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Escape):
			a.overlay = overlayNone
			a.form.Reset()
			return a, nil
		}
	}

	switch a.overlay {
	case overlayForm:
		return a.updateForm(msg)
	case overlayDetail:
		return a.updateDetail(msg)
	case overlayHelp:
		if _, ok := msg.(tea.KeyMsg); ok {
			a.overlay = overlayNone
			return a, nil
		}
	}

	return a, nil
}

func (a App) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" && a.form.focusTitle {
			// Tab to description if pressing enter on title
			a.form.titleInput.Blur()
			a.form.descInput.Focus()
			a.form.focusTitle = false
			return a, nil
		}
		if msg.String() == "ctrl+s" || (msg.String() == "enter" && !a.form.focusTitle) {
			title := a.form.Title()
			if title == "" {
				return a, a.notify("Title cannot be empty")
			}
			return a, a.createTask(title, a.form.Desc())
		}
	}

	var cmd tea.Cmd
	a.form, cmd = a.form.Update(msg)
	return a, cmd
}

func (a App) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.MoveRight):
			return a, a.moveTask(a.detail.task.ID, a.nextStatus(a.detail.task.Status))
		case key.Matches(msg, keys.MoveLeft):
			return a, a.moveTask(a.detail.task.ID, a.prevStatus(a.detail.task.Status))
		case key.Matches(msg, keys.Delete):
			a.overlay = overlayNone
			return a, a.deleteTask(a.detail.task.ID)
		}
	}

	var cmd tea.Cmd
	a.detail, cmd = a.detail.Update(msg)
	return a, cmd
}

func (a App) updateBoard(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return a, tea.Quit
		case key.Matches(msg, keys.Left):
			a.board.FocusLeft()
			return a, nil
		case key.Matches(msg, keys.Right):
			a.board.FocusRight()
			return a, nil
		case key.Matches(msg, keys.New):
			a.overlay = overlayForm
			a.form.Reset()
			return a, a.form.titleInput.Focus()
		case key.Matches(msg, keys.Enter):
			if task := a.board.SelectedTask(); task != nil {
				a.detail = newTaskDetail(*task)
				a.detail.SetSize(a.width, a.height)
				a.overlay = overlayDetail
			}
			return a, nil
		case key.Matches(msg, keys.MoveRight):
			if task := a.board.SelectedTask(); task != nil {
				return a, a.moveTask(task.ID, a.board.NextColumn())
			}
			return a, nil
		case key.Matches(msg, keys.MoveLeft):
			if task := a.board.SelectedTask(); task != nil {
				return a, a.moveTask(task.ID, a.board.PrevColumn())
			}
			return a, nil
		case key.Matches(msg, keys.Delete):
			if task := a.board.SelectedTask(); task != nil {
				return a, a.deleteTask(task.ID)
			}
			return a, nil
		case key.Matches(msg, keys.Help):
			a.overlay = overlayHelp
			return a, nil
		}
	}

	var cmd tea.Cmd
	a.board, cmd = a.board.Update(msg)
	return a, cmd
}

func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	boardView := a.board.View()
	statusBar := a.board.statusBar()

	if a.notification != nil {
		statusBar = notificationStyle.Render(a.notification.text)
	}

	help := helpStyle.Render(" h/l:columns  j/k:tasks  o:new  m/M:move  enter:open  x:delete  ?:help  q:quit")

	mainView := lipgloss.JoinVertical(lipgloss.Left, boardView, statusBar, help)

	// Overlay rendering
	switch a.overlay {
	case overlayForm:
		return a.renderOverlay(mainView, a.form.View())
	case overlayDetail:
		return a.renderOverlay(mainView, a.detail.View())
	case overlayHelp:
		return a.renderOverlay(mainView, a.helpView())
	}

	return mainView
}

func (a App) renderOverlay(bg, overlay string) string {
	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#333333")),
	)
}

func (a App) helpView() string {
	help := `Agentboard - Keyboard Shortcuts

Navigation:
  h / ←     Previous column
  l / →     Next column
  j / ↓     Next task
  k / ↑     Previous task

Actions:
  o         Create new task
  m         Move task right
  M         Move task left
  enter     Open task detail
  x         Delete task
  /         Search tasks

General:
  ?         Toggle help
  esc       Close overlay
  q         Quit

Press any key to close.`

	return overlayStyle.Width(a.width / 3).Render(help)
}

// Command helpers

func (a App) notify(text string) tea.Cmd {
	return func() tea.Msg {
		return notifyMsg{text: text}
	}
}

func (a App) createTask(title, description string) tea.Cmd {
	return func() tea.Msg {
		task, err := a.service.CreateTask(context.Background(), title, description)
		if err != nil {
			return errMsg{err}
		}
		return taskCreatedMsg{task: task}
	}
}

func (a App) moveTask(id string, newStatus db.TaskStatus) tea.Cmd {
	return func() tea.Msg {
		if err := a.service.MoveTask(context.Background(), id, newStatus); err != nil {
			return errMsg{err}
		}
		return taskMovedMsg{taskID: id, newStatus: newStatus}
	}
}

func (a App) deleteTask(id string) tea.Cmd {
	return func() tea.Msg {
		if err := a.service.DeleteTask(context.Background(), id); err != nil {
			return errMsg{err}
		}
		return taskDeletedMsg{taskID: id}
	}
}

func (a App) nextStatus(current db.TaskStatus) db.TaskStatus {
	for i, s := range columnOrder {
		if s == current && i < len(columnOrder)-1 {
			return columnOrder[i+1]
		}
	}
	return current
}

func (a App) prevStatus(current db.TaskStatus) db.TaskStatus {
	for i, s := range columnOrder {
		if s == current && i > 0 {
			return columnOrder[i-1]
		}
	}
	return current
}
