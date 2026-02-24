package board

import (
	"context"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

// Service defines all task operations.
type Service interface {
	ListTasks(ctx context.Context) ([]db.Task, error)
	ListTasksByStatus(ctx context.Context, status db.TaskStatus) ([]db.Task, error)
	GetTask(ctx context.Context, id string) (*db.Task, error)
	CreateTask(ctx context.Context, title, description string) (*db.Task, error)
	UpdateTask(ctx context.Context, task *db.Task) error
	UpdateTaskFields(ctx context.Context, id string, fields db.TaskFieldUpdate) error
	MoveTask(ctx context.Context, id string, newStatus db.TaskStatus) error
	DeleteTask(ctx context.Context, id string) error
	ClaimTask(ctx context.Context, id, assignee string) error
	UnclaimTask(ctx context.Context, id string) error
	UpdateAgentActivity(ctx context.Context, id, activity string) error

	// Comments
	AddComment(ctx context.Context, taskID, author, body string) (*db.Comment, error)
	ListComments(ctx context.Context, taskID string) ([]db.Comment, error)

	// Dependencies
	AddDependency(ctx context.Context, taskID, dependsOn string) error
	RemoveDependency(ctx context.Context, taskID, dependsOn string) error
	ListDependencies(ctx context.Context, taskID string) ([]string, error)
	ListAllDependencies(ctx context.Context) (map[string][]string, error)

	// Suggestions
	CreateSuggestion(ctx context.Context, taskID string, sugType db.SuggestionType, author, title, message string) (*db.Suggestion, error)
	GetSuggestion(ctx context.Context, id string) (*db.Suggestion, error)
	ListPendingSuggestions(ctx context.Context) ([]db.Suggestion, error)
	ListSuggestions(ctx context.Context, status db.SuggestionStatus) ([]db.Suggestion, error)
	AcceptSuggestion(ctx context.Context, id string) error
	DismissSuggestion(ctx context.Context, id string) error
}
