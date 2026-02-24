package db

import (
	"context"
	"fmt"
	"time"
)

func (d *DB) AddDependency(ctx context.Context, taskID, dependsOn string) error {
	_, err := d.conn.ExecContext(ctx,
		`INSERT INTO task_dependencies (task_id, depends_on, created_at) VALUES (?, ?, ?)`,
		taskID, dependsOn, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("adding dependency: %w", err)
	}
	return nil
}

func (d *DB) RemoveDependency(ctx context.Context, taskID, dependsOn string) error {
	_, err := d.conn.ExecContext(ctx,
		`DELETE FROM task_dependencies WHERE task_id=? AND depends_on=?`,
		taskID, dependsOn)
	if err != nil {
		return fmt.Errorf("removing dependency: %w", err)
	}
	return nil
}

// ListDependencies returns IDs of tasks that taskID depends on.
func (d *DB) ListDependencies(ctx context.Context, taskID string) ([]string, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT depends_on FROM task_dependencies WHERE task_id=?`, taskID)
	if err != nil {
		return nil, fmt.Errorf("listing dependencies: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning dependency: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListDependents returns IDs of tasks that depend on taskID.
func (d *DB) ListDependents(ctx context.Context, taskID string) ([]string, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT task_id FROM task_dependencies WHERE depends_on=?`, taskID)
	if err != nil {
		return nil, fmt.Errorf("listing dependents: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning dependent: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListAllDependencies returns a map of taskID -> list of dependency IDs for all tasks.
func (d *DB) ListAllDependencies(ctx context.Context) (map[string][]string, error) {
	rows, err := d.conn.QueryContext(ctx,
		`SELECT task_id, depends_on FROM task_dependencies`)
	if err != nil {
		return nil, fmt.Errorf("listing all dependencies: %w", err)
	}
	defer rows.Close()

	deps := make(map[string][]string)
	for rows.Next() {
		var taskID, dependsOn string
		if err := rows.Scan(&taskID, &dependsOn); err != nil {
			return nil, fmt.Errorf("scanning dependency: %w", err)
		}
		deps[taskID] = append(deps[taskID], dependsOn)
	}
	return deps, rows.Err()
}
