---
status: complete
priority: p2
issue_id: "015"
tags: [code-review, agent-native]
dependencies: []
---

# Agent System Prompt Missing New CLI Commands

## Problem Statement

The system prompt in `claude.go` and `cursor.go` teaches agents about `task move` and `agent status`, but not `task update` (for setting branch, PR URL, etc.) or `task block`/`task unblock` (for dependency management). Agents have tools available but don't know they exist ("Capability Hiding" anti-pattern).

## Findings

- **Agent-Native Reviewer:** "This is especially important for `--branch` and `--pr-url` since agents that open PRs should record that metadata on the task."

**Location:** `internal/agent/claude.go:38-82`, `internal/agent/cursor.go:39-88`

## Proposed Solutions

### Solution A: Add TASK METADATA and DEPENDENCIES sections to prompts (Recommended)
Add documentation for `task update`, `task block`, and `task unblock` to both agent system prompts.

- **Effort:** Small (15 min)
- **Risk:** None

## Acceptance Criteria

- [ ] Claude system prompt documents `task update`, `task block`, `task unblock`
- [ ] Cursor system prompt documents the same
- [ ] Agents can discover and use these commands

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-02-23 | Created | Flagged by agent-native reviewer |
