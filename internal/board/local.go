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
	task.Status = db.StatusBrainstorm
	// Reposition at end of brainstorm column
	pos, err := s.db.NextPosition(ctx, db.StatusBrainstorm)
	if err != nil {
		return err
	}
	task.Position = pos
	return s.db.UpdateTask(ctx, task)
}

func (s *LocalService) UpdateAgentActivity(ctx context.Context, id, activity string) error {
	return s.db.UpdateAgentActivity(ctx, id, activity)
}

func (s *LocalService) AddDependency(ctx context.Context, taskID, blockerID string) error {
	// Cycle check
	hasCycle, err := s.db.HasCycle(ctx, taskID, blockerID)
	if err != nil {
		return fmt.Errorf("checking cycle: %w", err)
	}
	if hasCycle {
		return fmt.Errorf("adding this dependency would create a cycle")
	}
	return s.db.AddDependency(ctx, taskID, blockerID)
}

func (s *LocalService) RemoveDependency(ctx context.Context, taskID, blockerID string) error {
	return s.db.RemoveDependency(ctx, taskID, blockerID)
}

func (s *LocalService) GetBlockers(ctx context.Context, taskID string) ([]string, error) {
	return s.db.GetBlockers(ctx, taskID)
}

func (s *LocalService) GetAllDependencies(ctx context.Context) (map[string][]string, error) {
	return s.db.GetAllDependencies(ctx)
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
