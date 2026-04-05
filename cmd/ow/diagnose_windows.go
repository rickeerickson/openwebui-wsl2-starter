//go:build windows

package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/netsh"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/wsl"
	"github.com/spf13/cobra"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Run diagnostic checks on WSL, Docker, port proxy, and connectivity",
	RunE:  runDiagnose,
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
}

func runDiagnose(cmd *cobra.Command, args []string) error {
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

	wslMgr := wsl.NewManager(runner, logger)
	netshClient := netsh.NewClient(runner, logger)

	// System info.
	fmt.Println("=== System Info ===")
	fmt.Printf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	out, err := runner.Run(ctx, "powershell.exe", "-Command", "Get-ComputerInfo | Select-Object WindowsVersion, OsBuildNumber, CsName | Format-List")
	if err == nil {
		fmt.Println(strings.TrimRight(out, "\n"))
	}
	fmt.Println()

	// WSL status.
	fmt.Println("=== WSL Status ===")
	installed, _ := wslMgr.IsInstalled(ctx)
	if installed {
		fmt.Println("WSL: installed")
	} else {
		fmt.Println("WSL: not installed")
	}
	out, err = runner.Run(ctx, "wsl.exe", "--list", "--verbose")
	if err == nil {
		fmt.Println(strings.TrimRight(out, "\n\x00"))
	} else {
		fmt.Printf("  (error: %v)\n", err)
	}
	fmt.Println()

	// Distro check.
	fmt.Printf("=== Distro: %s ===\n", cfg.WSL.Distro)
	distroInstalled, _ := wslMgr.IsDistroInstalled(ctx, cfg.WSL.Distro)
	if distroInstalled {
		fmt.Printf("%s: installed\n", cfg.WSL.Distro)
	} else {
		fmt.Printf("%s: not installed\n", cfg.WSL.Distro)
	}
	fmt.Println()

	// Port proxy rules.
	fmt.Println("=== Port Proxy Rules ===")
	rules, err := netshClient.ListPortProxy(ctx)
	if err != nil {
		fmt.Printf("  (error: %v)\n", err)
	} else if len(rules) == 0 {
		fmt.Println("  No rules found.")
	} else {
		for _, r := range rules {
			fmt.Printf("  %s:%d -> %s:%d\n",
				r.ListenAddress, r.ListenPort,
				r.ConnectAddress, r.ConnectPort)
		}
	}
	fmt.Println()

	// Firewall rule check.
	fmt.Println("=== Firewall Rules ===")
	fwName := firewallRuleName(cfg)
	fwExists, err := netshClient.FirewallRuleExists(ctx, fwName)
	if err != nil {
		fmt.Printf("  (error checking %s: %v)\n", fwName, err)
	} else if fwExists {
		fmt.Printf("  %s: exists\n", fwName)
	} else {
		fmt.Printf("  %s: missing\n", fwName)
	}
	fmt.Println()

	// TCP connectivity test to OpenWebUI port.
	fmt.Printf("=== Test OpenWebUI Port (%d) ===\n", cfg.OpenWebUI.Port)
	out, err = runner.Run(ctx, "powershell.exe", "-Command",
		fmt.Sprintf("Test-NetConnection -ComputerName %s -Port %d | Select-Object TcpTestSucceeded | Format-List",
			cfg.OpenWebUI.Host, cfg.OpenWebUI.Port))
	if err == nil {
		fmt.Println(strings.TrimRight(out, "\n"))
	} else {
		fmt.Printf("  (error: %v)\n", err)
	}
	fmt.Println()

	// TCP connectivity test to Ollama port.
	fmt.Printf("=== Test Ollama Port (%d) ===\n", cfg.Ollama.Port)
	out, err = runner.Run(ctx, "powershell.exe", "-Command",
		fmt.Sprintf("Test-NetConnection -ComputerName %s -Port %d | Select-Object TcpTestSucceeded | Format-List",
			cfg.Ollama.Host, cfg.Ollama.Port))
	if err == nil {
		fmt.Println(strings.TrimRight(out, "\n"))
	} else {
		fmt.Printf("  (error: %v)\n", err)
	}
	fmt.Println()

	// Docker status inside WSL (if distro is installed).
	if distroInstalled {
		fmt.Println("=== Docker Status (WSL) ===")
		out, err = wslMgr.RunCommand(ctx, cfg.WSL.Distro, "docker", "ps")
		if err != nil {
			fmt.Printf("  (error: %v)\n", err)
		} else {
			fmt.Println(strings.TrimRight(out, "\n"))
		}
		fmt.Println()
	}

	fmt.Println("=== END DIAGNOSTICS ===")
	return nil
}
