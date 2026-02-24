---
status: pending
priority: p1
issue_id: "016"
tags: [code-review, data-integrity, correctness]
dependencies: []
---

# Use UpdateTaskFields in Enrichment Reconciliation (Not Full UpdateTask)

## Problem Statement

The plan's enrichment reconciliation (section 2.7) calls `a.service.UpdateTask()` after modifying `EnrichmentStatus`. This is a full-row replacement of all 15+ columns, which can overwrite concurrent edits by enrichment agents or humans. The plan explicitly warns about this race in `SpawnEnrichment` (section 2.2) and recommends `UpdateTaskFields`, but then violates its own guidance in the reconciliation path.

## Findings

- **Source**: performance-oracle + simplicity-reviewer
- **Location**: Plan section 2.7 code sample
- **Evidence**: `a.service.UpdateTask(context.Background(), freshTask)` after setting only `EnrichmentStatus` and `EnrichmentAgentName`
- **Risk**: If an enrichment agent writes a description at the same moment the TUI sets enrichment_status=done via full UpdateTask, the description is overwritten

## Proposed Solutions

### Option A: Use UpdateTaskFields (Recommended, Effort: Small, Risk: Low)
```go
done := db.EnrichmentDone
empty := ""
svc.UpdateTaskFields(ctx, task.ID, db.TaskFieldUpdate{
    EnrichmentStatus: &done,
    EnrichmentAgent:  &empty,
})
```
- **Pros**: Touches only 2 fields, no race with concurrent edits
- **Cons**: None

## Recommended Action

Option A. Straightforward fix. Also applies to the existing `reconcileAgents()` (line 308-322 of app.go) as a follow-up.

## Acceptance Criteria

- [ ] Enrichment reconciliation uses `UpdateTaskFields`, not `UpdateTask`
- [ ] Only `enrichment_status` and `enrichment_agent_name` are written
- [ ] Existing `reconcileAgents()` flagged for future migration to partial updates
