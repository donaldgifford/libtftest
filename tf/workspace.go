// Package tf handles Terraform workspace management, override rendering, and options construction.
package tf

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Workspace holds a scratch copy of a Terraform module directory.
// The copy is rooted under t.TempDir() so it's cleaned up automatically.
type Workspace struct {
	Dir string // Root of the scratch copy.
	src string // Original module directory.
}

// NewWorkspace copies moduleDir into a temp directory and returns a Workspace.
// The original module is not modified. The caller can freely write overrides
// and run terraform in the scratch copy.
func NewWorkspace(tb testing.TB, moduleDir string) *Workspace {
	tb.Helper()

	dst := filepath.Join(tb.TempDir(), "module")
	if err := copyTree(moduleDir, dst); err != nil {
		tb.Fatalf("copy module %s: %v", moduleDir, err)
	}

	return &Workspace{Dir: dst, src: moduleDir}
}

// copyTree recursively copies src to dst. Symlinks are followed once (to
// support modules/shared -> ../shared patterns) and then rejected to avoid
// cycles. Regular files are copied with their permissions preserved.
func copyTree(src, dst string) error {
	seen := make(map[string]bool)

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("compute relative path: %w", err)
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}

		// Handle symlinks: resolve once, reject cycles.
		if d.Type()&fs.ModeSymlink != 0 {
			return copySymlink(path, target, seen)
		}

		return copyFile(path, target)
	})
}

func copySymlink(path, target string, seen map[string]bool) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve symlink %s: %w", path, err)
	}

	if seen[resolved] {
		return nil // Skip cycle.
	}
	seen[resolved] = true

	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("stat resolved symlink %s: %w", resolved, err)
	}

	if info.IsDir() {
		return copyTree(resolved, target)
	}

	return copyFile(resolved, target)
}

func copyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source %s: %w", src, err)
	}
	defer sf.Close() //nolint:errcheck // Read-only file, close error is not actionable.

	info, err := sf.Stat()
	if err != nil {
		return fmt.Errorf("stat source %s: %w", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", dst, err)
	}

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("create destination %s: %w", dst, err)
	}
	defer df.Close() //nolint:errcheck // Write is flushed by io.Copy; close error is not actionable.

	if _, err := io.Copy(df, sf); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}

	return nil
}
