// Package tmux encapsulates all tmux shell commands for managing agent windows.
// It uses a dedicated socket (-L agentboard) to isolate agent windows from the user's sessions.
package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const socket = "agentboard"

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// InTmux reports whether the current process is running inside tmux.
func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

// EnsureSession creates the agentboard tmux session if it doesn't already exist.
// It also binds Ctrl+q to detach so agent panes can be closed easily.
func EnsureSession() error {
	// Check if session already exists
	err := exec.Command("tmux", "-L", socket, "has-session", "-t", socket).Run()
	if err == nil {
		return nil
	}
	// Create a detached session
	if err := exec.Command("tmux", "-L", socket, "new-session", "-d", "-s", socket).Run(); err != nil {
		return err
	}
	// Bind Ctrl+q to detach (no-prefix) so split panes close easily
	_ = exec.Command("tmux", "-L", socket, "bind", "-n", "C-q", "detach-client").Run()
	return nil
}

// NewWindow launches a command in a named tmux window within the agentboard session.
func NewWindow(name, dir, command string) error {
	safe := sanitizeName(name)
	args := []string{"-L", socket, "new-window", "-t", socket, "-n", safe}
	if dir != "" {
		args = append(args, "-c", dir)
	}
	args = append(args, command)
	return exec.Command("tmux", args...).Run()
}

// KillWindow kills a tmux window by name (best-effort).
func KillWindow(name string) error {
	safe := sanitizeName(name)
	target := fmt.Sprintf("%s:%s", socket, safe)
	return exec.Command("tmux", "-L", socket, "kill-window", "-t", target).Run()
}

// ListWindows returns the set of live window names in the agentboard session.
func ListWindows() (map[string]bool, error) {
	out, err := exec.Command(
		"tmux", "-L", socket, "list-windows", "-t", socket, "-F", "#{window_name}",
	).Output()
	if err != nil {
		// Session doesn't exist or tmux not running — no windows alive
		return map[string]bool{}, nil
	}

	windows := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			windows[line] = true
		}
	}
	return windows, nil
}

// IsWindowAlive checks if a window with the given name exists in the agentboard session.
func IsWindowAlive(windowName string) bool {
	windows, _ := ListWindows()
	return windows[windowName]
}

// SplitView opens a horizontal tmux split in the caller's current window,
// attaching to the agentboard session with the given window focused.
// This requires the caller to be running inside tmux.
func SplitView(windowName string) error {
	safe := sanitizeName(windowName)
	target := fmt.Sprintf("%s:%s", socket, safe)
	// First select the correct window in the agentboard session
	_ = exec.Command("tmux", "-L", socket, "select-window", "-t", target).Run()
	// Split the caller's tmux pane and attach to the agentboard session
	cmd := fmt.Sprintf("tmux -L %s attach-session -t %s", socket, socket)
	return exec.Command("tmux", "split-window", "-h", "-l", "50%", cmd).Run()
}

// AttachCmd returns an exec.Cmd that attaches to the agentboard tmux session
// with the given window focused. The caller is expected to run this via tea.ExecProcess.
func AttachCmd(windowName string) *exec.Cmd {
	safe := sanitizeName(windowName)
	target := fmt.Sprintf("%s:%s", socket, safe)
	return exec.Command("tmux", "-L", socket, "select-window", "-t", target, ";", "attach-session", "-t", socket)
}

// sanitizeName strips shell metacharacters from a window name.
func sanitizeName(name string) string {
	return unsafeChars.ReplaceAllString(name, "_")
}
