package tui

import "github.com/marcosfelipeeipper/agentboard/internal/db"

// TUI-internal messages

type tasksLoadedMsg struct {
	tasks []db.Task
}

type taskCreatedMsg struct {
	task *db.Task
}

type taskMovedMsg struct {
	taskID    string
	newStatus db.TaskStatus
	hadAgent  bool
}

type taskDeletedMsg struct {
	taskID string
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string { return e.err.Error() }

type notifyMsg struct {
	text string
}

// Agent lifecycle messages

type agentSpawnedMsg struct {
	taskID string
}

type agentKilledMsg struct {
	taskID string
}

type agentPollMsg struct {
	alive map[string]bool
}

type agentPollTickMsg struct{}

type agentViewDoneMsg struct{}

