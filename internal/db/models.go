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

type EnrichmentStatus string

const (
	EnrichmentNone      EnrichmentStatus = ""
	EnrichmentPending   EnrichmentStatus = "pending"
	EnrichmentEnriching EnrichmentStatus = "enriching"
	EnrichmentDone      EnrichmentStatus = "done"
	EnrichmentError     EnrichmentStatus = "error"
	EnrichmentSkipped   EnrichmentStatus = "skipped"
)

func (s EnrichmentStatus) Valid() bool {
	switch s {
	case EnrichmentNone, EnrichmentPending, EnrichmentEnriching, EnrichmentDone, EnrichmentError, EnrichmentSkipped:
		return true
	}
	return false
}

type Task struct {
	ID                  string           `json:"id"`
	Title               string           `json:"title"`
	Description         string           `json:"description"`
	Status              TaskStatus       `json:"status"`
	Assignee            string           `json:"assignee"`
	BranchName          string           `json:"branch_name"`
	PRUrl               string           `json:"pr_url"`
	PRNumber            int              `json:"pr_number"`
	AgentName           string           `json:"agent_name"`
	AgentStatus         AgentStatus      `json:"agent_status"`
	AgentStartedAt      string           `json:"agent_started_at"`
	AgentSpawnedStatus  string           `json:"agent_spawned_status"`
	ResetRequested      bool             `json:"reset_requested"`
	SkipPermissions     bool             `json:"skip_permissions"`
	EnrichmentStatus    EnrichmentStatus `json:"enrichment_status"`
	EnrichmentAgentName string           `json:"enrichment_agent_name"`
	AgentActivity       string           `json:"agent_activity"`
	Position            int              `json:"position"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
	// BlockedBy is populated at read time, not stored in the tasks table.
	BlockedBy []string `json:"blocked_by,omitempty"`
}

// TaskFieldUpdate holds optional field updates. Nil pointer = don't update.
type TaskFieldUpdate struct {
	Title               *string           `json:"title,omitempty"`
	Description         *string           `json:"description,omitempty"`
	Status              *TaskStatus       `json:"status,omitempty"`
	Assignee            *string           `json:"assignee,omitempty"`
	BranchName          *string           `json:"branch_name,omitempty"`
	PRUrl               *string           `json:"pr_url,omitempty"`
	PRNumber            *int              `json:"pr_number,omitempty"`
	EnrichmentStatus    *EnrichmentStatus `json:"enrichment_status,omitempty"`
	EnrichmentAgentName *string           `json:"enrichment_agent_name,omitempty"`
}

type Comment struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type SuggestionType string

const (
	SuggestionEnrichment SuggestionType = "enrichment"
	SuggestionProposal   SuggestionType = "proposal"
	SuggestionHint       SuggestionType = "hint"
)

func (s SuggestionType) Valid() bool {
	switch s {
	case SuggestionEnrichment, SuggestionProposal, SuggestionHint:
		return true
	}
	return false
}

type SuggestionStatus string

const (
	SuggestionPending   SuggestionStatus = "pending"
	SuggestionAccepted  SuggestionStatus = "accepted"
	SuggestionDismissed SuggestionStatus = "dismissed"
)

func (s SuggestionStatus) Valid() bool {
	switch s {
	case SuggestionPending, SuggestionAccepted, SuggestionDismissed:
		return true
	}
	return false
}

type Suggestion struct {
	ID        string           `json:"id"`
	TaskID    string           `json:"task_id"`
	Type      SuggestionType   `json:"type"`
	Author    string           `json:"author"`
	Title     string           `json:"title"`
	Message   string           `json:"message"`
	Status    SuggestionStatus `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
}
