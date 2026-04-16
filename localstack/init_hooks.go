package localstack

import (
	"fmt"
	"os"
	"path/filepath"
)

// InitHook defines a script to run inside the LocalStack container during
// its ready.d init phase.
type InitHook struct {
	Name   string // Becomes the filename in /etc/localstack/init/ready.d/.
	Script string // Bash script content.
}

// WriteInitHooks writes all hooks to a temp directory and returns its path.
// The directory is suitable for bind-mounting at /etc/localstack/init/ready.d/.
func WriteInitHooks(hooks []InitHook) (string, error) {
	dir, err := os.MkdirTemp("", "libtftest-init-hooks-*")
	if err != nil {
		return "", fmt.Errorf("create init hooks dir: %w", err)
	}

	for _, h := range hooks {
		path := filepath.Join(dir, h.Name)
		if err := os.WriteFile(path, []byte(h.Script), 0o755); err != nil {
			return "", fmt.Errorf("write init hook %s: %w", h.Name, err)
		}
	}

	return dir, nil
}
