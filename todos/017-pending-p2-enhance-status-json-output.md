---
status: pending
priority: p2
issue_id: "017"
tags: [code-review, agent-native, cli]
dependencies: []
---

# Enhance `status --json` with Agent and Enrichment Info

## Problem Statement

The enrichment prompt instructs agents to run `agentboard status --json` to "see running agents" (plan section 2.3, step 2). But `status --json` only returns column counts: `{"columns": {"backlog": 2}, "total": 3}`. No agent info, no enrichment status, no pending suggestions.

## Findings

- **Source**: agent-native-reviewer
- **Location**: `internal/cli/status_cmd.go:27-30`
- **Evidence**: Current output is just column count aggregates
- **Impact**: Enrichment agents can't discover what other agents are working on via `status`

## Proposed Solutions

### Option A: Enhance status output (Recommended, Effort: Small, Risk: Low)
Add `agents`, `enrichments`, and `pending_suggestions` to JSON output:
```json
{
  "columns": {...},
  "total": 3,
  "agents": [{"task_id": "...", "agent_name": "claude", "status": "active"}],
  "enrichments": [{"task_id": "...", "status": "enriching"}],
  "pending_suggestions": 2
}
```

## Acceptance Criteria

- [ ] `status --json` includes `agents` array with active agent details
- [ ] `status --json` includes `enrichments` array with enrichment status
- [ ] `status --json` includes `pending_suggestions` count
