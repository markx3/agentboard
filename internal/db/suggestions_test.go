package db_test

import (
	"context"
	"testing"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

func TestSuggestionCRUD(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, _ := database.CreateTask(ctx, "Suggest Target", "")

	sug, err := database.CreateSuggestion(ctx, task.ID, db.SuggestionEnrichment, "claude", "Better title", "Rename to X")
	if err != nil {
		t.Fatalf("creating suggestion: %v", err)
	}
	if sug.ID == "" {
		t.Fatal("expected non-empty suggestion ID")
	}
	if sug.Status != db.SuggestionPending {
		t.Errorf("status: got %q, want %q", sug.Status, db.SuggestionPending)
	}
	if sug.Type != db.SuggestionEnrichment {
		t.Errorf("type: got %q, want %q", sug.Type, db.SuggestionEnrichment)
	}

	// Get by ID
	got, err := database.GetSuggestion(ctx, sug.ID)
	if err != nil {
		t.Fatalf("getting suggestion: %v", err)
	}
	if got.Title != "Better title" {
		t.Errorf("title: got %q, want %q", got.Title, "Better title")
	}
	if got.Message != "Rename to X" {
		t.Errorf("message: got %q, want %q", got.Message, "Rename to X")
	}
}

func TestListPendingSuggestions(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, _ := database.CreateTask(ctx, "Sug Task", "")

	database.CreateSuggestion(ctx, task.ID, db.SuggestionEnrichment, "claude", "S1", "msg1")
	database.CreateSuggestion(ctx, task.ID, db.SuggestionProposal, "claude", "S2", "msg2")

	pending, err := database.ListPendingSuggestions(ctx)
	if err != nil {
		t.Fatalf("listing pending: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("got %d pending, want 2", len(pending))
	}
}

func TestUpdateSuggestionStatus(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, _ := database.CreateTask(ctx, "Sug Task", "")
	sug, _ := database.CreateSuggestion(ctx, task.ID, db.SuggestionHint, "claude", "Hint", "A hint")

	if err := database.UpdateSuggestionStatus(ctx, sug.ID, db.SuggestionAccepted); err != nil {
		t.Fatalf("updating status: %v", err)
	}

	got, _ := database.GetSuggestion(ctx, sug.ID)
	if got.Status != db.SuggestionAccepted {
		t.Errorf("status: got %q, want %q", got.Status, db.SuggestionAccepted)
	}

	// Should no longer appear in pending list
	pending, _ := database.ListPendingSuggestions(ctx)
	if len(pending) != 0 {
		t.Errorf("got %d pending after accept, want 0", len(pending))
	}
}

func TestListSuggestionsByTask(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	t1, _ := database.CreateTask(ctx, "Task 1", "")
	t2, _ := database.CreateTask(ctx, "Task 2", "")

	database.CreateSuggestion(ctx, t1.ID, db.SuggestionEnrichment, "claude", "S1", "for t1")
	database.CreateSuggestion(ctx, t2.ID, db.SuggestionProposal, "claude", "S2", "for t2")
	database.CreateSuggestion(ctx, t1.ID, db.SuggestionHint, "claude", "S3", "also for t1")

	sugs, err := database.ListSuggestionsByTask(ctx, t1.ID)
	if err != nil {
		t.Fatalf("listing by task: %v", err)
	}
	if len(sugs) != 2 {
		t.Errorf("got %d suggestions for t1, want 2", len(sugs))
	}

	sugs2, _ := database.ListSuggestionsByTask(ctx, t2.ID)
	if len(sugs2) != 1 {
		t.Errorf("got %d suggestions for t2, want 1", len(sugs2))
	}
}

func TestListSuggestionsByStatus(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, _ := database.CreateTask(ctx, "Filter Task", "")
	s1, _ := database.CreateSuggestion(ctx, task.ID, db.SuggestionEnrichment, "claude", "S1", "msg")
	database.CreateSuggestion(ctx, task.ID, db.SuggestionProposal, "claude", "S2", "msg")

	// Dismiss s1
	database.UpdateSuggestionStatus(ctx, s1.ID, db.SuggestionDismissed)

	dismissed, err := database.ListSuggestions(ctx, db.SuggestionDismissed)
	if err != nil {
		t.Fatalf("listing dismissed: %v", err)
	}
	if len(dismissed) != 1 {
		t.Errorf("got %d dismissed, want 1", len(dismissed))
	}

	pending, _ := database.ListSuggestions(ctx, db.SuggestionPending)
	if len(pending) != 1 {
		t.Errorf("got %d pending, want 1", len(pending))
	}
}

func TestSuggestionsCascadeOnTaskDelete(t *testing.T) {
	database := setupTestDB(t)
	ctx := context.Background()

	task, _ := database.CreateTask(ctx, "Cascade Task", "")
	database.CreateSuggestion(ctx, task.ID, db.SuggestionEnrichment, "claude", "S1", "msg")

	if err := database.DeleteTask(ctx, task.ID); err != nil {
		t.Fatalf("deleting task: %v", err)
	}

	pending, _ := database.ListPendingSuggestions(ctx)
	if len(pending) != 0 {
		t.Errorf("after task delete: got %d pending, want 0", len(pending))
	}
}

func TestSuggestionTypeValid(t *testing.T) {
	tests := []struct {
		t    db.SuggestionType
		want bool
	}{
		{db.SuggestionEnrichment, true},
		{db.SuggestionProposal, true},
		{db.SuggestionHint, true},
		{"invalid", false},
	}
	for _, tt := range tests {
		if got := tt.t.Valid(); got != tt.want {
			t.Errorf("SuggestionType(%q).Valid() = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestSuggestionStatusValid(t *testing.T) {
	tests := []struct {
		s    db.SuggestionStatus
		want bool
	}{
		{db.SuggestionPending, true},
		{db.SuggestionAccepted, true},
		{db.SuggestionDismissed, true},
		{"invalid", false},
	}
	for _, tt := range tests {
		if got := tt.s.Valid(); got != tt.want {
			t.Errorf("SuggestionStatus(%q).Valid() = %v, want %v", tt.s, got, tt.want)
		}
	}
}
