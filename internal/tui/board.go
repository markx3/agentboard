package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

var columnOrder = []db.TaskStatus{
	db.StatusBacklog,
	db.StatusBrainstorm,
	db.StatusPlanning,
	db.StatusInProgress,
	db.StatusReview,
	db.StatusDone,
}

var columnTitles = map[db.TaskStatus]string{
	db.StatusBacklog:    "Backlog",
	db.StatusBrainstorm: "Brainstorm",
	db.StatusPlanning:   "Planning",
	db.StatusInProgress: "In Progress",
	db.StatusReview:     "Review",
	db.StatusDone:       "Done",
}

type kanban struct {
	columns    []column
	focusedCol int
	width      int
	height     int
}

func newKanban() kanban {
	cols := make([]column, len(columnOrder))
	for i, status := range columnOrder {
		cols[i] = newColumn(columnTitles[status], status)
	}
	cols[0].focused = true

	return kanban{
		columns: cols,
	}
}

func (b *kanban) SetSize(w, h int) {
	b.width = w
	b.height = h

	numCols := len(b.columns)
	// Each column's border adds 2 chars (left + right) outside lipgloss Width.
	// Subtract border space before dividing to prevent rightmost column overflow.
	borderW := 2
	availW := w - borderW*numCols
	colWidth := availW / numCols
	colHeight := h - 5 // 5 = summary bar + status bar + help bar + padding

	for i := range b.columns {
		cw := colWidth
		if i == numCols-1 {
			// Give remainder pixels to the last column to fill the terminal width.
			cw = availW - colWidth*(numCols-1)
		}
		b.columns[i].SetSize(cw, colHeight)
	}
}

func (b *kanban) LoadTasks(tasks []db.Task, deps map[string][]string) {
	grouped := make(map[db.TaskStatus][]db.Task)
	for _, t := range tasks {
		grouped[t.Status] = append(grouped[t.Status], t)
	}
	for i, status := range columnOrder {
		b.columns[i].SetItems(grouped[status], deps)
	}
}

func (b *kanban) SelectedTask() *db.Task {
	return b.columns[b.focusedCol].SelectedTask()
}

func (b *kanban) NextColumn() db.TaskStatus {
	idx := b.focusedCol
	if idx < len(columnOrder)-1 {
		return columnOrder[idx+1]
	}
	return columnOrder[idx]
}

func (b *kanban) PrevColumn() db.TaskStatus {
	idx := b.focusedCol
	if idx > 0 {
		return columnOrder[idx-1]
	}
	return columnOrder[idx]
}

func (b kanban) Update(msg tea.Msg) (kanban, tea.Cmd) {
	var cmd tea.Cmd
	b.columns[b.focusedCol], cmd = b.columns[b.focusedCol].Update(msg)
	return b, cmd
}

func (b *kanban) FocusLeft() {
	if b.focusedCol > 0 {
		b.columns[b.focusedCol].focused = false
		b.focusedCol--
		b.columns[b.focusedCol].focused = true
	}
}

func (b *kanban) FocusRight() {
	if b.focusedCol < len(b.columns)-1 {
		b.columns[b.focusedCol].focused = false
		b.focusedCol++
		b.columns[b.focusedCol].focused = true
	}
}

// FocusOnStatus moves column focus to the column matching the given status.
func (b *kanban) FocusOnStatus(status db.TaskStatus) {
	for i, s := range columnOrder {
		if s == status {
			b.columns[b.focusedCol].focused = false
			b.focusedCol = i
			b.columns[b.focusedCol].focused = true
			return
		}
	}
}

// SelectTaskByID selects a task by ID in the currently focused column.
func (b *kanban) SelectTaskByID(taskID string) {
	b.columns[b.focusedCol].SelectTaskByID(taskID)
}

func (b kanban) View() string {
	colViews := make([]string, len(b.columns))
	for i, col := range b.columns {
		colViews[i] = col.View()
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, colViews...)
}

func (b kanban) summaryBar() string {
	var agentActive int
	statusCounts := make(map[db.TaskStatus]int)

	for _, col := range b.columns {
		for _, item := range col.list.Items() {
			if ti, ok := item.(taskItem); ok {
				statusCounts[ti.task.Status]++
				if ti.task.AgentStatus == db.AgentActive {
					agentActive++
				}
			}
		}
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("Agents: %d active", agentActive))

	var taskParts []string
	for _, s := range columnOrder {
		if s == db.StatusDone || s == db.StatusBacklog {
			continue
		}
		if c := statusCounts[s]; c > 0 {
			taskParts = append(taskParts, fmt.Sprintf("%d %s", c, columnTitles[s]))
		}
	}
	if len(taskParts) > 0 {
		parts = append(parts, "Tasks: "+strings.Join(taskParts, ", "))
	}
	parts = append(parts, fmt.Sprintf("%d done", statusCounts[db.StatusDone]))

	return summaryBarStyle.Render("  " + strings.Join(parts, " | "))
}

func (b kanban) statusBar() string {
	task := b.SelectedTask()
	if task == nil {
		return statusBarStyle.Render(fmt.Sprintf("  Column %d/%d", b.focusedCol+1, len(b.columns)))
	}
	return statusBarStyle.Render(
		fmt.Sprintf("  %s | %s | Column %d/%d",
			task.Title, task.Status, b.focusedCol+1, len(b.columns)))
}

// serverStatusBar renders tunnel/connection info for the status bar.
func serverStatusBar(tunnelURL string, peerCount int, connected bool, width int) string {
	if tunnelURL == "" {
		return ""
	}

	var indicator string
	if connected {
		indicator = tunnelConnectedStyle.Render("●")
	} else {
		indicator = tunnelDisconnectedStyle.Render("●")
	}

	url := tunnelURL
	// Truncate long URLs on narrow terminals
	maxURLLen := width/2 - 20
	if maxURLLen < 20 {
		maxURLLen = 20
	}
	if len(url) > maxURLLen {
		url = url[:maxURLLen-3] + "..."
	}

	peers := fmt.Sprintf("%d peers", peerCount)
	if peerCount == 1 {
		peers = "1 peer"
	}

	return fmt.Sprintf("  %s %s  %s %s",
		indicator,
		tunnelURLStyle.Render(url),
		indicator,
		peerCountStyle.Render(peers),
	)
}
