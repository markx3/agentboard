package db_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
		os.Remove(dbPath)
	})
	return database
}

func TestCreateAndGetTask(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, err := database.CreateTask(ctx, "Test Task", "A description")
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	if task.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if task.Title != "Test Task" {
		t.Errorf("got title %q, want %q", task.Title, "Test Task")
	}
	if task.Status != db.StatusBacklog {
		t.Errorf("got status %q, want %q", task.Status, db.StatusBacklog)
	}

	got, err := database.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("getting task: %v", err)
	}
	if got.Title != "Test Task" {
		t.Errorf("got title %q, want %q", got.Title, "Test Task")
	}
}

func TestListTasks(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	database.CreateTask(ctx, "Task 1", "")
	database.CreateTask(ctx, "Task 2", "")
	database.CreateTask(ctx, "Task 3", "")

	tasks, err := database.ListTasks(ctx)
	if err != nil {
		t.Fatalf("listing tasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("got %d tasks, want 3", len(tasks))
	}
}

func TestMoveTask(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, err := database.CreateTask(ctx, "Move Me", "")
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	if err := database.MoveTask(ctx, task.ID, db.StatusPlanning); err != nil {
		t.Fatalf("moving task: %v", err)
	}

	got, err := database.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("getting task: %v", err)
	}
	if got.Status != db.StatusPlanning {
		t.Errorf("got status %q, want %q", got.Status, db.StatusPlanning)
	}
}

func TestDeleteTask(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, _ := database.CreateTask(ctx, "Delete Me", "")
	if err := database.DeleteTask(ctx, task.ID); err != nil {
		t.Fatalf("deleting task: %v", err)
	}

	tasks, _ := database.ListTasks(ctx)
	if len(tasks) != 0 {
		t.Errorf("got %d tasks, want 0", len(tasks))
	}
}

func TestListTasksByStatus(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	database.CreateTask(ctx, "Backlog 1", "")
	database.CreateTask(ctx, "Backlog 2", "")
	t1, _ := database.CreateTask(ctx, "To Move", "")
	database.MoveTask(ctx, t1.ID, db.StatusPlanning)

	backlog, err := database.ListTasksByStatus(ctx, db.StatusBacklog)
	if err != nil {
		t.Fatalf("listing by status: %v", err)
	}
	if len(backlog) != 2 {
		t.Errorf("got %d backlog tasks, want 2", len(backlog))
	}

	planning, _ := database.ListTasksByStatus(ctx, db.StatusPlanning)
	if len(planning) != 1 {
		t.Errorf("got %d planning tasks, want 1", len(planning))
	}
}

func TestTaskPositioning(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	t1, _ := database.CreateTask(ctx, "First", "")
	t2, _ := database.CreateTask(ctx, "Second", "")
	t3, _ := database.CreateTask(ctx, "Third", "")

	if t1.Position != 0 {
		t.Errorf("first task position: got %d, want 0", t1.Position)
	}
	if t2.Position != 1 {
		t.Errorf("second task position: got %d, want 1", t2.Position)
	}
	if t3.Position != 2 {
		t.Errorf("third task position: got %d, want 2", t3.Position)
	}
}

func TestCheckConstraints(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	// Empty title should fail
	_, err := database.CreateTask(ctx, "", "")
	if err == nil {
		t.Error("expected error for empty title, got nil")
	}
}

func TestAgentLifecycleFields(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, err := database.CreateTask(ctx, "Agent Task", "testing agent fields")
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	// Verify defaults
	if task.AgentStartedAt != "" {
		t.Errorf("expected empty agent_started_at, got %q", task.AgentStartedAt)
	}
	if task.AgentSpawnedStatus != "" {
		t.Errorf("expected empty agent_spawned_status, got %q", task.AgentSpawnedStatus)
	}
	if task.ResetRequested {
		t.Error("expected reset_requested=false")
	}

	// Set agent lifecycle fields
	task.AgentStatus = db.AgentActive
	task.AgentStartedAt = "2026-02-23T12:00:00Z"
	task.AgentSpawnedStatus = "backlog"
	task.ResetRequested = false
	if err := database.UpdateTask(ctx, task); err != nil {
		t.Fatalf("updating task: %v", err)
	}

	// Round-trip: verify fields persisted
	got, err := database.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("getting task: %v", err)
	}
	if got.AgentStatus != db.AgentActive {
		t.Errorf("agent_status: got %q, want %q", got.AgentStatus, db.AgentActive)
	}
	if got.AgentStartedAt != "2026-02-23T12:00:00Z" {
		t.Errorf("agent_started_at: got %q, want %q", got.AgentStartedAt, "2026-02-23T12:00:00Z")
	}
	if got.AgentSpawnedStatus != "backlog" {
		t.Errorf("agent_spawned_status: got %q, want %q", got.AgentSpawnedStatus, "backlog")
	}
	if got.ResetRequested {
		t.Error("expected reset_requested=false after update")
	}

	// Set completed status and reset flag
	task.AgentStatus = db.AgentCompleted
	task.ResetRequested = true
	if err := database.UpdateTask(ctx, task); err != nil {
		t.Fatalf("updating task: %v", err)
	}

	got, err = database.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("getting task: %v", err)
	}
	if got.AgentStatus != db.AgentCompleted {
		t.Errorf("agent_status: got %q, want %q", got.AgentStatus, db.AgentCompleted)
	}
	if !got.ResetRequested {
		t.Error("expected reset_requested=true")
	}
}

func TestMoveToBrainstorm(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, err := database.CreateTask(ctx, "Brainstorm Me", "")
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	if err := database.MoveTask(ctx, task.ID, db.StatusBrainstorm); err != nil {
		t.Fatalf("moving task to brainstorm: %v", err)
	}

	got, err := database.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("getting task: %v", err)
	}
	if got.Status != db.StatusBrainstorm {
		t.Errorf("got status %q, want %q", got.Status, db.StatusBrainstorm)
	}
}

func TestSchemaV2Migration(t *testing.T) {
	// Create a v1 database manually, then open it with the migrating code
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "v1.db")

	// Create v1 schema manually
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("opening raw db: %v", err)
	}

	v1Schema := `
	CREATE TABLE schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	INSERT INTO schema_version (version) VALUES (1);

	CREATE TABLE tasks (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL CHECK(length(title) > 0 AND length(title) <= 500),
		description TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'backlog'
			CHECK(status IN ('backlog','planning','in_progress','review','done')),
		assignee TEXT DEFAULT '',
		branch_name TEXT DEFAULT '',
		pr_url TEXT DEFAULT '',
		pr_number INTEGER DEFAULT 0,
		agent_name TEXT DEFAULT '',
		agent_status TEXT DEFAULT 'idle'
			CHECK(agent_status IN ('idle','active','error')),
		position INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE INDEX idx_tasks_status ON tasks(status);
	CREATE INDEX idx_tasks_assignee ON tasks(assignee);
	CREATE UNIQUE INDEX idx_tasks_status_position ON tasks(status, position);

	INSERT INTO tasks (id, title, description, status, assignee, branch_name, pr_url, pr_number,
		agent_name, agent_status, position, created_at, updated_at)
	VALUES ('test-id-1', 'V1 Task', 'old desc', 'backlog', '', '', '', 0,
		'', 'idle', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`

	if _, err := conn.Exec(v1Schema); err != nil {
		t.Fatalf("creating v1 schema: %v", err)
	}
	conn.Close()

	// Now open with the migrating code — should auto-migrate to v2
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("opening database for migration: %v", err)
	}
	defer database.Close()

	// Verify the migrated task still exists with new default fields
	ctx := context.Background()
	task, err := database.GetTask(ctx, "test-id-1")
	if err != nil {
		t.Fatalf("getting migrated task: %v", err)
	}

	if task.Title != "V1 Task" {
		t.Errorf("title: got %q, want %q", task.Title, "V1 Task")
	}
	if task.AgentStartedAt != "" {
		t.Errorf("agent_started_at should be empty after migration, got %q", task.AgentStartedAt)
	}
	if task.AgentSpawnedStatus != "" {
		t.Errorf("agent_spawned_status should be empty after migration, got %q", task.AgentSpawnedStatus)
	}
	if task.ResetRequested {
		t.Error("reset_requested should be false after migration")
	}

	// Verify completed status works on migrated DB
	task.AgentStatus = db.AgentCompleted
	if err := database.UpdateTask(ctx, task); err != nil {
		t.Fatalf("setting completed status on migrated task: %v", err)
	}
	got, _ := database.GetTask(ctx, task.ID)
	if got.AgentStatus != db.AgentCompleted {
		t.Errorf("completed status not persisted on migrated DB: got %q", got.AgentStatus)
	}
}
