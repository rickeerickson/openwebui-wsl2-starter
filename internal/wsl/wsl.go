//go:build windows

// Package wsl manages WSL2 installation, updates, and lifecycle via wsl.exe.
package wsl

import (
	"context"
	"fmt"
	"strings"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// Manager handles WSL2 lifecycle operations by shelling out to wsl.exe.
type Manager struct {
	Runner exec.Runner
	Logger *logging.Logger
}

// NewManager creates a Manager with the given runner and logger.
func NewManager(runner exec.Runner, logger *logging.Logger) *Manager {
	return &Manager{Runner: runner, Logger: logger}
}

// IsInstalled returns true if WSL is installed and functional.
// It runs `wsl.exe --status` and checks for a zero exit code.
func (m *Manager) IsInstalled(ctx context.Context) (bool, error) {
	_, err := m.Runner.Run(ctx, "wsl.exe", "--status")
	if err != nil {
		return false, nil //nolint:nilerr // wsl --status exits non-zero when not installed
	}
	return true, nil
}

// Install installs WSL without a distribution.
// The caller should install a distro separately.
func (m *Manager) Install(ctx context.Context) error {
	m.Logger.Info("installing WSL")
	_, err := m.Runner.RunWithRetry(ctx, exec.DefaultRetryOpts(),
		"wsl.exe", "--install", "--no-distribution")
	if err != nil {
		return fmt.Errorf("wsl install: %w", err)
	}
	m.Logger.Info("WSL installed")
	return nil
}

// Update runs `wsl --update` to update the WSL kernel.
func (m *Manager) Update(ctx context.Context) error {
	m.Logger.Info("updating WSL")
	_, err := m.Runner.RunWithRetry(ctx, exec.DefaultRetryOpts(),
		"wsl.exe", "--update")
	if err != nil {
		return fmt.Errorf("wsl update: %w", err)
	}
	m.Logger.Info("WSL updated")
	return nil
}

// SetDefaultVersion sets the default WSL version (1 or 2).
func (m *Manager) SetDefaultVersion(ctx context.Context, version int) error {
	m.Logger.Info("setting WSL default version to %d", version)
	_, err := m.Runner.Run(ctx, "wsl.exe", "--set-default-version",
		fmt.Sprintf("%d", version))
	if err != nil {
		return fmt.Errorf("wsl set-default-version: %w", err)
	}
	return nil
}

// Stop shuts down all running WSL distributions.
func (m *Manager) Stop(ctx context.Context) error {
	m.Logger.Info("shutting down WSL")
	_, err := m.Runner.Run(ctx, "wsl.exe", "--shutdown")
	if err != nil {
		return fmt.Errorf("wsl shutdown: %w", err)
	}
	m.Logger.Info("WSL shut down")
	return nil
}

// IsDistroInstalled returns true if the named distribution is registered.
// It parses `wsl --list --verbose` output.
func (m *Manager) IsDistroInstalled(ctx context.Context, name string) (bool, error) {
	out, err := m.Runner.Run(ctx, "wsl.exe", "--list", "--verbose")
	if err != nil {
		// WSL not installed or no distros: treat as not installed.
		return false, nil //nolint:nilerr // expected when WSL has no distros
	}
	return containsDistro(out, name), nil
}

// InstallDistro installs a WSL distribution. This is interactive: the user
// must set a username and password. The caller should wire stdin/stdout
// for terminal passthrough.
func (m *Manager) InstallDistro(ctx context.Context, name string) error {
	m.Logger.Info("installing WSL distribution: %s", name)
	_, err := m.Runner.Run(ctx, "wsl.exe", "--install", "-d", name)
	if err != nil {
		return fmt.Errorf("wsl install distro %s: %w", name, err)
	}

	// Set as default distribution.
	_, err = m.Runner.Run(ctx, "wsl.exe", "--setdefault", name)
	if err != nil {
		return fmt.Errorf("wsl setdefault %s: %w", name, err)
	}

	m.Logger.Info("distribution %s installed and set as default", name)
	return nil
}

// RemoveDistro unregisters a WSL distribution, deleting its filesystem.
func (m *Manager) RemoveDistro(ctx context.Context, name string) error {
	m.Logger.Info("removing WSL distribution: %s", name)
	_, err := m.Runner.Run(ctx, "wsl.exe", "--unregister", name)
	if err != nil {
		return fmt.Errorf("wsl unregister %s: %w", name, err)
	}
	m.Logger.Info("distribution %s removed", name)
	return nil
}

// RunCommand executes a command inside a WSL distribution.
func (m *Manager) RunCommand(ctx context.Context, distro string, cmd string, args ...string) (string, error) {
	wslArgs := []string{"-d", distro, "--"}
	wslArgs = append(wslArgs, cmd)
	wslArgs = append(wslArgs, args...)
	return m.Runner.Run(ctx, "wsl.exe", wslArgs...)
}

// containsDistro checks if a distro name appears in wsl --list --verbose output.
// The output format has a header line followed by lines like:
//
//	* Ubuntu    Running  2
//	  Debian    Stopped  2
func containsDistro(output, name string) bool {
	for _, line := range strings.Split(output, "\n") {
		// Strip leading *, spaces, and null bytes (wsl outputs UTF-16).
		cleaned := strings.TrimLeft(line, "* \t\x00")
		fields := strings.Fields(cleaned)
		if len(fields) >= 1 && strings.EqualFold(fields[0], name) {
			return true
		}
	}
	return false
}
