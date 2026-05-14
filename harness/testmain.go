package harness

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/donaldgifford/libtftest/localstack"
)

// Config configures the shared container started by Run.
type Config struct {
	Edition  localstack.Edition
	Image    string
	Services []string
	Sidecars []Sidecar
}

var (
	shared    *localstack.Container
	sharedMu  sync.Mutex
	edgeURL   string
	sidecarMu sync.Mutex
)

// Current returns the shared container started by Run, or nil if Run has not
// been called. This is the auto-detection mechanism used by libtftest.New.
func Current() *localstack.Container {
	sharedMu.Lock()
	defer sharedMu.Unlock()

	return shared
}

// EdgeURL returns the effective edge URL. If sidecars are running, this is
// the sidecar's URL; otherwise, the container's edge URL.
func EdgeURL() string {
	sharedMu.Lock()
	defer sharedMu.Unlock()

	return edgeURL
}

// Run starts a shared LocalStack container, optionally starts sidecars,
// runs the test suite, and cleans up. This is designed to be called from
// TestMain.
func Run(m *testing.M, cfg Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Start the shared container.
	lsCfg := &localstack.Config{
		Edition:  cfg.Edition,
		Image:    cfg.Image,
		Services: cfg.Services,
	}

	ctr, err := localstack.Start(ctx, lsCfg)
	if err != nil {
		cancel()
		logger.Error("start shared container", "error", err)
		os.Exit(1)
	}

	sharedMu.Lock()
	shared = ctr
	edgeURL = ctr.EdgeURL
	sharedMu.Unlock()

	logger.Info("shared container started", "id", ctr.ID, "url", ctr.EdgeURL)

	// Start sidecars.
	activeSidecars := make([]Sidecar, 0, len(cfg.Sidecars))
	for _, sc := range cfg.Sidecars {
		url, err := sc.Start(ctx, ctr.EdgeURL)
		if err != nil {
			cancel()
			logger.Error("start sidecar", "error", err)
			stopAll(context.Background(), logger, activeSidecars, ctr)
			os.Exit(1)
		}

		activeSidecars = append(activeSidecars, sc)

		sidecarMu.Lock()
		edgeURL = url
		sidecarMu.Unlock()

		logger.Info("sidecar started", "url", url)
	}

	// Cancel the startup context before running tests.
	cancel()

	// Run tests.
	code := m.Run()

	// Cleanup: stop sidecars first (reverse order), then container.
	stopAll(ctx, logger, activeSidecars, ctr)

	sharedMu.Lock()
	shared = nil
	edgeURL = ""
	sharedMu.Unlock()

	os.Exit(code)
}

func stopAll(ctx context.Context, logger *slog.Logger, sidecars []Sidecar, ctr *localstack.Container) {
	// Stop sidecars in reverse order.
	for i := len(sidecars) - 1; i >= 0; i-- {
		if err := sidecars[i].Stop(ctx); err != nil {
			logger.Error("stop sidecar", "error", err)
		}
	}

	if err := ctr.Stop(ctx); err != nil {
		logger.Error("stop container", "error", err)
	}
}

// PrefixWarning emits a warning when duplicate prefixes are detected,
// indicating a test forgot to use tc.Prefix() for namespacing.
func PrefixWarning(tb testing.TB, prefix string) {
	tb.Helper()

	prefixMu.Lock()
	defer prefixMu.Unlock()

	if seenPrefixes[prefix] {
		tb.Errorf("duplicate prefix %q detected — ensure each test uses tc.Prefix() for resource namespacing", prefix)
		return
	}

	seenPrefixes[prefix] = true
}

var (
	prefixMu     sync.Mutex
	seenPrefixes = make(map[string]bool)
)

// ResetPrefixes clears the prefix tracker. Used in tests.
func ResetPrefixes() {
	prefixMu.Lock()
	defer prefixMu.Unlock()

	seenPrefixes = make(map[string]bool)
}

// FormatContainerInfo returns a human-readable summary of the shared container.
func FormatContainerInfo() string {
	ctr := Current()
	if ctr == nil {
		return "no shared container"
	}

	return fmt.Sprintf("container=%s url=%s edition=%s", ctr.ID[:12], ctr.EdgeURL, ctr.Edition)
}
