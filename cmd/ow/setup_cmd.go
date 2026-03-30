//go:build linux

package main

import (
	"fmt"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/apt"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/docker"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/nvidia"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/ollama"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Full Linux setup: packages, Docker, NVIDIA, Ollama, and OpenWebUI",
	Long: `Runs the complete setup sequence matching update_open-webui.sh:
  1. Update system packages
  2. Setup Docker keyring and install Docker
  3. Install NVIDIA Container Toolkit
  4. Configure Docker for NVIDIA and add user to docker group
  5. Install Ollama
  6. Verify Docker environment
  7. Start Ollama container and pull models
  8. Start OpenWebUI container`,
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

	aptMgr := apt.NewManager(runner, logger)
	nvInstaller := nvidia.NewInstaller(runner, logger)
	ollamaMgr := ollama.NewManager(runner, logger)
	dockerClient := docker.NewClient(runner, logger)

	// Step 1: Update system packages.
	logger.Info("step 1/11: updating system packages")
	if err := aptMgr.UpdatePackages(ctx); err != nil {
		return fmt.Errorf("update packages: %w", err)
	}

	// Step 2: Setup Docker keyring.
	logger.Info("step 2/11: setting up Docker keyring")
	if err := aptMgr.SetupDockerKeyring(ctx); err != nil {
		return fmt.Errorf("setup docker keyring: %w", err)
	}

	// Step 3: Install NVIDIA Container Toolkit.
	logger.Info("step 3/11: installing NVIDIA Container Toolkit")
	if err := nvInstaller.Install(ctx); err != nil {
		return fmt.Errorf("install nvidia toolkit: %w", err)
	}

	// Step 4: Install Docker.
	logger.Info("step 4/11: installing Docker")
	if err := aptMgr.InstallDocker(ctx); err != nil {
		return fmt.Errorf("install docker: %w", err)
	}

	// Step 5: Configure Docker NVIDIA runtime and add user to docker group.
	logger.Info("step 5/11: configuring Docker for NVIDIA")
	if err := aptMgr.ConfigureDockerNvidia(ctx); err != nil {
		return fmt.Errorf("configure docker nvidia: %w", err)
	}
	if err := aptMgr.AddUserToDockerGroup(ctx); err != nil {
		return fmt.Errorf("add user to docker group: %w", err)
	}

	// Step 6: Install Ollama.
	logger.Info("step 6/11: installing Ollama")
	if err := ollamaMgr.Install(ctx); err != nil {
		return fmt.Errorf("install ollama: %w", err)
	}

	// Step 7: Verify Docker environment.
	logger.Info("step 7/11: verifying Docker environment")
	if _, err := runner.Run(ctx, "docker", "run", "hello-world"); err != nil {
		return fmt.Errorf("docker environment verification failed: %w", err)
	}

	// Step 8: Pull Ollama image and start container.
	logger.Info("step 8/11: starting Ollama container")
	ollamaCfg := ollamaContainerConfig(cfg)
	if err := dockerClient.PullImage(ctx, ollamaCfg.Image, ollamaCfg.Tag); err != nil {
		return fmt.Errorf("pull ollama image: %w", err)
	}
	if err := dockerClient.EnsureRunning(ctx, ollamaCfg); err != nil {
		return fmt.Errorf("start ollama container: %w", err)
	}

	// Step 9: Health check Ollama.
	logger.Info("step 9/11: verifying Ollama health")
	if err := dockerClient.WaitForHTTP(ctx, cfg.Ollama.Host, cfg.Ollama.Port, "/", exec.DefaultRetryOpts()); err != nil {
		return fmt.Errorf("ollama health check: %w", err)
	}

	// Step 10: Pull configured models.
	logger.Info("step 10/11: pulling Ollama models")
	if err := ollamaMgr.PullModels(ctx, cfg.Ollama.Models); err != nil {
		return fmt.Errorf("pull models: %w", err)
	}

	// Step 11: Pull OpenWebUI image and start container.
	logger.Info("step 11/11: starting OpenWebUI container")
	openWebUICfg := openWebUIContainerConfig(cfg)
	if err := dockerClient.PullImage(ctx, openWebUICfg.Image, openWebUICfg.Tag); err != nil {
		return fmt.Errorf("pull openwebui image: %w", err)
	}
	if err := dockerClient.EnsureRunning(ctx, openWebUICfg); err != nil {
		return fmt.Errorf("start openwebui container: %w", err)
	}
	if err := dockerClient.WaitForHTTP(ctx, cfg.OpenWebUI.Host, cfg.OpenWebUI.Port, "/", exec.DefaultRetryOpts()); err != nil {
		return fmt.Errorf("openwebui health check: %w", err)
	}

	logger.Info("setup complete")
	return nil
}
