package worktree

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func (m *Manager) Setup(ctx context.Context, worktreePath string, copyFiles []string, initScript string) error {
	for _, f := range copyFiles {
		if err := copyFile(f, filepath.Join(worktreePath, f)); err != nil {
			// Non-fatal: file may not exist in the source project
			continue
		}
	}

	if initScript != "" {
		cmd := exec.CommandContext(ctx, "sh", "-c", initScript)
		cmd.Dir = worktreePath
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("running init script: %s: %w", string(out), err)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
