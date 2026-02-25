# Brainstorm: README.md for Agentboard

**Date:** 2026-02-23
**Status:** Accepted
**Task:** Add README.md (agentboard task 43c1f2d2)

## What We're Building

A comprehensive, standalone README.md for agentboard — a terminal-based collaborative Kanban board for managing agentic coding tasks. The README serves both end users (teams adopting agentboard) and contributors (developers building on it).

## Why This Approach

**Getting-Started-First structure** — prioritizes action over marketing. After a brief overview with logo and screenshot, the README jumps straight to prerequisites, installation, and quick start. Features, architecture, and reference material follow for those who want depth.

This mirrors the pattern of tools like `tmux`, `htop`, and `jq` — utilitarian tools where users want to get running fast.

## Key Decisions

1. **Audience:** Both end users and contributors, balanced equally
2. **Depth:** Comprehensive standalone (300-500 lines), not just a pointer to docs/
3. **Visuals:** ASCII art banner/logo, placeholder for terminal screenshots/GIFs, mermaid architecture diagram
4. **Structure:** Getting-Started-First (overview → install → quick start → features → reference)
5. **Sections included:**
   - ASCII logo/banner
   - One-liner description
   - Screenshot/GIF placeholder
   - Prerequisites (Go 1.25+, tmux 3.0+, gh CLI 2.0+, AI CLI tool)
   - Installation (go install, build from source, Homebrew planned)
   - Quick Start (init → run → collaborate)
   - Features list
   - Supported AI agents table
   - CLI Reference (all commands with flags)
   - Configuration (global + project config)
   - Architecture overview (mermaid diagram)
   - Roadmap / current status
   - Contributing guidelines
   - License
6. **No comparison section** — not needed at this stage
7. **Dependencies to document:** Bubble Tea, Bubbles, Lipgloss, Cobra, gorilla/websocket, modernc.org/sqlite, google/uuid

## Section Details

### Logo / Banner
Simple ASCII art. Could use a box-drawing style or figlet-style text. Keep it compact (3-5 lines).

### One-Liner
"Real-time collaborative Kanban board for AI coding agents. Terminal-native. Agent-agnostic."

### Prerequisites
- Go 1.25.0+
- tmux 3.0+ (for agent session management)
- gh CLI 2.0+ (for GitHub auth)
- At least one AI CLI tool (Claude Code, Cursor, Antigravity, etc.)

### Installation
Primary: `go install github.com/markx3/agentboard/cmd/agentboard@latest`
Secondary: Build from source (`go build -o agentboard ./cmd/agentboard`)
Future: Homebrew tap

### Quick Start
3-step flow:
1. `agentboard init` — initialize project
2. `agentboard` — launch TUI
3. Team members run `agentboard` in the same repo (or `agentboard --connect host:port`)

### Supported Agents Table
| Agent | Status |
| Claude Code | Supported |
| Cursor CLI | Supported |
| Antigravity | Supported |
| (others) | Detection via PATH |

### CLI Reference
Document all commands from Cobra:
- `agentboard` (TUI)
- `agentboard init`
- `agentboard serve`
- `agentboard status [--json]`
- `agentboard task {list,create,move,claim,unclaim,get,delete} [--json]`
- `agentboard peers`

### Architecture
Mermaid diagram showing:
- TUI Layer (Bubble Tea)
- Board Service
- SQLite (local state)
- WebSocket Server/Client (sync)
- Agent Orchestration (tmux + worktrees)

### Roadmap
- Phase 1-4: Complete
- Phase 5: Polish and agent CLI integration (current)
- Future: Homebrew distribution, optional Review/Compound columns

## Open Questions

None — all key decisions resolved through dialogue.
