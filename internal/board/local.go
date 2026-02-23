package board

import (
	"context"
	"fmt"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

// LocalService implements Service backed by a local SQLite database.
type LocalService struct {
	db *db.DB
}

func NewLocalService(database *db.DB) *LocalService {
	return &LocalService{db: database}
}

func (s *LocalService) ListTasks(ctx context.Context) ([]db.Task, error) {
	return s.db.ListTasks(ctx)
}

func (s *LocalService) ListTasksByStatus(ctx context.Context, status db.TaskStatus) ([]db.Task, error) {
	return s.db.ListTasksByStatus(ctx, status)
}

func (s *LocalService) GetTask(ctx context.Context, id string) (*db.Task, error) {
	return s.db.GetTask(ctx, id)
}

func (s *LocalService) CreateTask(ctx context.Context, title, description string) (*db.Task, error) {
	return s.db.CreateTask(ctx, title, description)
}

func (s *LocalService) UpdateTask(ctx context.Context, task *db.Task) error {
	return s.db.UpdateTask(ctx, task)
}

func (s *LocalService) MoveTask(ctx context.Context, id string, newStatus db.TaskStatus) error {
	return s.db.MoveTask(ctx, id, newStatus)
}

func (s *LocalService) DeleteTask(ctx context.Context, id string) error {
	return s.db.DeleteTask(ctx, id)
}

func (s *LocalService) ClaimTask(ctx context.Context, id, assignee string) error {
	task, err := s.db.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task.Assignee != "" {
		return fmt.Errorf("task already claimed by %s", task.Assignee)
	}
	task.Assignee = assignee
	task.Status = db.StatusPlanning
	// Reposition at end of planning column
	pos, err := s.db.NextPosition(ctx, db.StatusPlanning)
	if err != nil {
		return err
	}
	task.Position = pos
	return s.db.UpdateTask(ctx, task)
}

func (s *LocalService) UnclaimTask(ctx context.Context, id string) error {
	task, err := s.db.GetTask(ctx, id)
	if err != nil {
		return err
	}
	task.Assignee = ""
	task.AgentName = ""
	task.AgentStatus = db.AgentIdle
	task.BranchName = ""
	task.Status = db.StatusBacklog
	pos, err := s.db.NextPosition(ctx, db.StatusBacklog)
	if err != nil {
		return err
	}
	task.Position = pos
	return s.db.UpdateTask(ctx, task)
}
