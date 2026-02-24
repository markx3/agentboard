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

func (s *LocalService) UpdateTaskFields(ctx context.Context, id string, fields db.TaskFieldUpdate) error {
	return s.db.UpdateTaskFields(ctx, id, fields)
}

func (s *LocalService) AddComment(ctx context.Context, taskID, author, body string) (*db.Comment, error) {
	return s.db.AddComment(ctx, taskID, author, body)
}

func (s *LocalService) ListComments(ctx context.Context, taskID string) ([]db.Comment, error) {
	return s.db.ListComments(ctx, taskID)
}

func (s *LocalService) AddDependency(ctx context.Context, taskID, dependsOn string) error {
	return s.db.AddDependency(ctx, taskID, dependsOn)
}

func (s *LocalService) RemoveDependency(ctx context.Context, taskID, dependsOn string) error {
	return s.db.RemoveDependency(ctx, taskID, dependsOn)
}

func (s *LocalService) ListDependencies(ctx context.Context, taskID string) ([]string, error) {
	return s.db.ListDependencies(ctx, taskID)
}

func (s *LocalService) ListAllDependencies(ctx context.Context) (map[string][]string, error) {
	return s.db.ListAllDependencies(ctx)
}

func (s *LocalService) CreateSuggestion(ctx context.Context, taskID string, sugType db.SuggestionType, author, title, message string) (*db.Suggestion, error) {
	return s.db.CreateSuggestion(ctx, taskID, sugType, author, title, message)
}

func (s *LocalService) GetSuggestion(ctx context.Context, id string) (*db.Suggestion, error) {
	return s.db.GetSuggestion(ctx, id)
}

func (s *LocalService) ListPendingSuggestions(ctx context.Context) ([]db.Suggestion, error) {
	return s.db.ListPendingSuggestions(ctx)
}

func (s *LocalService) ListSuggestions(ctx context.Context, status db.SuggestionStatus) ([]db.Suggestion, error) {
	return s.db.ListSuggestions(ctx, status)
}

func (s *LocalService) AcceptSuggestion(ctx context.Context, id string) error {
	sug, err := s.db.GetSuggestion(ctx, id)
	if err != nil {
		return fmt.Errorf("getting suggestion: %w", err)
	}
	if sug.Status != db.SuggestionPending {
		return fmt.Errorf("suggestion is not pending (status: %s)", sug.Status)
	}

	// For proposals, create a new task and update suggestion atomically
	if sug.Type == db.SuggestionProposal {
		task, err := s.db.CreateTask(ctx, sug.Title, sug.Message)
		if err != nil {
			return fmt.Errorf("creating task from proposal: %w", err)
		}
		// Set enrichment pending so the new task gets auto-enriched
		pending := db.EnrichmentPending
		if err := s.db.UpdateTaskFields(ctx, task.ID, db.TaskFieldUpdate{
			EnrichmentStatus: &pending,
		}); err != nil {
			return fmt.Errorf("setting enrichment on proposed task: %w", err)
		}
	}

	return s.db.UpdateSuggestionStatus(ctx, id, db.SuggestionAccepted)
}

func (s *LocalService) DismissSuggestion(ctx context.Context, id string) error {
	return s.db.UpdateSuggestionStatus(ctx, id, db.SuggestionDismissed)
}
