//go:build linux

package main

import (
	"fmt"
	"os"
	osexec "os/exec"
	"syscall"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/ollama"
	"github.com/spf13/cobra"
)

var ollamaCmd = &cobra.Command{
	Use:   "ollama",
	Short: "Manage Ollama models",
}

var ollamaPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull configured models (or specify model names as arguments)",
	RunE:  runOllamaPull,
}

var ollamaModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List installed models",
	RunE:  runOllamaModels,
}

var ollamaRunCmd = &cobra.Command{
	Use:   "run [model]",
	Short: "Run a model interactively",
	Args:  cobra.ExactArgs(1),
	RunE:  runOllamaRun,
}

func init() {
	rootCmd.AddCommand(ollamaCmd)
	ollamaCmd.AddCommand(ollamaPullCmd)
	ollamaCmd.AddCommand(ollamaModelsCmd)
	ollamaCmd.AddCommand(ollamaRunCmd)
}

func newOllamaManager() (*ollama.Manager, error) {
	logger, err := logging.NewLogger("", logging.Info)
	if err != nil {
		return nil, fmt.Errorf("creating logger: %w", err)
	}
	runner := &exec.RealRunner{Logger: logger}
	return ollama.NewManager(runner, logger), nil
}

func runOllamaPull(cmd *cobra.Command, args []string) error {
	mgr, err := newOllamaManager()
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	// If models specified as arguments, pull those. Otherwise use config.
	models := args
	if len(models) == 0 {
		cfg, err := config.Resolve(nil)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		models = cfg.Ollama.Models
	}

	return mgr.PullModels(ctx, models)
}

func runOllamaModels(cmd *cobra.Command, args []string) error {
	mgr, err := newOllamaManager()
	if err != nil {
		return err
	}

	models, err := mgr.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	if len(models) == 0 {
		fmt.Println("No models installed.")
		return nil
	}

	for _, m := range models {
		fmt.Println(m)
	}
	return nil
}

func runOllamaRun(cmd *cobra.Command, args []string) error {
	model := args[0]

	// Use syscall.Exec to replace the process with ollama run,
	// giving the user a direct interactive terminal session.
	binary, err := osexec.LookPath("ollama")
	if err != nil {
		return fmt.Errorf("ollama not found in PATH: %w", err)
	}

	return syscall.Exec(binary, []string{"ollama", "run", model}, os.Environ()) //nolint:gosec // G204: binary is from LookPath("ollama"), not user input
}
