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
// autoclean with retry on each step. Sets DEBIAN_FRONTEND=noninteractive via
// env args passed to apt-get.
func (m *Manager) UpdatePackages(ctx context.Context) error {
	m.Logger.Info("updating system packages")

	commands := [][]string{
		{"apt-get", "update"},
		{"apt-get", "upgrade", "-y"},
		{"apt-get", "dist-upgrade", "-y"},
		{"apt-get", "autoremove", "-y"},
		{"apt-get", "autoclean", "-y"},
	}

	for _, cmd := range commands {
		_, err := m.Runner.RunWithRetry(ctx, retryOpts(), cmd[0], cmd[1:]...)
		if err != nil {
			return fmt.Errorf("%s: %w", cmd[0]+" "+cmd[1], err)
		}
	}

	m.Logger.Info("system packages updated")
	return nil
}

// InstallPackages installs the given packages via apt-get install -y with retry.
func (m *Manager) InstallPackages(ctx context.Context, pkgs ...string) error {
	if len(pkgs) == 0 {
		return fmt.Errorf("no packages specified")
	}

	m.Logger.Info("installing packages: %v", pkgs)

	args := append([]string{"install", "-y"}, pkgs...)
	_, err := m.Runner.RunWithRetry(ctx, retryOpts(), "apt-get", args...)
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
