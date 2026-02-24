package db_test

import (
	"context"
	"testing"
)

func TestDependencyCRUD(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	t1, _ := database.CreateTask(ctx, "Task A", "")
	t2, _ := database.CreateTask(ctx, "Task B", "")
	t3, _ := database.CreateTask(ctx, "Task C", "")

	// Add dependencies: t1 depends on t2 and t3
	if err := database.AddDependency(ctx, t1.ID, t2.ID); err != nil {
		t.Fatalf("adding dependency t1->t2: %v", err)
	}
	if err := database.AddDependency(ctx, t1.ID, t3.ID); err != nil {
		t.Fatalf("adding dependency t1->t3: %v", err)
	}

	// ListDependencies: t1 depends on t2 and t3
	deps, err := database.ListDependencies(ctx, t1.ID)
	if err != nil {
		t.Fatalf("listing dependencies: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("got %d dependencies, want 2", len(deps))
	}

	// ListDependents: t2 is depended on by t1
	dependents, err := database.ListDependents(ctx, t2.ID)
	if err != nil {
		t.Fatalf("listing dependents: %v", err)
	}
	if len(dependents) != 1 {
		t.Errorf("got %d dependents, want 1", len(dependents))
	}
	if len(dependents) > 0 && dependents[0] != t1.ID {
		t.Errorf("dependent: got %q, want %q", dependents[0], t1.ID)
	}

	// RemoveDependency
	if err := database.RemoveDependency(ctx, t1.ID, t2.ID); err != nil {
		t.Fatalf("removing dependency: %v", err)
	}
	deps, _ = database.ListDependencies(ctx, t1.ID)
	if len(deps) != 1 {
		t.Errorf("after remove: got %d deps, want 1", len(deps))
	}
}

func TestListAllDependencies(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	t1, _ := database.CreateTask(ctx, "Task A", "")
	t2, _ := database.CreateTask(ctx, "Task B", "")
	t3, _ := database.CreateTask(ctx, "Task C", "")

	database.AddDependency(ctx, t1.ID, t2.ID)
	database.AddDependency(ctx, t3.ID, t2.ID)

	all, err := database.ListAllDependencies(ctx)
	if err != nil {
		t.Fatalf("listing all dependencies: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("got %d task entries, want 2", len(all))
	}
	if len(all[t1.ID]) != 1 || all[t1.ID][0] != t2.ID {
		t.Errorf("t1 deps: got %v, want [%s]", all[t1.ID], t2.ID)
	}
	if len(all[t3.ID]) != 1 || all[t3.ID][0] != t2.ID {
		t.Errorf("t3 deps: got %v, want [%s]", all[t3.ID], t2.ID)
	}
}

func TestSelfDependencyRejected(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, _ := database.CreateTask(ctx, "Self Dep", "")
	err := database.AddDependency(ctx, task.ID, task.ID)
	if err == nil {
		t.Error("expected error for self-dependency, got nil")
	}
}

func TestDuplicateDependencyRejected(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	t1, _ := database.CreateTask(ctx, "Task A", "")
	t2, _ := database.CreateTask(ctx, "Task B", "")

	if err := database.AddDependency(ctx, t1.ID, t2.ID); err != nil {
		t.Fatalf("first add: %v", err)
	}
	err := database.AddDependency(ctx, t1.ID, t2.ID)
	if err == nil {
		t.Error("expected error for duplicate dependency, got nil")
	}
}

func TestDependenciesCascadeOnDelete(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	t1, _ := database.CreateTask(ctx, "Task A", "")
	t2, _ := database.CreateTask(ctx, "Task B", "")
	database.AddDependency(ctx, t1.ID, t2.ID)

	// Delete t2 — should cascade and remove the dependency row
	if err := database.DeleteTask(ctx, t2.ID); err != nil {
		t.Fatalf("deleting task: %v", err)
	}

	deps, _ := database.ListDependencies(ctx, t1.ID)
	if len(deps) != 0 {
		t.Errorf("after cascade delete: got %d deps, want 0", len(deps))
	}
}
