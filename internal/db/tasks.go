package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (d *DB) CreateTask(ctx context.Context, title, description string) (*Task, error) {
	now := time.Now().UTC()
	id := uuid.New().String()

	// Get next position for backlog column
	var maxPos sql.NullInt64
	err := d.conn.QueryRowContext(ctx,
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

	_, err = d.conn.ExecContext(ctx,
		`INSERT INTO tasks (id, title, description, status, assignee, branch_name, pr_url, pr_number, agent_name, agent_status, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Title, task.Description, task.Status,
		task.Assignee, task.BranchName, task.PRUrl, task.PRNumber,
		task.AgentName, task.AgentStatus, task.Position,
		task.CreatedAt.Format(time.RFC3339), task.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("inserting task: %w", err)
	}

	return task, nil
}

func (d *DB) GetTask(ctx context.Context, id string) (*Task, error) {
	task := &Task{}
	var createdAt, updatedAt string
	err := d.conn.QueryRowContext(ctx,
		`SELECT id, title, description, status, assignee, branch_name, pr_url, pr_number,
		        agent_name, agent_status, position, created_at, updated_at
		 FROM tasks WHERE id = ?`, id).Scan(
		&task.ID, &task.Title, &task.Description, &task.Status,
		&task.Assignee, &task.BranchName, &task.PRUrl, &task.PRNumber,
		&task.AgentName, &task.AgentStatus, &task.Position,
		&createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting task: %w", err)
	}
	task.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	task.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return task, nil
}

func (d *DB) ListTasks(ctx context.Context) ([]Task, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT id, title, description, status, assignee, branch_name, pr_url, pr_number,
		        agent_name, agent_status, position, created_at, updated_at
		 FROM tasks ORDER BY status, position`)
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var createdAt, updatedAt string
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Status,
			&t.Assignee, &t.BranchName, &t.PRUrl, &t.PRNumber,
			&t.AgentName, &t.AgentStatus, &t.Position,
			&createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (d *DB) ListTasksByStatus(ctx context.Context, status TaskStatus) ([]Task, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT id, title, description, status, assignee, branch_name, pr_url, pr_number,
		        agent_name, agent_status, position, created_at, updated_at
		 FROM tasks WHERE status = ? ORDER BY position`, status)
	if err != nil {
		return nil, fmt.Errorf("listing tasks by status: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		var createdAt, updatedAt string
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Status,
			&t.Assignee, &t.BranchName, &t.PRUrl, &t.PRNumber,
			&t.AgentName, &t.AgentStatus, &t.Position,
			&createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (d *DB) UpdateTask(ctx context.Context, task *Task) error {
	task.UpdatedAt = time.Now().UTC()
	_, err := d.conn.ExecContext(ctx,
		`UPDATE tasks SET title=?, description=?, status=?, assignee=?, branch_name=?,
		 pr_url=?, pr_number=?, agent_name=?, agent_status=?, position=?, updated_at=?
		 WHERE id=?`,
		task.Title, task.Description, task.Status, task.Assignee, task.BranchName,
		task.PRUrl, task.PRNumber, task.AgentName, task.AgentStatus, task.Position,
		task.UpdatedAt.Format(time.RFC3339), task.ID)
	if err != nil {
		return fmt.Errorf("updating task: %w", err)
	}
	return nil
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
