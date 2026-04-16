package tf

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

var (
	pluginCacheDirOnce sync.Once
	pluginCacheDirPath string
)

// BuildOptions constructs a terraform.Options suitable for running against
// a LocalStack-backed workspace.
func BuildOptions(tb testing.TB, workDir string, vars map[string]any) *terraform.Options {
	tb.Helper()

	return terraform.WithDefaultRetryableErrors(tb, &terraform.Options{
		TerraformDir: workDir,
		Vars:         vars,
		EnvVars: map[string]string{
			"AWS_ACCESS_KEY_ID":     "test",
			"AWS_SECRET_ACCESS_KEY": "test",
			"AWS_DEFAULT_REGION":    "us-east-1",
			"TF_PLUGIN_CACHE_DIR":   PluginCacheDir(),
			"TF_IN_AUTOMATION":      "1",
		},
		NoColor:      true,
		Lock:         true,
		LockTimeout:  "60s",
		Logger:       logger.Discard,
		PlanFilePath: filepath.Join(workDir, "libtftest.plan"),
	})
}

// PluginCacheDir returns a stable directory for Terraform's plugin cache.
// On macOS it uses ~/Library/Caches/libtftest/plugin-cache; on Linux it
// uses $XDG_CACHE_HOME/libtftest/plugin-cache (falling back to ~/.cache).
// The directory is created on first call.
func PluginCacheDir() string {
	pluginCacheDirOnce.Do(func() {
		pluginCacheDirPath = resolvePluginCacheDir()
	})

	return pluginCacheDirPath
}

func resolvePluginCacheDir() string {
	var base string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.TempDir()
		}
		base = filepath.Join(home, "Library", "Caches")
	default:
		if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
			base = xdg
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				home = os.TempDir()
			}
			base = filepath.Join(home, ".cache")
		}
	}

	dir := filepath.Join(base, "libtftest", "plugin-cache")
	if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // Cache dir is under user's home or XDG_CACHE_HOME by design.
		// Fall back to a temp dir if cache creation fails.
		return filepath.Join(os.TempDir(), "libtftest-plugin-cache")
	}

	return dir
}
