//go:build linux

package main

import (
	"context"
	"fmt"
	"os/user"
	"strings"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
	"github.com/spf13/cobra"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Run diagnostic checks on the system, Docker, and containers",
	RunE:  runDiagnose,
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
}

// diagSection prints a section header and runs a command, printing its output.
// Errors are printed but do not stop the diagnostic flow.
func diagSection(ctx context.Context, runner exec.Runner, header string, bin string, args ...string) {
	fmt.Printf("=== %s ===\n", header)
	out, err := runner.Run(ctx, bin, args...)
	if err != nil {
		fmt.Printf("  (error: %v)\n", err)
	} else {
		fmt.Println(strings.TrimRight(out, "\n"))
	}
	fmt.Println()
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

	// Basic system info.
	fmt.Println("=== Basic System Info ===")
	if u, err := user.Current(); err == nil {
		fmt.Printf("User: %s\n", u.Username)
	}
	out, err := runner.Run(ctx, "cat", "/etc/os-release")
	if err == nil {
		fmt.Println(strings.TrimRight(out, "\n"))
	}
	fmt.Println()

	// Network interfaces.
	diagSection(ctx, runner, "Network Interfaces & IPs", "ip", "addr", "show")

	// Listening ports.
	diagSection(ctx, runner, "Listening Ports (TCP)", "lsof", "-i", "-P", "-n")

	// NVIDIA status.
	diagSection(ctx, runner, "NVIDIA GPU Status", "nvidia-smi")

	// Docker diagnostics.
	diagSection(ctx, runner, "Docker Version", "docker", "--version")
	diagSection(ctx, runner, "Docker Containers", "docker", "ps")
	diagSection(ctx, runner, "Docker Images", "docker", "images")

	// Test port connectivity.
	fmt.Printf("=== Test Ollama Port (%d) ===\n", cfg.Ollama.Port)
	url := fmt.Sprintf("http://%s:%d", cfg.Ollama.Host, cfg.Ollama.Port)
	if _, err := runner.Run(ctx, "curl", "-sf", url); err != nil {
		fmt.Printf("  HTTP check failed: %v\n", err)
	} else {
		fmt.Printf("  HTTP check succeeded on port %d\n", cfg.Ollama.Port)
	}
	fmt.Println()

	fmt.Printf("=== Test OpenWebUI Port (%d) ===\n", cfg.OpenWebUI.Port)
	url = fmt.Sprintf("http://%s:%d", cfg.OpenWebUI.Host, cfg.OpenWebUI.Port)
	if _, err := runner.Run(ctx, "curl", "-sf", url); err != nil {
		fmt.Printf("  HTTP check failed: %v\n", err)
	} else {
		fmt.Printf("  HTTP check succeeded on port %d\n", cfg.OpenWebUI.Port)
	}
	fmt.Println()

	// Container logs (filtered for errors/warnings).
	for _, name := range []string{cfg.Ollama.Container, cfg.OpenWebUI.Container} {
		fmt.Printf("=== %s Container Logs ===\n", name)
		out, err := runner.Run(ctx, "docker", "logs", "--tail", "50", name)
		if err != nil {
			fmt.Printf("  (error: %v)\n", err)
		} else {
			for _, line := range strings.Split(out, "\n") {
				lower := strings.ToLower(line)
				if strings.Contains(lower, "error") ||
					strings.Contains(lower, "warn") ||
					strings.Contains(lower, "listen") {
					fmt.Println(line)
				}
			}
		}
		fmt.Println()
	}

	// Routing and connectivity.
	diagSection(ctx, runner, "Default Routes", "ip", "route", "show", "default")
	diagSection(ctx, runner, "External Connectivity", "ping", "-c", "4", "google.com")

	fmt.Println("=== END DIAGNOSTICS ===")
	return nil
}
