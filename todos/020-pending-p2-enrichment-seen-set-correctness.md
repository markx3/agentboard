---
status: pending
priority: p2
issue_id: "020"
tags: [code-review, correctness, enrichment]
dependencies: []
---

# Fix Enrichment Seen-Set for Re-Enrichment Support

## Problem Statement

The `seen map[string]bool` in the enrichment tracker grows monotonically. Once a task is added (regardless of enrichment success or failure), it is never re-enriched automatically. If enrichment fails, the task stays in `seen` and no automatic retry occurs.

## Findings

- **Source**: performance-oracle
- **Evidence**: Plan section 2.5 defines `seen map[string]bool` with no pruning or reset mechanism
- **Risk**: Failed enrichments are permanent; user must manually trigger re-enrichment

## Proposed Solutions

### Option A: Use status-aware map (Recommended, Effort: Small, Risk: Low)
Change to `map[string]db.EnrichmentStatus`. Track what enrichment state was last seen. If the task's status resets to `pending`, the tracker detects the change and re-enriches. Periodically prune entries for deleted tasks.

## Acceptance Criteria

- [ ] Re-enrichment triggers when `enrichment_status` is reset to `pending`
- [ ] Deleted tasks are pruned from the seen set
- [ ] Failed enrichments can be retried by setting `enrichment_status=pending`
