//go:build windows

package main

import (
	"fmt"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/netsh"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/winfeature"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/wsl"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Full Windows setup: WSL2, Ubuntu, Linux setup, port proxy",
	Long: `Runs the complete Windows setup sequence matching RUNME.ps1:
  1. Enable Windows features (WSL, VirtualMachinePlatform)
  2. Install/update WSL2, set default version to 2
  3. Install Ubuntu distribution
  4. Run Linux setup inside WSL via ow setup
  5. Enable port proxy and firewall rule
  6. Print access URL`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve(nil)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger, err := logging.NewLogger("", logging.Info)
	if err != nil {
		return fmt.Errorf("creating logger: %w", err)
	}

	runner := &exec.RealRunner{Logger: logger}
	ctx := cmd.Context()

	featureMgr := winfeature.NewManager(runner, logger)
	wslMgr := wsl.NewManager(runner, logger)
	netshClient := netsh.NewClient(runner, logger)

	// Step 1: Enable Windows features.
	logger.Info("step 1/6: enabling Windows features")
	rebootNeeded, err := featureMgr.EnableWSLFeatures(ctx)
	if err != nil {
		return fmt.Errorf("enable WSL features: %w", err)
	}
	if rebootNeeded {
		fmt.Println("Windows features were enabled. Please reboot and re-run ow setup.")
		return nil
	}

	// Step 2: Install/update WSL2.
	logger.Info("step 2/6: installing and updating WSL2")
	installed, err := wslMgr.IsInstalled(ctx)
	if err != nil {
		return fmt.Errorf("check WSL: %w", err)
	}
	if !installed {
		if err := wslMgr.Install(ctx); err != nil {
			return fmt.Errorf("install WSL: %w", err)
		}
	}
	if err := wslMgr.Update(ctx); err != nil {
		return fmt.Errorf("update WSL: %w", err)
	}
	if err := wslMgr.SetDefaultVersion(ctx, 2); err != nil {
		return fmt.Errorf("set WSL version: %w", err)
	}

	// Step 3: Install Ubuntu distribution.
	logger.Info("step 3/6: installing %s distribution", cfg.WSL.Distro)
	distroInstalled, err := wslMgr.IsDistroInstalled(ctx, cfg.WSL.Distro)
	if err != nil {
		return fmt.Errorf("check distro: %w", err)
	}
	if !distroInstalled {
		if err := wslMgr.InstallDistro(ctx, cfg.WSL.Distro); err != nil {
			return fmt.Errorf("install distro: %w", err)
		}
	}

	// Step 4: Run Linux setup inside WSL.
	logger.Info("step 4/6: running Linux setup inside WSL")
	if _, err := wslMgr.RunCommand(ctx, cfg.WSL.Distro, "ow", "setup"); err != nil {
		return fmt.Errorf("wsl ow setup: %w", err)
	}

	// Step 5: Enable port proxy and firewall.
	logger.Info("step 5/6: enabling port proxy and firewall rule")
	rule := proxyRuleFromConfig(cfg)
	if err := netshClient.AddPortProxy(ctx, rule); err != nil {
		return fmt.Errorf("add port proxy: %w", err)
	}
	if err := netshClient.AddFirewallRule(ctx, firewallRuleName(cfg), cfg.Proxy.ListenPort); err != nil {
		return fmt.Errorf("add firewall rule: %w", err)
	}

	// Step 6: Print access URL.
	logger.Info("step 6/6: setup complete")
	fmt.Printf("\nOpenWebUI is available at: http://%s:%d\n",
		cfg.Proxy.ListenAddress, cfg.Proxy.ListenPort)

	return nil
}
