---
title: "feat: Add ngrok tunnel integration to expose server"
type: feat
date: 2026-02-23
brainstorm: docs/brainstorms/2026-02-23-ngrok-tunnel-integration-brainstorm.md
---

# feat: Add ngrok Tunnel Integration to Expose Server

## Overview

Add a `--tunnel` flag to `agentboard serve` that creates an ngrok tunnel using the Go SDK, exposing the WebSocket server over the internet with an HTTPS URL. The tunnel URL is displayed in the TUI status bar (for peers connecting with `--connect`) alongside a connection status indicator and connected peer count. All connections remain authenticated via GitHub tokens.

## Problem Statement

Agentboard's WebSocket server binds to `127.0.0.1` by default, making it inaccessible to remote coworkers. Sharing a board requires either:
- VPN/network configuration (IT overhead)
- Manual port forwarding (technical barrier)
- Cloud deployment (operational overhead)

Users need a single-command solution to make their board accessible to remote teammates.

## Proposed Solution

Use the ngrok Go SDK (`golang.ngrok.com/ngrok/v2`) as a drop-in `net.Listener` replacement in the server's `Start()` method. When `--tunnel` is passed to `agentboard serve`:

1. `ngrok.Listen(ctx)` creates a tunnel and returns a `net.Listener`
2. The existing `http.Server.Serve(listener)` works unchanged
3. The tunnel's public HTTPS URL is logged to stdout and propagated to connected TUI clients
4. The TUI status bar shows: tunnel URL, status indicator, connected peer count
5. The WebSocket connector supports `wss://` for remote `--connect` usage

## Technical Approach

### Architecture

```
┌─────────────────────────────────────────────────┐
│  agentboard serve --tunnel                      │
│                                                 │
│  ┌──────────┐    ┌──────────┐    ┌───────────┐  │
│  │  SQLite   │───▶│  Service  │───▶│    Hub    │  │
│  │  board.db │    │ (local)  │    │ (clients) │  │
│  └──────────┘    └──────────┘    └─────┬─────┘  │
│                                        │        │
│                              ┌─────────▼──────┐ │
│                              │  HTTP Server   │ │
│                              │  mux: /ws      │ │
│                              └─────────┬──────┘ │
│                                        │        │
│                              ┌─────────▼──────┐ │
│                              │ ngrok Listener │ │
│                              │ (net.Listener) │ │
│                              └─────────┬──────┘ │
│                                        │        │
└────────────────────────────────────────┼────────┘
                                         │
                              ┌──────────▼─────────┐
                              │  ngrok Cloud Edge   │
                              │  https://abc.ngrok  │
                              │    -free.app        │
                              └──────────┬──────────┘
                                         │ wss://
                              ┌──────────▼─────────┐
                              │  Remote Peer TUI    │
                              │  --connect <url>    │
                              └─────────────────────┘
```

### Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Entry point | `serve` command only | TUI root has no server; adding one is a separate feature |
| Listener strategy | ngrok replaces `net.Listen` | Minimal code change, idiomatic SDK usage |
| Local access | Tunnel-only when `--tunnel` | Simplest; local peers use `--connect <tunnel-url>` |
| `--tunnel` + `--connect` | Mutually exclusive | Clean separation: server vs. client |
| Origin check | Allow all when tunneling | GitHub auth is the real gate; avoids domain pattern maintenance |
| `--connect` format | Accept full URL or bare host | Parse scheme if present; infer for bare hosts |
| Authtoken | `NGROK_AUTHTOKEN` env var | Standard ngrok convention |
| Tunnel URL persistence | Display-only | Not written to `server.json`; shown in stdout and TUI |

### Implementation Phases

#### Phase 1: Foundation — ngrok SDK + Server Listener Injection

**Goal:** Server can accept an external `net.Listener` and the `serve` command can create an ngrok tunnel.

**Tasks:**

- [x] Add `golang.ngrok.com/ngrok/v2` dependency to `go.mod`
- [x] Create `internal/tunnel/tunnel.go` — thin wrapper around ngrok SDK

```go
// internal/tunnel/tunnel.go
package tunnel

func Listen(ctx context.Context) (net.Listener, error)
func URLFromListener(ln net.Listener) string
```

- [x] Modify `Server.Start()` in `internal/server/server.go` to accept an optional external listener

```go
// Option A: Add StartWithListener method
func (s *Server) StartWithListener(ctx context.Context, ln net.Listener) error

// Option B: Add SetListener before Start
func (s *Server) SetListener(ln net.Listener)
```

- [x] Add `--tunnel` flag to `serve` command in `internal/cli/serve.go`
- [x] Update `runServe()` to create ngrok listener when `--tunnel` is set
- [x] Validate `NGROK_AUTHTOKEN` presence before attempting tunnel — fail fast with helpful error
- [x] Print tunnel URL to stdout after tunnel is established
- [x] Skip `peersync.WriteServerInfo()` when in tunnel mode (no local address to write)

**Success criteria:**
- `agentboard serve --tunnel` starts and prints an ngrok HTTPS URL
- WebSocket connections accepted via the tunnel URL
- Graceful shutdown tears down the tunnel

**Files modified:**
- `go.mod` / `go.sum` — new dependency
- `internal/tunnel/tunnel.go` — new file
- `internal/server/server.go` — listener injection
- `internal/cli/serve.go` — `--tunnel` flag and ngrok startup

#### Phase 2: WebSocket Compatibility — Origin Check + wss:// Connector

**Goal:** Remote peers can connect to the tunnel via `--connect <ngrok-url>`.

**Tasks:**

- [x] Update `CheckOrigin` in `internal/server/server.go` to allow all origins when tunnel mode is active

```go
// Make upgrader configurable instead of package-level var
// Pass a tunnelActive bool to control origin checking
```

- [x] Update `Connector.Connect()` in `internal/peersync/connector.go` to support `wss://`

```go
// Parse the address:
// - "https://abc.ngrok-free.app" → "wss://abc.ngrok-free.app/ws"
// - "abc.ngrok-free.app" → "wss://abc.ngrok-free.app/ws" (detect known domains)
// - "127.0.0.1:8080" → "ws://127.0.0.1:8080/ws" (backward compatible)
```

- [x] Wire up the `--connect` flag in TUI root command (`internal/cli/root.go`)
  - When `--connect` is provided, TUI connects as a peer client instead of local-only mode
  - Validate mutual exclusivity: error if both `--tunnel` and `--connect` are passed
  - Note: `--tunnel` is on `serve` only, `--connect` is a persistent flag — so they won't conflict on the same command, but document this clearly

**Success criteria:**
- `agentboard --connect https://abc.ngrok-free.app` connects via `wss://`
- `agentboard --connect 127.0.0.1:8080` still works via `ws://`
- Origin check does not block ngrok WebSocket upgrades

**Files modified:**
- `internal/server/server.go` — configurable origin check
- `internal/peersync/connector.go` — URL parsing and `wss://` support
- `internal/cli/root.go` — wire up `--connect`

#### Phase 3: Status Bar — Tunnel URL, Status Indicator, Peer Count

**Goal:** The TUI displays tunnel/server status in the status bar.

**Tasks:**

- [x] Add atomic client counter to `Hub` in `internal/server/hub.go`

```go
// Add to Hub struct:
clientCount atomic.Int32

// Increment on register, decrement on unregister
// Export: func (h *Hub) ClientCount() int
```

- [x] Add new TUI message types in `internal/tui/messages.go`

```go
type serverStatusMsg struct {
    tunnelURL  string // empty if no tunnel
    peerCount  int
    connected  bool
}
```

- [x] Add server status fields to `App` struct in `internal/tui/app.go`

```go
// Add to App struct:
tunnelURL  string
peerCount  int
serverActive bool
```

- [x] Create a Bubble Tea command that periodically polls hub client count (or subscribes to changes)
- [x] Update `statusBar()` in `internal/tui/board.go` (or `App.View()`) to render tunnel info

```
  https://abc123.ngrok-free.app  ● 2 peers  |  Task Title | backlog | Column 1/4
```

- [x] Add styles for tunnel status in `internal/tui/styles.go`
  - Green dot `●` for connected tunnel
  - Red dot `●` for disconnected/no tunnel
  - Truncate long URLs on narrow terminals

**Success criteria:**
- Status bar shows tunnel URL when tunnel is active
- Peer count updates in real time as peers join/leave
- Status bar gracefully handles narrow terminals (URL truncation)
- When no tunnel is active, status bar shows only existing info (no empty tunnel section)

**Files modified:**
- `internal/server/hub.go` — atomic client counter
- `internal/tui/messages.go` — new message types
- `internal/tui/app.go` — server status state + update handlers
- `internal/tui/board.go` — status bar rendering
- `internal/tui/styles.go` — tunnel status styles

#### Phase 4: Polish & Error Handling

**Goal:** Production-quality UX for all error paths and edge cases.

**Tasks:**

- [x] Handle missing `NGROK_AUTHTOKEN` with clear error message:
  `"--tunnel requires NGROK_AUTHTOKEN environment variable. Get yours at https://dashboard.ngrok.com/get-started/your-authtoken"`
- [x] Handle ngrok SDK errors (invalid token, quota exceeded, network) with user-friendly messages
- [x] Handle tunnel disconnect mid-session: status bar shows red indicator, log warning
- [x] Ensure graceful shutdown: context cancellation closes ngrok listener and all connections
- [x] Sanitize ngrok SDK error messages to avoid leaking authtoken
- [x] Add `--tunnel` to help text and command documentation
- [x] When `--bind` or `--port` flags are used alongside `--tunnel`, print a warning that they are ignored

**Success criteria:**
- Every error path produces a helpful, actionable message
- No authtoken leakage in logs or error output
- Clean shutdown with no goroutine leaks

**Files modified:**
- `internal/tunnel/tunnel.go` — error wrapping
- `internal/cli/serve.go` — flag validation and warnings

## Alternative Approaches Considered

### Separate Tunnel Package (Rejected)

Start the server on localhost first, then create a separate ngrok tunnel forwarding to it. Would allow simultaneous local + remote access but adds complexity (two-step startup, lifecycle coordination). Rejected in favor of the simpler listener replacement.

### Shell out to ngrok CLI (Rejected)

Exec the `ngrok` binary as a subprocess. Simpler initially but harder to manage lifecycle, extract URL, handle errors. Requires users to install ngrok separately. Rejected in favor of the Go SDK.

### `--tunnel` on TUI root command (Deferred)

The TUI root command has no embedded server today. Adding `--tunnel` there requires starting a server alongside the TUI — a larger architectural change. Deferred to a follow-up. Users can use `agentboard serve --tunnel` in one terminal and `agentboard --connect <url>` in another.

## Acceptance Criteria

### Functional Requirements

- [ ] `agentboard serve --tunnel` starts an ngrok tunnel and prints the HTTPS URL to stdout
- [ ] Remote peers can connect via `agentboard --connect <ngrok-url>` (full URL or bare host)
- [ ] WebSocket connections work over `wss://` through the ngrok tunnel
- [ ] GitHub token authentication is enforced for all tunnel connections
- [ ] TUI status bar displays: tunnel URL, green/red status indicator, connected peer count
- [ ] Peer count updates in real time as peers join/leave
- [ ] `--tunnel` without `NGROK_AUTHTOKEN` fails with a clear, actionable error message
- [ ] `--tunnel` and `--connect` are mutually exclusive (error if both passed)
- [ ] Graceful shutdown tears down the tunnel and disconnects all peers cleanly
- [ ] Existing local-only mode (`agentboard serve` without `--tunnel`) is unaffected

### Non-Functional Requirements

- [ ] Tunnel creation completes within 10 seconds (ngrok SDK typical)
- [ ] No goroutine leaks on shutdown
- [ ] No authtoken leakage in logs or error messages
- [ ] Status bar renders correctly on terminals 80+ columns wide
- [ ] Tunnel URL is truncated gracefully on narrow terminals

### Quality Gates

- [ ] All existing tests pass
- [ ] New unit tests for: tunnel listener wrapper, URL parsing in connector, origin check logic, hub client counter
- [ ] Manual test: end-to-end tunnel connection between two machines
- [ ] `go vet` and `go build` pass cleanly

## Success Metrics

- A user can expose their board with a single command: `agentboard serve --tunnel`
- A remote coworker can connect with: `agentboard --connect <shared-url>`
- The shared URL is visible in the TUI for easy copy-paste
- Zero configuration beyond setting `NGROK_AUTHTOKEN`

## Dependencies & Prerequisites

| Dependency | Type | Status |
|------------|------|--------|
| `golang.ngrok.com/ngrok/v2` | Go module | To be added |
| ngrok account + authtoken | User requirement | Users must sign up at ngrok.com |
| `--connect` flag wired up in TUI | Code prerequisite | Currently declared but unused; must be wired as part of Phase 2 |
| GitHub token auth | Existing | Already implemented, no changes needed |

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| ngrok SDK adds significant binary size | Medium | Low | Acceptable trade-off; can be build-tagged out later if needed |
| ngrok free tier has rate limits/bandwidth caps | Medium | Medium | Document limits in help text; works fine for dev collaboration |
| ngrok SDK API changes in future | Low | Medium | Pin dependency version; SDK is stable |
| Tunnel drops mid-session | Medium | Medium | Show red indicator in TUI; users restart to re-establish |
| Users forget to set NGROK_AUTHTOKEN | High | Low | Clear error message with dashboard link |
| Origin check bypass reduces security | Low | Low | GitHub auth is the real security gate; origin check is defense-in-depth for browsers |

## Future Considerations

- **`--tunnel` on TUI root command:** Requires embedding a server in the TUI. Natural next step after this feature ships.
- **Custom ngrok domains:** Add `--tunnel-domain` flag for paid ngrok users with reserved domains.
- **Dual listener (local + tunnel):** Allow both local and remote access simultaneously.
- **Tunnel URL in `agentboard status`:** Show tunnel URL in the CLI status command output.
- **Auto-copy tunnel URL to clipboard:** On macOS/Linux, pipe to `pbcopy`/`xclip`.
- **QR code for tunnel URL:** Display in terminal for quick mobile sharing.

## Documentation Plan

- [ ] Update `--help` text for `serve` command with `--tunnel` flag description
- [ ] Add "Sharing with Teammates" section to README
- [ ] Document `NGROK_AUTHTOKEN` setup steps

## References & Research

### Internal References

- Brainstorm document: `docs/brainstorms/2026-02-23-ngrok-tunnel-integration-brainstorm.md`
- Server startup: `internal/server/server.go:49-75`
- WebSocket upgrader: `internal/server/server.go:23-38`
- Serve command: `internal/cli/serve.go:35-85`
- Root command: `internal/cli/root.go:17-54`
- Hub client tracking: `internal/server/hub.go:17-24`
- Connector: `internal/peersync/connector.go:32-56`
- TUI status bar: `internal/tui/board.go:147-155`
- TUI app view: `internal/tui/app.go:514-545`
- TUI messages: `internal/tui/messages.go`

### External References

- ngrok Go SDK: `golang.ngrok.com/ngrok/v2`
- ngrok SDK docs: https://ngrok.com/docs/agent-sdks/
- ngrok authtoken setup: https://dashboard.ngrok.com/get-started/your-authtoken
