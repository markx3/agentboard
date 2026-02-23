package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const serverName = "agentboard"

type Manager struct {
	session string
}

func NewManager(projectName string) *Manager {
	if projectName == "" {
		projectName = "default"
	}
	return &Manager{session: "agentboard:" + projectName}
}

func (m *Manager) EnsureSession(ctx context.Context) error {
	// Check if session exists
	cmd := exec.CommandContext(ctx, "tmux", "-L", serverName, "has-session", "-t", m.session)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Create session in detached mode
	cmd = exec.CommandContext(ctx, "tmux", "-L", serverName, "new-session", "-d", "-s", m.session)
	return cmd.Run()
}

func (m *Manager) NewWindow(ctx context.Context, name, dir, command string) error {
	if err := m.EnsureSession(ctx); err != nil {
		return fmt.Errorf("ensuring session: %w", err)
	}

	args := []string{"-L", serverName, "new-window", "-t", m.session, "-n", name}
	if dir != "" {
		args = append(args, "-c", dir)
	}
	if command != "" {
		args = append(args, command)
	}

	cmd := exec.CommandContext(ctx, "tmux", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating window: %s: %w", string(out), err)
	}
	return nil
}

func (m *Manager) KillWindow(ctx context.Context, name string) error {
	target := m.session + ":" + name
	cmd := exec.CommandContext(ctx, "tmux", "-L", serverName, "kill-window", "-t", target)
	return cmd.Run()
}

func (m *Manager) HasWindow(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "tmux", "-L", serverName, "list-windows", "-t", m.session, "-F", "#{window_name}")
	out, err := cmd.Output()
	if err != nil {
		return false, nil // Session may not exist
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}
	return false, nil
}

func (m *Manager) SendKeys(ctx context.Context, windowName, keys string) error {
	target := m.session + ":" + windowName
	cmd := exec.CommandContext(ctx, "tmux", "-L", serverName, "send-keys", "-t", target, keys, "Enter")
	return cmd.Run()
}

func (m *Manager) KillServer(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "tmux", "-L", serverName, "kill-server")
	return cmd.Run()
}
