//go:build linux

package ollama

import (
	"context"
	"fmt"
	"os"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

const installScriptURL = "https://ollama.com/install.sh"

// Manager handles Ollama installation and model management.
type Manager struct {
	Runner exec.Runner
	Logger *logging.Logger
}

// NewManager creates a Manager with the given runner and logger.
func NewManager(runner exec.Runner, logger *logging.Logger) *Manager {
	return &Manager{Runner: runner, Logger: logger}
}

// IsInstalled returns true if the ollama binary is available.
func (m *Manager) IsInstalled(ctx context.Context) (bool, error) {
	_, err := m.Runner.Run(ctx, "ollama", "--version")
	if err != nil {
		return false, nil //nolint:nilerr // expected: ollama not found
	}
	return true, nil
}

// Install checks for an existing ollama installation. If missing, it downloads
// the official install script via curl and executes it with sh.
func (m *Manager) Install(ctx context.Context) error {
	installed, err := m.IsInstalled(ctx)
	if err != nil {
		return fmt.Errorf("check ollama: %w", err)
	}
	if installed {
		m.Logger.Info("ollama is already installed, skipping")
		return nil
	}

	m.Logger.Info("installing ollama via official install script")

	// Download the install script to a temp file.
	tmpFile, err := os.CreateTemp("", "ollama-install-*.sh")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck // best-effort cleanup
	tmpFile.Close()                 //nolint:errcheck,gosec // closed before use by curl

	_, err = m.Runner.Run(ctx, "curl", "-fsSL", "-o", tmpFile.Name(), installScriptURL)
	if err != nil {
		return fmt.Errorf("download install script: %w", err)
	}

	_, err = m.Runner.Run(ctx, "sh", tmpFile.Name())
	if err != nil {
		return fmt.Errorf("run install script: %w", err)
	}

	// Verify installation succeeded.
	installed, err = m.IsInstalled(ctx)
	if err != nil {
		return fmt.Errorf("verify ollama post-install: %w", err)
	}
	if !installed {
		return fmt.Errorf("ollama installation completed but binary not found")
	}

	m.Logger.Info("ollama installed")
	return nil
}
