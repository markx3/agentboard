package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/marcosfelipeeipper/agentboard/internal/tmux"
)

func Spawn(ctx context.Context, tm *tmux.Manager, ag *Agent, taskTitle, worktreePath string) error {
	// Sanitize task title for use as tmux window name
	windowName := sanitizeWindowName(taskTitle)

	cmdParts := []string{ag.Command}
	cmdParts = append(cmdParts, ag.Args...)
	fullCmd := strings.Join(cmdParts, " ")

	return tm.NewWindow(ctx, windowName, worktreePath, fullCmd)
}

func sanitizeWindowName(name string) string {
	// Remove characters that are problematic in tmux window names
	replacer := strings.NewReplacer(
		".", "-", ":", "-", " ", "-",
		"'", "", "\"", "", "`", "",
		"$", "", ";", "", "&", "",
		"|", "", "(", "", ")", "",
	)
	result := replacer.Replace(name)
	if len(result) > 40 {
		result = result[:40]
	}
	return result
}

func Kill(ctx context.Context, tm *tmux.Manager, taskTitle string) error {
	windowName := sanitizeWindowName(taskTitle)
	return tm.KillWindow(ctx, windowName)
}

func Status(ctx context.Context, tm *tmux.Manager, taskTitle string) (string, error) {
	windowName := sanitizeWindowName(taskTitle)
	ok, err := tm.HasWindow(ctx, windowName)
	if err != nil {
		return "error", fmt.Errorf("checking window: %w", err)
	}
	if !ok {
		return "idle", nil
	}
	return "active", nil
}
