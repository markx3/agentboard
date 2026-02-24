package board

import (
	"context"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

// Service defines all task operations. The TUI talks only to this interface.
// LocalService backs it with SQLite; NetworkService (Phase 3) will proxy via WebSocket.
type Service interface {
	ListTasks(ctx context.Context) ([]db.Task, error)
	ListTasksByStatus(ctx context.Context, status db.TaskStatus) ([]db.Task, error)
	GetTask(ctx context.Context, id string) (*db.Task, error)
	CreateTask(ctx context.Context, title, description string) (*db.Task, error)
	UpdateTask(ctx context.Context, task *db.Task) error
	MoveTask(ctx context.Context, id string, newStatus db.TaskStatus) error
	DeleteTask(ctx context.Context, id string) error
	ClaimTask(ctx context.Context, id, assignee string) error
	UnclaimTask(ctx context.Context, id string) error
	UpdateAgentActivity(ctx context.Context, id, activity string) error
}
