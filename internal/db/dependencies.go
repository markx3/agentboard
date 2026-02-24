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

// HasCycle checks if adding a dependency (taskID depends on dependsOnID) would create a cycle.
// It loads the full dependency graph in a single query and traverses in-memory.
func (d *DB) HasCycle(ctx context.Context, taskID, dependsOnID string) (bool, error) {
	rows, err := d.conn.QueryContext(ctx, `SELECT task_id, depends_on FROM task_dependencies`)
	if err != nil {
		return false, fmt.Errorf("loading dependency graph: %w", err)
	}
	defer rows.Close()

	// Build adjacency: dependsOn -> []taskIDs (what depends on it)
	// We need to check: if taskID depends on dependsOnID, does dependsOnID
	// transitively depend on taskID? i.e., can we reach taskID from dependsOnID
	// following depends_on edges?
	graph := make(map[string][]string)
	for rows.Next() {
		var tid, dep string
		if err := rows.Scan(&tid, &dep); err != nil {
			return false, err
		}
		// tid depends on dep. So from dep's perspective, dep -> tid in the
		// "blocks" direction. We want to traverse: starting from dependsOnID,
		// follow the "depends_on" chain to see if we reach taskID.
		// tid depends on dep means: dep is upstream of tid.
		// We want: does dependsOnID depend (transitively) on taskID?
		// Build graph as: node -> what it depends on
		graph[tid] = append(graph[tid], dep)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	// DFS from dependsOnID through "depends_on" edges. If we reach taskID,
	// adding taskID -> dependsOnID would form a cycle.
	visited := make(map[string]bool)
	stack := []string{dependsOnID}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if node == taskID {
			return true, nil
		}
		if visited[node] {
			continue
		}
		visited[node] = true
		stack = append(stack, graph[node]...)
	}
	return false, nil
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
