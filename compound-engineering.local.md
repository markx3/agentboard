---
review_agents:
  - compound-engineering:review:architecture-strategist
  - compound-engineering:review:security-sentinel
  - compound-engineering:review:performance-oracle
  - compound-engineering:review:code-simplicity-reviewer
---

## Review Context

This is a Go TUI application (Agentboard) using:
- Bubble Tea + Lipgloss for terminal UI
- gorilla/websocket for real-time collaboration
- modernc.org/sqlite for persistence (pure Go, no CGO)
- Cobra for CLI framework
- tmux for agent session management

Focus on: Go idioms, concurrency safety, WebSocket security, SQLite best practices, Bubble Tea patterns.
