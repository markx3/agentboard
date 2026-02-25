---
title: "feat: Add cURL install command"
type: feat
date: 2026-02-23
brainstorm: docs/brainstorms/2026-02-23-curl-installer-brainstorm.md
---

# feat: Add cURL install command

## Overview

Add a one-liner cURL install command for agentboard, backed by a GitHub Actions release pipeline that builds cross-platform binaries on git tag push. This removes the Go toolchain requirement for end users.

End result:

```bash
curl -fsSL https://raw.githubusercontent.com/markx3/agentboard/main/install.sh | bash
```

## Proposed Solution

Three components, built in dependency order:

1. **Version flag** — Wire `--version` into the CLI via Cobra + ldflags injection
2. **GitHub Actions release workflow** — Build cross-platform binaries on `v*` tag push, create GitHub Release
3. **Install script** — Shell script that detects OS/arch, downloads the right binary, installs to `/usr/local/bin`
4. **README update** — Add curl one-liner as the primary install method

## Technical Approach

### Phase 1: Add `--version` flag

**Files:**
- `internal/cli/root.go` — Add `Version` variable and wire into `rootCmd.Version`

**Implementation:**

Add an exported `Version` variable defaulting to `"dev"`, set it on the Cobra root command:

```go
// internal/cli/root.go

var Version = "dev"

var rootCmd = &cobra.Command{
    Use:     "agentboard",
    Short:   "Collaborative agentic task management TUI",
    Long:    "A terminal-based collaborative Kanban board for managing agentic coding tasks across a team.",
    Version: Version,
    RunE:    runBoard,
}
```

At build time, inject via ldflags:

```bash
go build -ldflags "-X github.com/markx3/agentboard/internal/cli.Version=v0.1.0" -o agentboard ./cmd/agentboard
```

Cobra auto-generates `--version` flag when `Version` is set. Output: `agentboard version v0.1.0`.

**Acceptance criteria:**
- [x] `agentboard --version` prints `agentboard version dev` when built without ldflags
- [x] `agentboard --version` prints `agentboard version v0.1.0` when built with `-ldflags "-X github.com/markx3/agentboard/internal/cli.Version=v0.1.0"`

---

### Phase 2: GitHub Actions release workflow

**Files:**
- `.github/workflows/release.yml` — New file

**Trigger:** Push of tags matching `v[0-9]*` (filters garbage tags like `vtest`).

**Build matrix:**

| GOOS | GOARCH | Archive name |
|------|--------|------|
| darwin | amd64 | `agentboard-darwin-amd64.tar.gz` |
| darwin | arm64 | `agentboard-darwin-arm64.tar.gz` |
| linux | amd64 | `agentboard-linux-amd64.tar.gz` |
| linux | arm64 | `agentboard-linux-arm64.tar.gz` |

**Key details:**
- Set `CGO_ENABLED=0` for all builds (pure Go, no C toolchain needed)
- Inject version via ldflags: `-X github.com/markx3/agentboard/internal/cli.Version=${{ github.ref_name }}`
- **Archive structure:** Each `.tar.gz` contains a single file named `agentboard` at the root (no subdirectory)
- Generate `checksums.txt` with SHA256 hashes for all archives
- Release creation is gated on all matrix builds succeeding (`needs: [build]`)
- Uses `GITHUB_TOKEN` — no extra secrets needed

**Workflow structure:**

```yaml
# Pseudocode — two jobs
jobs:
  build:
    strategy:
      matrix:
        - {goos: darwin, goarch: amd64}
        - {goos: darwin, goarch: arm64}
        - {goos: linux, goarch: amd64}
        - {goos: linux, goarch: arm64}
    steps:
      - checkout
      - setup-go
      - go build with GOOS/GOARCH/CGO_ENABLED=0/ldflags
      - tar -czf agentboard-{os}-{arch}.tar.gz agentboard
      - upload artifact

  release:
    needs: [build]
    steps:
      - download all artifacts
      - generate checksums.txt (sha256sum *.tar.gz)
      - create GitHub Release with all .tar.gz + checksums.txt
```

**Acceptance criteria:**
- [x] Pushing a `v*` tag triggers the workflow
- [x] All 4 platform binaries are built and archived
- [x] A GitHub Release is created with 4 `.tar.gz` files + `checksums.txt`
- [x] Release is NOT created if any platform build fails
- [x] The binary inside each archive is named `agentboard` (not `agentboard-darwin-amd64`)

---

### Phase 3: Install script

**Files:**
- `install.sh` — New file at repo root

**Behavior:**

1. Detect OS via `uname -s` → map `Darwin` to `darwin`, `Linux` to `linux`
2. Detect arch via `uname -m` → map `x86_64` to `amd64`, `arm64`/`aarch64` to `arm64`
3. Fetch latest release tag from GitHub API (`/repos/markx3/agentboard/releases/latest`)
4. Download `agentboard-{os}-{arch}.tar.gz` to a temp directory (`mktemp -d`)
5. Verify SHA256 checksum against `checksums.txt`
6. Extract the binary from the archive
7. Install to `INSTALL_DIR` (default `/usr/local/bin`), using `sudo` if needed
8. Verify with `agentboard --version`
9. Clean up temp directory (via trap on EXIT)
10. Print success message with quick-start hint

**Edge cases to handle:**

| Case | Behavior |
|------|----------|
| `uname -m` returns `aarch64` (Linux ARM) | Normalize to `arm64` |
| Unsupported OS (FreeBSD, Windows/MINGW) | Exit with clear error message |
| No releases exist (HTTP 404) | Exit: "No releases found. Install from source." |
| GitHub API rate limited (HTTP 403) | Exit: "GitHub API rate limited. Try again later or install from source." |
| `/usr/local/bin` not writable, no sudo | Print warning, suggest `INSTALL_DIR=~/.local/bin` |
| Download fails (network error) | Exit with error, clean up temp files |
| Checksum mismatch | Exit: "Checksum verification failed" |
| Binary already exists (upgrade) | Overwrite silently, print old → new version |
| `curl` not available | Try `wget` as fallback |

**Environment variable overrides:**
- `INSTALL_DIR` — Override install location (default: `/usr/local/bin`)
- `AGENTBOARD_VERSION` — Install a specific version instead of latest

**Post-install message:**

```
agentboard v0.1.0 installed to /usr/local/bin/agentboard

Get started:
  agentboard init    # initialize in your project
  agentboard         # launch the TUI

Prerequisites: tmux (3.0+), gh CLI (2.0+)
```

**Acceptance criteria:**
- [x] `curl -fsSL .../install.sh | bash` installs the binary on macOS (Intel + Apple Silicon)
- [x] `curl -fsSL .../install.sh | bash` installs the binary on Linux (amd64 + arm64)
- [x] Script exits with clear error on unsupported platforms
- [x] Script exits with clear error when no releases exist
- [x] Script handles `aarch64` → `arm64` mapping correctly
- [x] `INSTALL_DIR` env var overrides install path
- [x] Temp files are cleaned up on success and failure
- [x] Checksum is verified before installing

---

### Phase 4: README update

**Files:**
- `README.md` — Edit existing Installation section

**Changes:**

1. Remove Go from Prerequisites table (it's only needed for building from source now)
2. Restructure Installation section to lead with curl, then alternatives:

```markdown
## Installation

### Quick install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/markx3/agentboard/main/install.sh | bash
```

Or if you prefer to inspect the script first:

```bash
curl -fsSL https://raw.githubusercontent.com/markx3/agentboard/main/install.sh -o install.sh
less install.sh
bash install.sh
```

### Other methods

```bash
# Via go install (requires Go 1.25+)
go install github.com/markx3/agentboard/cmd/agentboard@latest

# Or build from source
git clone https://github.com/markx3/agentboard.git
cd agentboard
go build -o agentboard ./cmd/agentboard
```
```

3. Update Prerequisites table: move Go to a footnote ("only needed for building from source")

**Acceptance criteria:**
- [x] Curl one-liner is the first install option shown
- [x] "Inspect before run" alternative is documented
- [x] `go install` and build-from-source remain as alternatives
- [x] Prerequisites table no longer lists Go as required for basic install

## Dependencies & Risks

| Risk | Mitigation |
|------|------------|
| Repo is private — raw.githubusercontent.com won't serve `install.sh` publicly | Must make repo public before the curl command works for external users, OR document that users need a GitHub token |
| No existing releases — install script will 404 until first tag is pushed | Document in README that a release must exist; include `go install` as fallback |
| GitHub API rate limit (60/hr unauthenticated) | Script handles 403 gracefully with a clear message |

## Build Sequence

```
Phase 1: --version flag (internal/cli/root.go)
    ↓
Phase 2: GitHub Actions workflow (.github/workflows/release.yml)
    ↓
Phase 3: Install script (install.sh)
    ↓
Phase 4: README update (README.md)
```

Phases 1→2 are strictly sequential (workflow needs the version variable to exist). Phases 3 and 4 can be done in parallel after Phase 2 is written (they don't depend on a real release existing).

## References

- Brainstorm: `docs/brainstorms/2026-02-23-curl-installer-brainstorm.md`
- Entry point: `cmd/agentboard/main.go`
- CLI root: `internal/cli/root.go:18-23`
- Current install docs: `README.md:24-33`
