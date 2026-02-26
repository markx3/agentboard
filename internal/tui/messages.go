package tui

import "github.com/markx3/agentboard/internal/db"

type tasksLoadedMsg struct {
	tasks []db.Task
	deps  map[string][]string
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

type agentSpawnedMsg struct {
	taskID string
}

type agentKilledMsg struct {
	taskID string
}

type agentViewDoneMsg struct{}

type agentTickMsg struct{}

type serverStatusMsg struct {
	tunnelURL string
	peerCount int
	connected bool
}

type spawnAfterConfirmMsg struct {
	task db.Task
}

// taskSavedMsg is emitted after a task is saved in edit mode.
type taskSavedMsg struct {
	task db.Task
}

// taskSaveRequestedMsg is emitted by the task detail overlay when the user confirms an edit.
type taskSaveRequestedMsg struct {
	task db.Task
}

type suggestionsLoadedMsg struct {
	items []db.Suggestion
}

type suggestionAcceptedMsg struct {
	suggestionID string
}

type suggestionDismissedMsg struct {
	suggestionID string
}

type clearNotificationMsg struct{}
