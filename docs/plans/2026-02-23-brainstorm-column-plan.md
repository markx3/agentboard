# Brainstorm Column Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a Brainstorm column between Backlog and Planning, update agent prompts so Backlog nudges toward Brainstorm, and change claim behavior to target Brainstorm.

**Architecture:** New `StatusBrainstorm` status value threaded through the existing column system. SQLite migration (v3→v4) recreates tasks table to update CHECK constraint. All movement/rendering logic works automatically from `columnOrder`.

**Tech Stack:** Go, SQLite, Bubble Tea TUI, Cobra CLI

---

### Task 1: Add StatusBrainstorm to data model

**Files:**
- Modify: `internal/db/models.go:7-13` (status constants)
- Modify: `internal/db/models.go:15-21` (Valid method)

**Step 1: Write the failing test**

Add to `internal/db/tasks_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/db/ -run TestMoveToBrainstorm -v`
Expected: compile error — `StatusBrainstorm` not defined.

**Step 3: Add StatusBrainstorm constant and update Valid()**

In `internal/db/models.go`, change the constants block to:

```go
const (
	StatusBacklog    TaskStatus = "backlog"
	StatusBrainstorm TaskStatus = "brainstorm"
	StatusPlanning   TaskStatus = "planning"
	StatusInProgress TaskStatus = "in_progress"
	StatusReview     TaskStatus = "review"
	StatusDone       TaskStatus = "done"
)
```

Update `Valid()`:

```go
func (s TaskStatus) Valid() bool {
	switch s {
	case StatusBacklog, StatusBrainstorm, StatusPlanning, StatusInProgress, StatusReview, StatusDone:
		return true
	}
	return false
}
```

**Step 4: Run test — still fails (CHECK constraint)**

Run: `go test ./internal/db/ -run TestMoveToBrainstorm -v`
Expected: FAIL — SQLite CHECK constraint rejects `'brainstorm'`.

**Step 5: Commit model changes**

```bash
git add internal/db/models.go internal/db/tasks_test.go
git commit -m "feat(db): add StatusBrainstorm constant and validation"
```

---

### Task 2: Schema migration v3→v4

**Files:**
- Modify: `internal/db/schema.go:3` (bump schemaVersion to 4)
- Modify: `internal/db/schema.go:10-11` (update CHECK in schemaSQL)
- Modify: `internal/db/schema.go` (add migrateV3toV4 constant)
- Modify: `internal/db/sqlite.go:104-121` (add v3→v4 migration block)

**Step 1: Update schema.go**

Change `schemaVersion` from 3 to 4:

```go
const schemaVersion = 4
```

Update the CHECK constraint in `schemaSQL` (line 10-11):

```go
    status TEXT NOT NULL DEFAULT 'backlog'
        CHECK(status IN ('backlog','brainstorm','planning','in_progress','review','done')),
```

Add migration constant after `migrateV2toV3`:

```go
const migrateV3toV4 = `
CREATE TABLE tasks_v4 (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL CHECK(length(title) > 0 AND length(title) <= 500),
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'backlog'
        CHECK(status IN ('backlog','brainstorm','planning','in_progress','review','done')),
    assignee TEXT DEFAULT '',
    branch_name TEXT DEFAULT '',
    pr_url TEXT DEFAULT '',
    pr_number INTEGER DEFAULT 0,
    agent_name TEXT DEFAULT '',
    agent_status TEXT DEFAULT 'idle'
        CHECK(agent_status IN ('idle','active','completed','error')),
    agent_started_at TEXT DEFAULT '',
    agent_spawned_status TEXT DEFAULT '',
    reset_requested INTEGER DEFAULT 0,
    skip_permissions INTEGER DEFAULT 0,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

INSERT INTO tasks_v4 SELECT * FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_v4 RENAME TO tasks;

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_assignee ON tasks(assignee);
CREATE UNIQUE INDEX idx_tasks_status_position ON tasks(status, position);
`
```

**Step 2: Add migration block in sqlite.go**

Add after the `currentVersion < 3` block (after line 121):

```go
	if currentVersion < 4 {
		tx, txErr := d.conn.BeginTx(ctx, nil)
		if txErr != nil {
			return fmt.Errorf("beginning v4 migration transaction: %w", txErr)
		}
		defer tx.Rollback()

		if _, txErr = tx.ExecContext(ctx, migrateV3toV4); txErr != nil {
			return fmt.Errorf("applying v4 migration: %w", txErr)
		}
		if _, txErr = tx.ExecContext(ctx,
			"INSERT OR REPLACE INTO schema_version (version) VALUES (4)"); txErr != nil {
			return fmt.Errorf("updating schema version to 4: %w", txErr)
		}
		if txErr = tx.Commit(); txErr != nil {
			return fmt.Errorf("committing v4 migration: %w", txErr)
		}
	}
```

**Step 3: Run the brainstorm test — should pass now**

Run: `go test ./internal/db/ -run TestMoveToBrainstorm -v`
Expected: PASS

**Step 4: Run all DB tests to ensure migration doesn't break anything**

Run: `go test ./internal/db/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/db/schema.go internal/db/sqlite.go
git commit -m "feat(db): add v3→v4 migration for brainstorm status"
```

---

### Task 3: Update TUI column order

**Files:**
- Modify: `internal/tui/board.go:11-17` (columnOrder)
- Modify: `internal/tui/board.go:19-25` (columnTitles)

**Step 1: Update columnOrder**

```go
var columnOrder = []db.TaskStatus{
	db.StatusBacklog,
	db.StatusBrainstorm,
	db.StatusPlanning,
	db.StatusInProgress,
	db.StatusReview,
	db.StatusDone,
}
```

**Step 2: Update columnTitles**

```go
var columnTitles = map[db.TaskStatus]string{
	db.StatusBacklog:    "Backlog",
	db.StatusBrainstorm: "Brainstorm",
	db.StatusPlanning:   "Planning",
	db.StatusInProgress: "In Progress",
	db.StatusReview:     "Review",
	db.StatusDone:       "Done",
}
```

**Step 3: Run all tests to verify nothing breaks**

Run: `go test ./... 2>&1 | tail -20`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/tui/board.go
git commit -m "feat(tui): add Brainstorm column between Backlog and Planning"
```

---

### Task 4: Update Claude agent prompts

**Files:**
- Modify: `internal/agent/claude.go:50-70` (buildClaudeSystemPrompt switch)
- Modify: `internal/agent/claude.go:76-88` (buildClaudeInitialPrompt switch)

**Step 1: Write the failing test**

Update `TestClaudeRunnerStagePrompts` in `internal/agent/spawn_test.go`. Replace the test table (lines 129-138):

```go
	tests := []struct {
		status       db.TaskStatus
		wantSysStage string
		wantInitial  string
	}{
		{db.StatusBacklog, "Backlog", "Move it to brainstorm"},
		{db.StatusBrainstorm, "Brainstorm", "/workflows:brainstorm"},
		{db.StatusPlanning, "Planning", "/workflows:plan"},
		{db.StatusInProgress, "In Progress", "/workflows:work"},
		{db.StatusDone, "Done", "Verify the pull request"},
	}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/agent/ -run TestClaudeRunnerStagePrompts -v`
Expected: FAIL — Backlog test expects "Move it to brainstorm" but gets "/workflows:brainstorm". Brainstorm test fails (no case match).

**Step 3: Update buildClaudeSystemPrompt**

Replace the Backlog case and add Brainstorm case in `internal/agent/claude.go`:

```go
	case db.StatusBacklog:
		b.WriteString("STAGE: Backlog — Unplanned\n")
		b.WriteString("Move to brainstorm to begin work:\n")
		fmt.Fprintf(&b, "  agentboard task move %s brainstorm\n", shortID)
	case db.StatusBrainstorm:
		b.WriteString("STAGE: Brainstorm — Exploring Ideas\n")
		b.WriteString("When brainstorming is complete, move to planning:\n")
		fmt.Fprintf(&b, "  agentboard task move %s planning\n", shortID)
```

**Step 4: Update buildClaudeInitialPrompt**

Replace the Backlog case and add Brainstorm case:

```go
	case db.StatusBacklog:
		return "This task is in backlog. Move it to brainstorm to begin work."
	case db.StatusBrainstorm:
		return "Run /workflows:brainstorm to explore ideas for this task."
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/agent/ -run TestClaudeRunnerStagePrompts -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/agent/claude.go internal/agent/spawn_test.go
git commit -m "feat(agent): update Claude prompts for Backlog and Brainstorm stages"
```

---

### Task 5: Update Cursor agent prompts

**Files:**
- Modify: `internal/agent/cursor.go:51-75` (buildCursorPrompt switch)

**Step 1: Update buildCursorPrompt**

Replace the Backlog case and add Brainstorm case:

```go
	case db.StatusBacklog:
		b.WriteString("STAGE: Backlog — Unplanned\n")
		b.WriteString("This task is in the backlog. Move it to brainstorm to begin work.\n")
		b.WriteString("To move:\n")
		fmt.Fprintf(&b, "  agentboard task move %s brainstorm\n", shortID)
	case db.StatusBrainstorm:
		b.WriteString("STAGE: Brainstorm — Exploring Ideas\n")
		b.WriteString("Explore ideas and brainstorm approaches for this task.\n")
		b.WriteString("When brainstorming is complete, move to planning:\n")
		fmt.Fprintf(&b, "  agentboard task move %s planning\n", shortID)
```

**Step 2: Run Cursor test to verify**

Run: `go test ./internal/agent/ -run TestCursorRunnerBuildCommand -v`
Expected: PASS (existing test uses StatusPlanning, not affected)

**Step 3: Commit**

```bash
git add internal/agent/cursor.go
git commit -m "feat(agent): update Cursor prompts for Backlog and Brainstorm stages"
```

---

### Task 6: Update claim behavior

**Files:**
- Modify: `internal/board/local.go:56-58` (ClaimTask target)

**Step 1: Write the failing test**

Update `TestClaimTask` in `internal/board/local_test.go` (line 45-47):

```go
	if got.Status != db.StatusBrainstorm {
		t.Errorf("got status %q, want %q", got.Status, db.StatusBrainstorm)
	}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/board/ -run TestClaimTask -v`
Expected: FAIL — got "planning", want "brainstorm"

**Step 3: Update ClaimTask in local.go**

Change lines 56-58:

```go
	task.Status = db.StatusBrainstorm
	// Reposition at end of brainstorm column
	pos, err := s.db.NextPosition(ctx, db.StatusBrainstorm)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/board/ -run TestClaimTask -v`
Expected: PASS

**Step 5: Update TestMoveTaskService to include brainstorm**

In `internal/board/local_test.go`, update the statuses slice (line 89-94):

```go
	statuses := []db.TaskStatus{
		db.StatusBrainstorm,
		db.StatusPlanning,
		db.StatusInProgress,
		db.StatusReview,
		db.StatusDone,
	}
```

**Step 6: Run all board tests**

Run: `go test ./internal/board/ -v`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/board/local.go internal/board/local_test.go
git commit -m "feat(board): change claim target from planning to brainstorm"
```

---

### Task 7: Update CLI status command and move error message

**Files:**
- Modify: `internal/cli/status_cmd.go:51-58` (boardSummary columns)
- Modify: `internal/cli/status_cmd.go:66-71` (display output)
- Modify: `internal/cli/task_cmd.go:182` (error message)

**Step 1: Update boardSummary in status_cmd.go**

Add brainstorm to the Columns map (after backlog):

```go
	summary := boardSummary{
		Columns: map[string]int{
			string(db.StatusBacklog):    counts[string(db.StatusBacklog)],
			string(db.StatusBrainstorm): counts[string(db.StatusBrainstorm)],
			string(db.StatusPlanning):   counts[string(db.StatusPlanning)],
			string(db.StatusInProgress): counts[string(db.StatusInProgress)],
			string(db.StatusReview):     counts[string(db.StatusReview)],
			string(db.StatusDone):       counts[string(db.StatusDone)],
		},
		Total: len(tasks),
	}
```

Add brainstorm to the text output (after Backlog line):

```go
	fmt.Printf("Backlog:     %d\n", summary.Columns[string(db.StatusBacklog)])
	fmt.Printf("Brainstorm:  %d\n", summary.Columns[string(db.StatusBrainstorm)])
	fmt.Printf("Planning:    %d\n", summary.Columns[string(db.StatusPlanning)])
```

**Step 2: Update error message in task_cmd.go**

Change line 182:

```go
		return fmt.Errorf("invalid status: %s (use: backlog, brainstorm, planning, in_progress, review, done)", args[1])
```

**Step 3: Build to verify compilation**

Run: `go build ./...`
Expected: SUCCESS

**Step 4: Run full test suite**

Run: `go test ./... 2>&1 | tail -20`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/cli/status_cmd.go internal/cli/task_cmd.go
git commit -m "feat(cli): add brainstorm to status summary and move validation"
```

---

### Task 8: Final verification

**Step 1: Run full test suite**

Run: `go test ./... -v 2>&1 | tail -40`
Expected: All PASS

**Step 2: Build the binary**

Run: `go build -o agentboard .`
Expected: SUCCESS

**Step 3: Verify the binary works**

Run: `./agentboard --help`
Expected: Shows help output without errors

**Step 4: Clean up build artifact**

Run: `rm -f agentboard`
