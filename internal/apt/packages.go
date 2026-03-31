//go:build linux

// Package apt manages system packages and GPG keyring setup on Ubuntu/Debian.
// It shells out via the exec.Runner interface.
package apt

import (
	"context"
	"fmt"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// Manager handles apt package operations and keyring setup.
type Manager struct {
	Runner exec.Runner
	Logger *logging.Logger
}

// NewManager creates a Manager with the given runner and logger.
func NewManager(runner exec.Runner, logger *logging.Logger) *Manager {
	return &Manager{
		Runner: runner,
		Logger: logger,
	}
}

// retryOpts returns the standard retry options for apt operations.
func retryOpts() exec.RetryOpts {
	return exec.DefaultRetryOpts()
}

// UpdatePackages runs apt-get update, upgrade, dist-upgrade, autoremove, and
// autoclean with retry on each step. Uses DEBIAN_FRONTEND=noninteractive
// to suppress interactive prompts during upgrades.
func (m *Manager) UpdatePackages(ctx context.Context) error {
	m.Logger.Info("updating system packages")

	commands := [][]string{
		{"sh", "-c", "apt-get update"},
		{"sh", "-c", "DEBIAN_FRONTEND=noninteractive apt-get upgrade -y"},
		{"sh", "-c", "DEBIAN_FRONTEND=noninteractive apt-get dist-upgrade -y"},
		{"sh", "-c", "apt-get autoremove -y"},
		{"sh", "-c", "apt-get autoclean -y"},
	}

	for _, cmd := range commands {
		_, err := m.Runner.RunWithRetry(ctx, retryOpts(), cmd[0], cmd[1:]...)
		if err != nil {
			return fmt.Errorf("%s: %w", cmd[2], err)
		}
	}

	m.Logger.Info("system packages updated")
	return nil
}

// InstallPackages installs the given packages via apt-get install -y with retry.
// Uses DEBIAN_FRONTEND=noninteractive to suppress interactive prompts.
func (m *Manager) InstallPackages(ctx context.Context, pkgs ...string) error {
	if len(pkgs) == 0 {
		return fmt.Errorf("no packages specified")
	}

	m.Logger.Info("installing packages: %v", pkgs)

	shellCmd := "DEBIAN_FRONTEND=noninteractive apt-get install -y"
	for _, pkg := range pkgs {
		shellCmd += " " + pkg
	}
	_, err := m.Runner.RunWithRetry(ctx, retryOpts(), "sh", "-c", shellCmd)
	if err != nil {
		return fmt.Errorf("apt-get install: %w", err)
	}

	m.Logger.Info("packages installed: %v", pkgs)
	return nil
}

// IsInstalled checks whether a package is installed by running dpkg -l.
func (m *Manager) IsInstalled(ctx context.Context, pkg string) (bool, error) {
	_, err := m.Runner.Run(ctx, "dpkg", "-l", pkg)
	if err != nil {
		return false, nil //nolint:nilerr // dpkg returns non-zero when package is not installed
	}
	return true, nil
}
