package tmux

import (
	"os/exec"
	"strings"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

// WindowName returns the tmux window name for a given task.
func WindowName(task *db.Task) string {
	return "agent-" + task.ID[:8]
}

// ListWindows returns the names of all tmux windows in the current session.
// Returns an empty slice (not an error) if tmux is not running.
func ListWindows() []string {
	out, err := exec.Command("tmux", "list-windows", "-F", "#{window_name}").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

// IsWindowAlive checks if a window with the given name exists.
func IsWindowAlive(windowName string) bool {
	for _, w := range ListWindows() {
		if w == windowName {
			return true
		}
	}
	return false
}

// KillWindow kills a tmux window by name. No error if it doesn't exist.
func KillWindow(windowName string) {
	exec.Command("tmux", "kill-window", "-t", windowName).Run()
}
