package db

import "time"

type TaskStatus string

const (
	StatusBacklog    TaskStatus = "backlog"
	StatusBrainstorm TaskStatus = "brainstorm"
	StatusPlanning   TaskStatus = "planning"
	StatusInProgress TaskStatus = "in_progress"
	StatusReview     TaskStatus = "review"
	StatusDone       TaskStatus = "done"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case StatusBacklog, StatusBrainstorm, StatusPlanning, StatusInProgress, StatusReview, StatusDone:
		return true
	}
	return false
}

type AgentStatus string

const (
	AgentIdle      AgentStatus = "idle"
	AgentActive    AgentStatus = "active"
	AgentCompleted AgentStatus = "completed"
	AgentError     AgentStatus = "error"
)

type Task struct {
	ID                 string      `json:"id"`
	Title              string      `json:"title"`
	Description        string      `json:"description"`
	Status             TaskStatus  `json:"status"`
	Assignee           string      `json:"assignee"`
	BranchName         string      `json:"branch_name"`
	PRUrl              string      `json:"pr_url"`
	PRNumber           int         `json:"pr_number"`
	AgentName          string      `json:"agent_name"`
	AgentStatus        AgentStatus `json:"agent_status"`
	AgentStartedAt     string      `json:"agent_started_at"`
	AgentSpawnedStatus string      `json:"agent_spawned_status"`
	ResetRequested     bool        `json:"reset_requested"`
	SkipPermissions    bool        `json:"skip_permissions"`
	AgentActivity      string      `json:"agent_activity"`
	Position           int         `json:"position"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

type Comment struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
