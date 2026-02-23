package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
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
