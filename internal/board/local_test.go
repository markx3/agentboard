package board_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

func setupTestService(t *testing.T) board.Service {
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
	return board.NewLocalService(database)
}

func TestClaimTask(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	task, err := svc.CreateTask(ctx, "Claimable", "desc")
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}

	if err := svc.ClaimTask(ctx, task.ID, "alice"); err != nil {
		t.Fatalf("claiming task: %v", err)
	}

	got, _ := svc.GetTask(ctx, task.ID)
	if got.Assignee != "alice" {
		t.Errorf("got assignee %q, want %q", got.Assignee, "alice")
	}
	if got.Status != db.StatusPlanning {
		t.Errorf("got status %q, want %q", got.Status, db.StatusPlanning)
	}
}

func TestClaimAlreadyClaimed(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	task, _ := svc.CreateTask(ctx, "Claimed", "")
	svc.ClaimTask(ctx, task.ID, "alice")

	err := svc.ClaimTask(ctx, task.ID, "bob")
	if err == nil {
		t.Error("expected error claiming already-claimed task")
	}
}

func TestUnclaimTask(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	task, _ := svc.CreateTask(ctx, "Unclaim Me", "")
	svc.ClaimTask(ctx, task.ID, "alice")

	if err := svc.UnclaimTask(ctx, task.ID); err != nil {
		t.Fatalf("unclaiming: %v", err)
	}

	got, _ := svc.GetTask(ctx, task.ID)
	if got.Assignee != "" {
		t.Errorf("got assignee %q, want empty", got.Assignee)
	}
	if got.Status != db.StatusBacklog {
		t.Errorf("got status %q, want %q", got.Status, db.StatusBacklog)
	}
}

func TestMoveTaskService(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	task, _ := svc.CreateTask(ctx, "Move Through", "")

	statuses := []db.TaskStatus{
		db.StatusPlanning,
		db.StatusInProgress,
		db.StatusDone,
	}

	for _, s := range statuses {
		if err := svc.MoveTask(ctx, task.ID, s); err != nil {
			t.Fatalf("moving to %s: %v", s, err)
		}
		got, _ := svc.GetTask(ctx, task.ID)
		if got.Status != s {
			t.Errorf("after move: got %q, want %q", got.Status, s)
		}
	}
}
