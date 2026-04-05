//go:build windows

package main

import (
	"fmt"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/wsl"
	"github.com/spf13/cobra"
)

var wslCmd = &cobra.Command{
	Use:   "wsl",
	Short: "Manage WSL2 and distributions",
}

var wslInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install WSL2 and the configured distribution",
	RunE:  runWslInstall,
}

var wslRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Unregister the configured WSL distribution",
	RunE:  runWslRemove,
}

var wslStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Shut down all running WSL distributions",
	RunE:  runWslStop,
}

func init() {
	rootCmd.AddCommand(wslCmd)
	wslCmd.AddCommand(wslInstallCmd)
	wslCmd.AddCommand(wslRemoveCmd)
	wslCmd.AddCommand(wslStopCmd)
}

func newWslManager() (*wsl.Manager, error) {
	logger, err := logging.NewLogger("", logging.Info)
	if err != nil {
		return nil, fmt.Errorf("creating logger: %w", err)
	}
	runner := &exec.RealRunner{Logger: logger}
	return wsl.NewManager(runner, logger), nil
}

func runWslInstall(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve(nil)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	mgr, err := newWslManager()
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	// Install WSL if needed.
	installed, err := mgr.IsInstalled(ctx)
	if err != nil {
		return fmt.Errorf("checking WSL: %w", err)
	}
	if !installed {
		if err := mgr.Install(ctx); err != nil {
			return err
		}
	}

	// Update WSL.
	if err := mgr.Update(ctx); err != nil {
		return err
	}

	// Set default version to 2.
	if err := mgr.SetDefaultVersion(ctx, 2); err != nil {
		return err
	}

	// Install distro if needed.
	distro := cfg.WSL.Distro
	distroInstalled, err := mgr.IsDistroInstalled(ctx, distro)
	if err != nil {
		return fmt.Errorf("checking distro: %w", err)
	}
	if !distroInstalled {
		if err := mgr.InstallDistro(ctx, distro); err != nil {
			return err
		}
	}

	return nil
}

func runWslRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve(nil)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	mgr, err := newWslManager()
	if err != nil {
		return err
	}

	return mgr.RemoveDistro(cmd.Context(), cfg.WSL.Distro)
}

func runWslStop(cmd *cobra.Command, args []string) error {
	mgr, err := newWslManager()
	if err != nil {
		return err
	}

	return mgr.Stop(cmd.Context())
}
