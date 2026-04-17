// Package logx provides structured logging and test artifact dumping.
package logx

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// artifactDirEnv is the environment variable that specifies an additional
// directory for test artifacts (e.g., for CI upload-artifact).
const artifactDirEnv = "LIBTFTEST_ARTIFACT_DIR"

// NewLogger returns an slog.Logger scoped to the given test name.
func NewLogger(tb testing.TB) *slog.Logger {
	tb.Helper()

	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})).With("test", tb.Name())
}

// DumpArtifact writes data to a named file under the given artifact directory.
// If LIBTFTEST_ARTIFACT_DIR is set, a copy is also written there for CI
// artifact collection.
func DumpArtifact(tb testing.TB, artifactDir, name string, data []byte) {
	tb.Helper()

	writeFile(tb, artifactDir, name, data)

	// Optionally write to the CI artifact dir.
	if ciBase := os.Getenv(artifactDirEnv); ciBase != "" {
		ciDir := filepath.Join(ciBase, tb.Name())
		writeFile(tb, ciDir, name, data)
	}
}

// ResolveArtifactDir returns the artifact directory for a test. If
// LIBTFTEST_ARTIFACT_DIR is set, returns a test-scoped subdirectory there.
// Otherwise returns a "libtftest-artifacts" subdirectory under baseDir.
func ResolveArtifactDir(tb testing.TB, baseDir string) string {
	tb.Helper()

	if ciBase := os.Getenv(artifactDirEnv); ciBase != "" {
		return filepath.Join(ciBase, tb.Name())
	}

	return filepath.Join(baseDir, "libtftest-artifacts")
}

func writeFile(tb testing.TB, dir, name string, data []byte) {
	tb.Helper()

	//nolint:gosec // Artifact dirs are test-scoped, not user-controlled in production.
	if err := os.MkdirAll(dir, 0o750); err != nil {
		tb.Errorf("create artifact dir %s: %v", dir, err)
		return
	}

	path := filepath.Join(dir, name)
	//nolint:gosec // Artifact paths are test-scoped, not user-controlled in production.
	if err := os.WriteFile(path, data, 0o640); err != nil {
		tb.Errorf("write artifact %s: %v", path, err)
		return
	}

	tb.Logf("artifact written: %s", path)
}
