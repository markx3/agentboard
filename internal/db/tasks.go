package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(s scanner) (Task, error) {
	var t Task
	var createdAt, updatedAt string
	var resetRequested, skipPermissions int
	if err := s.Scan(
		&t.ID, &t.Title, &t.Description, &t.Status,
		&t.Assignee, &t.BranchName, &t.PRUrl, &t.PRNumber,
		&t.AgentName, &t.AgentStatus, &t.AgentStartedAt, &t.AgentSpawnedStatus,
		&resetRequested, &skipPermissions,
		&t.EnrichmentStatus, &t.EnrichmentAgentName,
		&t.AgentActivity, &t.Position,
		&createdAt, &updatedAt); err != nil {
		return Task{}, err
	}
	t.ResetRequested = resetRequested != 0
	t.SkipPermissions = skipPermissions != 0
	var err error
	t.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		log.Printf("warning: invalid created_at for task %s: %v", t.ID, err)
	}
	t.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		log.Printf("warning: invalid updated_at for task %s: %v", t.ID, err)
	}
	return t, nil
}

const taskColumns = `id, title, description, status, assignee, branch_name, pr_url, pr_number,
		        agent_name, agent_status, agent_started_at, agent_spawned_status,
		        reset_requested, skip_permissions,
		        enrichment_status, enrichment_agent_name,
		        agent_activity, position, created_at, updated_at`

func (d *DB) CreateTask(ctx context.Context, title, description string) (*Task, error) {
	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	id := uuid.New().String()

	// Get next position for backlog column (within transaction)
	var maxPos sql.NullInt64
	err = tx.QueryRowContext(ctx,
		"SELECT MAX(position) FROM tasks WHERE status = ?", StatusBacklog).Scan(&maxPos)
	if err != nil {
		return nil, fmt.Errorf("getting max position: %w", err)
	}
	pos := 0
	if maxPos.Valid {
		pos = int(maxPos.Int64) + 1
	}

	task := &Task{
		ID:          id,
		Title:       title,
		Description: description,
		Status:      StatusBacklog,
		AgentStatus: AgentIdle,
		Position:    pos,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO tasks (id, title, description, status, assignee, branch_name, pr_url, pr_number,
		 agent_name, agent_status, agent_started_at, agent_spawned_status, reset_requested,
		 skip_permissions, enrichment_status, enrichment_agent_name, agent_activity,
		 position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Title, task.Description, task.Status,
		task.Assignee, task.BranchName, task.PRUrl, task.PRNumber,
		task.AgentName, task.AgentStatus, task.AgentStartedAt, task.AgentSpawnedStatus,
		boolToInt(task.ResetRequested), boolToInt(task.SkipPermissions),
		task.EnrichmentStatus, task.EnrichmentAgentName, task.AgentActivity,
		task.Position,
		task.CreatedAt.Format(time.RFC3339), task.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("inserting task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing task: %w", err)
	}
	return task, nil
}

func (d *DB) GetTask(ctx context.Context, id string) (*Task, error) {
	row := d.conn.QueryRowContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if err != nil {
		return nil, fmt.Errorf("getting task: %w", err)
	}
	return &t, nil
}

func (d *DB) ListTasks(ctx context.Context) ([]Task, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks ORDER BY status, position`)
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]Task, 0, 64)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (d *DB) ListTasksByStatus(ctx context.Context, status TaskStatus) ([]Task, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE status = ? ORDER BY position`, status)
	if err != nil {
		return nil, fmt.Errorf("listing tasks by status: %w", err)
	}
	defer rows.Close()

	tasks := make([]Task, 0, 64)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (d *DB) UpdateTask(ctx context.Context, task *Task) error {
	task.UpdatedAt = time.Now().UTC()
	_, err := d.conn.ExecContext(ctx,
		`UPDATE tasks SET title=?, description=?, status=?, assignee=?, branch_name=?,
		 pr_url=?, pr_number=?, agent_name=?, agent_status=?, agent_started_at=?,
		 agent_spawned_status=?, reset_requested=?, skip_permissions=?,
		 enrichment_status=?, enrichment_agent_name=?,
		 agent_activity=?, position=?, updated_at=?
		 WHERE id=?`,
		task.Title, task.Description, task.Status, task.Assignee, task.BranchName,
		task.PRUrl, task.PRNumber, task.AgentName, task.AgentStatus, task.AgentStartedAt,
		task.AgentSpawnedStatus, boolToInt(task.ResetRequested), boolToInt(task.SkipPermissions),
		task.EnrichmentStatus, task.EnrichmentAgentName,
		task.AgentActivity, task.Position, task.UpdatedAt.Format(time.RFC3339), task.ID)
	if err != nil {
		return fmt.Errorf("updating task: %w", err)
	}
	return nil
}

// UpdateTaskFields updates only non-nil fields. Column names are hardcoded
// (not user-supplied) so there is no SQL injection risk.
func (d *DB) UpdateTaskFields(ctx context.Context, id string, fields TaskFieldUpdate) error {
	var setClauses []string
	var args []interface{}

	if fields.Title != nil {
		setClauses = append(setClauses, "title=?")
		args = append(args, *fields.Title)
	}
	if fields.Description != nil {
		setClauses = append(setClauses, "description=?")
		args = append(args, *fields.Description)
	}
	if fields.Status != nil {
		setClauses = append(setClauses, "status=?")
		args = append(args, *fields.Status)
	}
	if fields.Assignee != nil {
		setClauses = append(setClauses, "assignee=?")
		args = append(args, *fields.Assignee)
	}
	if fields.BranchName != nil {
		setClauses = append(setClauses, "branch_name=?")
		args = append(args, *fields.BranchName)
	}
	if fields.PRUrl != nil {
		setClauses = append(setClauses, "pr_url=?")
		args = append(args, *fields.PRUrl)
	}
	if fields.PRNumber != nil {
		setClauses = append(setClauses, "pr_number=?")
		args = append(args, *fields.PRNumber)
	}
	if fields.EnrichmentStatus != nil {
		setClauses = append(setClauses, "enrichment_status=?")
		args = append(args, *fields.EnrichmentStatus)
	}
	if fields.EnrichmentAgentName != nil {
		setClauses = append(setClauses, "enrichment_agent_name=?")
		args = append(args, *fields.EnrichmentAgentName)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at=?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, id)

	query := fmt.Sprintf("UPDATE tasks SET %s WHERE id=?", strings.Join(setClauses, ", "))
	_, err := d.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("updating task fields: %w", err)
	}
	return nil
}

// UpdateAgentActivity updates the agent_activity field for a task.
func (d *DB) UpdateAgentActivity(ctx context.Context, id, activity string) error {
	_, err := d.conn.ExecContext(ctx,
		`UPDATE tasks SET agent_activity=?, updated_at=? WHERE id=?`,
		activity, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("updating agent activity: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (d *DB) MoveTask(ctx context.Context, id string, newStatus TaskStatus) error {
	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Get next position in target column
	var maxPos sql.NullInt64
	err = tx.QueryRowContext(ctx,
		"SELECT MAX(position) FROM tasks WHERE status = ?", newStatus).Scan(&maxPos)
	if err != nil {
		return fmt.Errorf("getting max position: %w", err)
	}
	pos := 0
	if maxPos.Valid {
		pos = int(maxPos.Int64) + 1
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.ExecContext(ctx,
		"UPDATE tasks SET status=?, position=?, updated_at=? WHERE id=?",
		newStatus, pos, now, id)
	if err != nil {
		return fmt.Errorf("moving task: %w", err)
	}

	return tx.Commit()
}

func (d *DB) DeleteTask(ctx context.Context, id string) error {
	_, err := d.conn.ExecContext(ctx, "DELETE FROM tasks WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}
	return nil
}

// NextPosition returns the next available position for a given status column.
func (d *DB) NextPosition(ctx context.Context, status TaskStatus) (int, error) {
	var maxPos sql.NullInt64
	err := d.conn.QueryRowContext(ctx,
		"SELECT MAX(position) FROM tasks WHERE status = ?", status).Scan(&maxPos)
	if err != nil {
		return 0, err
	}
	if maxPos.Valid {
		return int(maxPos.Int64) + 1, nil
	}
	return 0, nil
}
