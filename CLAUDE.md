# Agentboard

Real-time collaborative Kanban board for AI coding agents. Go CLI/TUI application.

## Build & Test

```bash
go build -o agentboard ./cmd/agentboard    # build binary
go test ./...                                # run all tests
go test -race -v ./...                       # tests with race detection
go vet ./...                                 # static analysis
```

## Architecture

Peer-leader model: first instance becomes WebSocket server, others connect as peers.

```
cmd/agentboard/     → entry point, delegates to internal/cli
internal/
  cli/              → Cobra commands (root, serve, init, task, status, agent)
  tui/              → Bubble Tea TUI (App model, board, forms, overlays)
  server/           → WebSocket hub, client pump goroutines, protocol, sequencer
  board/            → Service interface + LocalService implementation
  db/               → SQLite: models, schema migrations, queries
  agent/            → Agent runners (Claude, Cursor), spawn/kill via tmux
  auth/             → GitHub token verification
  tmux/             → tmux session/window management
  peersync/         → Peer discovery via server.json files
```

## Key Patterns

**Error handling**: Wrap with `fmt.Errorf("context: %w", err)`. No sentinel errors or custom types.

**Concurrency**: Channels for message passing (hub pattern with select loop). Mutexes only for local state. `sync/atomic` for counters. Context-based cancellation.

**Bubble Tea**: Unexported `xxxMsg` message types. Single `App` model dispatches to nested components. Commands returned from `Update()`, long work in `func() tea.Msg` closures.

**WebSocket protocol**: JSON envelope with `Type`, `Seq`, `Sender`, `Payload` (json.RawMessage). Server assigns sequence numbers. Hub broadcasts via buffered channels.

**SQLite**: Single connection (`SetMaxOpenConns(1)`). WAL mode. Version-based migrations applied on `db.Open()`. Parameterized queries with `?` placeholders.

**Testing**: `t.Helper()` + `t.TempDir()` + `t.Cleanup()` for test isolation. Each test gets a fresh DB. Assert with `t.Errorf("got %v, want %v", ...)`.

## Conventions

- Files: `lowercase_underscore.go`
- Receivers: short 1-2 letter names (`c *Client`, `h *Hub`)
- Constructors: `New{Type}()`
- Messages: unexported `xxxMsg` structs (Bubble Tea), `Msg` prefix constants (WebSocket)
- Tests: `*_test.go` in `xxx_test` package, no mocks, integration tests with real DB

## Task Columns

Valid statuses: `backlog`, `brainstorm`, `planning`, `in_progress`, `review`, `done`
