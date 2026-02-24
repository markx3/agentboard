---
date: 2026-02-23
topic: ngrok-tunnel-integration
---

# ngrok Tunnel Integration for Agentboard

## What We're Building

An ngrok tunnel integration that lets agentboard users expose their server to the internet with a single `--tunnel` flag, making it easy to share a live board URL with coworkers. The TUI status bar will show the tunnel URL, connection status indicator, and a count of connected peers.

The integration uses the ngrok Go SDK (`golang.ngrok.com/ngrok`) to programmatically create tunnels — no external `ngrok` CLI binary required. Users configure their ngrok authtoken via the `NGROK_AUTHTOKEN` environment variable.

## Why This Approach

**Approach chosen: ngrok as Listener Replacement**

We considered two approaches:

1. **Listener Replacement (chosen):** The ngrok Go SDK provides a `net.Listener`. When `--tunnel` is set, the server uses `ngrok.Listen()` instead of `net.Listen("tcp", addr)`. The rest of the server (WebSocket handling, hub, auth) is completely unaware of the tunnel. This is the simplest, most idiomatic integration with minimal code changes.

2. **Separate Tunnel Package:** Start the server on localhost first, then create a separate tunnel forwarding to it. More code, two-step startup, but allows simultaneous local+remote access. Rejected in favor of simplicity.

## Key Decisions

- **Entry point:** `--tunnel` flag on `agentboard serve` only. TUI root has no embedded server; adding `--tunnel` there is deferred.
- **ngrok method:** Go SDK (`golang.ngrok.com/ngrok`). No external binary dependency. Programmatic control, clean lifecycle management.
- **Auth policy:** Keep existing GitHub token auth for all connections, including remote ones via ngrok. Secure by default.
- **Authtoken config:** `NGROK_AUTHTOKEN` environment variable only. Standard ngrok convention, simple.
- **Status display:** TUI status bar shows tunnel URL + connected count. Headless `serve` mode logs the tunnel URL to stdout (no TUI).
- **Discovery:** Tunnel URL is display-only (shown in status bar for copy-paste sharing). Not written to `server.json`. Local peers still discover via local address.
- **WebSocket origin check:** Must be updated to allow ngrok origins when tunneling is active (currently only allows localhost).
- **WebSocket protocol:** Connector needs to support `wss://` for ngrok URLs (currently hardcodes `ws://`).

## Scope

### In Scope
- `--tunnel` flag on root and serve commands
- ngrok Go SDK integration to create HTTPS tunnel
- TUI status bar: tunnel URL, status indicator (connected/disconnected), connected peer count
- Update WebSocket upgrader to allow ngrok origins
- Support `wss://` in WebSocket connector for `--connect` with ngrok URLs

### Out of Scope
- ngrok CLI binary support (SDK only)
- Auth bypass for tunneled connections
- Writing tunnel URL to server.json
- Custom ngrok domains or subdomains (can be added later)
- `agentboard status` command changes
- Headless serve TUI (just stdout logging)

## Open Questions

_None — all questions resolved during brainstorming._

## Next Steps

-> `/workflows:plan` for implementation details
