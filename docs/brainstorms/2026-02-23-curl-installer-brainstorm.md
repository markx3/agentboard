# Brainstorm: cURL Install Command for Agentboard

**Date:** 2026-02-23
**Status:** Draft

## What We're Building

A `curl | bash` one-liner that installs the `agentboard` binary on macOS and Linux, backed by a GitHub Actions release pipeline that builds cross-platform binaries on git tag push.

The end-user experience:

```bash
curl -fsSL https://raw.githubusercontent.com/markx3/agentboard/main/install.sh | bash
```

## Why This Approach

- **Single command install** removes the Go toolchain prerequisite for end users
- **GitHub Releases** is the standard distribution mechanism for Go CLI tools
- **Shell script + pre-built binaries** is the pattern used by widely adopted tools (Homebrew, Deno, Bun, Fly.io, etc.)
- **Pure Go (no CGO)** makes cross-compilation trivial — no C toolchain needed in CI

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scope | Full pipeline | GitHub Actions + install script + README update |
| Platforms | macOS + Linux | darwin/{amd64,arm64}, linux/{amd64,arm64} — covers most developers |
| Install path | `/usr/local/bin` | Standard location, widely in PATH |
| GitHub owner | `markx3` | All release URLs and script references use `markx3/agentboard` |
| Archive format | `.tar.gz` | Standard for Unix platforms, easy to extract |
| Script hosting | `install.sh` at repo root | Raw GitHub URL, versioned with the repo |
| Release trigger | Git tag (`v*`) | Push a tag like `v0.1.0` to trigger the release workflow |

## Components

### 1. GitHub Actions Release Workflow (`.github/workflows/release.yml`)

- Triggers on `v*` tag push
- Builds binaries for 4 platform/arch combos
- Archives each as `agentboard-{os}-{arch}.tar.gz`
- Creates a GitHub Release with the archives attached
- Uses `GITHUB_TOKEN` for release creation (no extra secrets needed)

### 2. Install Script (`install.sh`)

- Detects OS (`uname -s`) and architecture (`uname -m`)
- Maps to the correct release archive name
- Fetches the latest release tag from GitHub API
- Downloads the archive, extracts the binary, moves to `/usr/local/bin`
- Uses `sudo` if needed for `/usr/local/bin`
- Verifies the binary works with `agentboard --version` (if available)
- Clean error messages if platform is unsupported or download fails

### 3. README Update

- Add a prominent "Install" section with the curl one-liner
- Keep existing `go install` and build-from-source options as alternatives

## Open Questions

None — all key decisions resolved.

## Next Steps

Proceed to `/workflows:plan` to define the implementation steps.
