package worktree

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Manager struct {
	baseDir string
}

func NewManager() *Manager {
	return &Manager{
		baseDir: filepath.Join(".agentboard", "worktrees"),
	}
}

func (m *Manager) Create(ctx context.Context, taskSlug string) (string, error) {
	branchName := "agentboard/" + taskSlug
	worktreePath := filepath.Join(m.baseDir, taskSlug)

	cmd := exec.CommandContext(ctx, "git", "worktree", "add", worktreePath, "-b", branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("creating worktree: %s: %w", string(out), err)
	}

	return worktreePath, nil
}

func (m *Manager) Remove(ctx context.Context, taskSlug string) error {
	worktreePath := filepath.Join(m.baseDir, taskSlug)

	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", worktreePath, "--force")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("removing worktree: %s: %w", string(out), err)
	}
	return nil
}

func (m *Manager) HasUncommittedChanges(ctx context.Context, taskSlug string) (bool, error) {
	worktreePath := filepath.Join(m.baseDir, taskSlug)
	cmd := exec.CommandContext(ctx, "git", "-C", worktreePath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("checking status: %w", err)
	}
	return len(strings.TrimSpace(string(out))) > 0, nil
}

func (m *Manager) Path(taskSlug string) string {
	return filepath.Join(m.baseDir, taskSlug)
}

func slugify(title string) string {
	replacer := strings.NewReplacer(
		" ", "-", "/", "-", "\\", "-",
		":", "", "'", "", "\"", "",
		".", "-", ",", "",
	)
	slug := strings.ToLower(replacer.Replace(title))
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return strings.Trim(slug, "-")
}

func SlugFromTitle(title string) string {
	return slugify(title)
}
