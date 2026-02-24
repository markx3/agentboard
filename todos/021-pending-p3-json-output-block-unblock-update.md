---
status: pending
priority: p3
issue_id: "021"
tags: [code-review, agent-native]
dependencies: []
---

# Add --json Output to task block, unblock, and update Commands

## Problem Statement

`task block`, `task unblock`, and `task update` only output human-readable text. Agents need `--json` for programmatic verification.

**Location:** `internal/cli/task_cmd.go`

## Acceptance Criteria

- [ ] `--json` flag on `task block`, `task unblock`, and `task update`
- [ ] JSON output includes relevant task data
