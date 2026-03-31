//go:build linux

package main

import (
	"fmt"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/docker"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
	"github.com/spf13/cobra"
)

// Volume mount paths inside containers. These are hardcoded constants
// that must match the upstream container expectations. Changing them
// breaks data persistence.
const (
	ollamaVolPath    = "/root/.ollama"
	openWebUIVolPath = "/app/backend/data"
)

var containersCmd = &cobra.Command{
	Use:   "containers",
	Short: "Manage Docker containers",
}

var containersUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Pull images and start Ollama and OpenWebUI containers",
	RunE:  runContainersUp,
}

var containersDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove Ollama and OpenWebUI containers",
	RunE:  runContainersDown,
}

var containersStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show running state of Ollama and OpenWebUI containers",
	RunE:  runContainersStatus,
}

func init() {
	rootCmd.AddCommand(containersCmd)
	containersCmd.AddCommand(containersUpCmd)
	containersCmd.AddCommand(containersDownCmd)
	containersCmd.AddCommand(containersStatusCmd)
}

// ollamaContainerConfig builds the Docker container config for Ollama.
func ollamaContainerConfig(cfg config.Config) docker.ContainerConfig {
	return docker.ContainerConfig{
		Name:    cfg.Ollama.Container,
		Image:   cfg.Ollama.Image,
		Tag:     cfg.Ollama.Tag,
		Volume:  cfg.Ollama.Volume,
		VolPath: ollamaVolPath,
		Env: map[string]string{
			"OLLAMA_HOST": cfg.Ollama.Host,
		},
		GPUs:    "all",
		Network: "host",
		Restart: "always",
	}
}

// openWebUIContainerConfig builds the Docker container config for OpenWebUI.
func openWebUIContainerConfig(cfg config.Config) docker.ContainerConfig {
	ollamaURL := fmt.Sprintf("http://%s:%d", cfg.Ollama.Host, cfg.Ollama.Port)
	return docker.ContainerConfig{
		Name:    cfg.OpenWebUI.Container,
		Image:   cfg.OpenWebUI.Image,
		Tag:     cfg.OpenWebUI.Tag,
		Volume:  cfg.OpenWebUI.Volume,
		VolPath: openWebUIVolPath,
		Env: map[string]string{
			"OLLAMA_BASE_URL": ollamaURL,
			"PORT":            fmt.Sprintf("%d", cfg.OpenWebUI.Port),
		},
		GPUs:    "all",
		Network: "host",
		Restart: "always",
	}
}

func newDockerClient() (*docker.Client, *logging.Logger, error) {
	logger, err := logging.NewLogger("", logging.Info)
	if err != nil {
		return nil, nil, fmt.Errorf("creating logger: %w", err)
	}
	runner := &exec.RealRunner{Logger: logger}
	return docker.NewClient(runner, logger), logger, nil
}

func runContainersUp(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve(nil)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client, logger, err := newDockerClient()
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	ollamaCfg := ollamaContainerConfig(cfg)
	openWebUICfg := openWebUIContainerConfig(cfg)

	// Pull images.
	if err := client.PullImage(ctx, ollamaCfg.Image, ollamaCfg.Tag); err != nil {
		return fmt.Errorf("pulling ollama image: %w", err)
	}
	if err := client.PullImage(ctx, openWebUICfg.Image, openWebUICfg.Tag); err != nil {
		return fmt.Errorf("pulling openwebui image: %w", err)
	}

	// Start Ollama, wait for container, then health check.
	if err := client.EnsureRunning(ctx, ollamaCfg); err != nil {
		return fmt.Errorf("starting ollama: %w", err)
	}
	logger.Info("waiting for Ollama health check on port %d", cfg.Ollama.Port)
	if err := client.WaitForHTTP(ctx, cfg.Ollama.Host, cfg.Ollama.Port, "/", exec.DefaultRetryOpts()); err != nil {
		return fmt.Errorf("ollama health check: %w", err)
	}

	// Start OpenWebUI, wait for container, then health check.
	if err := client.EnsureRunning(ctx, openWebUICfg); err != nil {
		return fmt.Errorf("starting openwebui: %w", err)
	}
	logger.Info("waiting for OpenWebUI health check on port %d", cfg.OpenWebUI.Port)
	if err := client.WaitForHTTP(ctx, cfg.OpenWebUI.Host, cfg.OpenWebUI.Port, "/", exec.DefaultRetryOpts()); err != nil {
		return fmt.Errorf("openwebui health check: %w", err)
	}

	logger.Info("all containers running")
	return nil
}

func runContainersDown(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve(nil)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client, _, err := newDockerClient()
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	// Stop OpenWebUI first, then Ollama.
	if err := client.StopAndRemove(ctx, cfg.OpenWebUI.Container); err != nil {
		return fmt.Errorf("stopping openwebui: %w", err)
	}
	if err := client.StopAndRemove(ctx, cfg.Ollama.Container); err != nil {
		return fmt.Errorf("stopping ollama: %w", err)
	}

	return nil
}

func runContainersStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve(nil)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client, _, err := newDockerClient()
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	for _, name := range []string{cfg.Ollama.Container, cfg.OpenWebUI.Container} {
		running, err := client.ContainerIsRunning(ctx, name)
		if err != nil {
			return fmt.Errorf("checking %s: %w", name, err)
		}
		status := "stopped"
		if running {
			status = "running"
		}
		fmt.Printf("%-20s %s\n", name, status)
	}

	return nil
}
