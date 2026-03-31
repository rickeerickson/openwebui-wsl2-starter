//go:build windows

// Package winfeature manages Windows optional features via powershell.exe.
package winfeature

import (
	"context"
	"fmt"
	"strings"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

const (
	featureWSL = "Microsoft-Windows-Subsystem-Linux"
	featureVMP = "VirtualMachinePlatform"
)

// Manager handles Windows optional feature operations via powershell.exe.
type Manager struct {
	Runner exec.Runner
	Logger *logging.Logger
}

// NewManager creates a Manager with the given runner and logger.
func NewManager(runner exec.Runner, logger *logging.Logger) *Manager {
	return &Manager{Runner: runner, Logger: logger}
}

// IsEnabled returns true if the named Windows optional feature is enabled.
func (m *Manager) IsEnabled(ctx context.Context, feature string) (bool, error) {
	out, err := m.Runner.Run(ctx, "powershell.exe", "-Command",
		fmt.Sprintf("(Get-WindowsOptionalFeature -FeatureName %s -Online).State", feature))
	if err != nil {
		return false, fmt.Errorf("check feature %s: %w", feature, err)
	}
	return strings.TrimSpace(out) == "Enabled", nil
}

// Enable enables a Windows optional feature without restarting.
func (m *Manager) Enable(ctx context.Context, feature string) error {
	m.Logger.Info("enabling Windows feature: %s", feature)
	_, err := m.Runner.Run(ctx, "powershell.exe", "-Command",
		fmt.Sprintf("Enable-WindowsOptionalFeature -Online -FeatureName %s -NoRestart", feature))
	if err != nil {
		return fmt.Errorf("enable feature %s: %w", feature, err)
	}
	m.Logger.Info("feature %s enabled", feature)
	return nil
}

// EnableIfNeeded checks if a feature is enabled and enables it if not.
// Returns true if the feature was newly enabled (reboot may be required).
func (m *Manager) EnableIfNeeded(ctx context.Context, feature string) (bool, error) {
	enabled, err := m.IsEnabled(ctx, feature)
	if err != nil {
		return false, err
	}
	if enabled {
		m.Logger.Info("feature %s is already enabled", feature)
		return false, nil
	}
	if err := m.Enable(ctx, feature); err != nil {
		return false, err
	}
	return true, nil
}

// EnableWSLFeatures enables both features required for WSL2:
// Microsoft-Windows-Subsystem-Linux and VirtualMachinePlatform.
// Returns true if any feature was newly enabled (reboot may be required).
func (m *Manager) EnableWSLFeatures(ctx context.Context) (bool, error) {
	m.Logger.Info("enabling required WSL features")

	wslEnabled, err := m.EnableIfNeeded(ctx, featureWSL)
	if err != nil {
		return false, err
	}

	vmpEnabled, err := m.EnableIfNeeded(ctx, featureVMP)
	if err != nil {
		return false, err
	}

	return wslEnabled || vmpEnabled, nil
}
