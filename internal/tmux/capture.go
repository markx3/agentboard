package tmux

import (
	"context"
	"fmt"
	"os/exec"
)

const captureLines = 500

func (m *Manager) CapturePane(ctx context.Context, windowName string) (string, error) {
	target := m.session + ":" + windowName
	cmd := exec.CommandContext(ctx, "tmux", "-L", serverName,
		"capture-pane", "-t", target, "-p", "-S", fmt.Sprintf("-%d", captureLines))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("capturing pane: %w", err)
	}
	return string(out), nil
}
