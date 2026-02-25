package agent

import "github.com/markx3/agentboard/internal/db"

// AgentRunner abstracts an AI agent CLI.
type AgentRunner interface {
	ID() string                                  // Canonical DB identifier (e.g., "claude", "cursor")
	Name() string                                // Display name (e.g., "Claude Code", "Cursor")
	Binary() string                              // Executable name for PATH lookup
	Available() bool                             // Is this agent detected and verified?
	BuildCommand(opts SpawnOpts) string           // Build the full shell command for task work
	BuildEnrichmentCommand(opts SpawnOpts) string // Build enrichment command ("" if unsupported)
}

// SpawnOpts holds the context needed to build an agent command.
type SpawnOpts struct {
	WorkDir string
	Task    db.Task
	ExePath string // Absolute path to agentboard binary
}

var runners = []AgentRunner{
	&ClaudeRunner{},
	&CursorRunner{},
}

// AvailableRunners returns all runners whose CLI binary is detected.
func AvailableRunners() []AgentRunner {
	var available []AgentRunner
	for _, r := range runners {
		if r.Available() {
			available = append(available, r)
		}
	}
	return available
}

// GetRunner looks up a runner by its canonical ID.
func GetRunner(id string) AgentRunner {
	for _, r := range runners {
		if r.ID() == id {
			return r
		}
	}
	return nil
}
