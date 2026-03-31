//go:build linux

package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// Client manages Docker container lifecycle by shelling out to the docker CLI.
type Client struct {
	Runner exec.Runner
	Logger *logging.Logger
}

// NewClient creates a Client with the given runner and logger.
func NewClient(runner exec.Runner, logger *logging.Logger) *Client {
	return &Client{Runner: runner, Logger: logger}
}

// ContainerExists returns true if a container with the given name exists
// (running or stopped). It runs `docker inspect name`.
func (c *Client) ContainerExists(ctx context.Context, name string) (bool, error) {
	output, err := c.Runner.Run(ctx, "docker", "inspect", name)
	if err != nil {
		// docker inspect exits non-zero with "No such object" when the
		// container does not exist. Any other error is a real failure.
		if strings.Contains(output, "No such object") ||
			strings.Contains(output, "No such container") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ContainerIsRunning returns true if a container with the given name exists
// and is currently running.
func (c *Client) ContainerIsRunning(ctx context.Context, name string) (bool, error) {
	out, err := c.Runner.Run(ctx, "docker", "inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		// docker inspect exits non-zero with "No such object" when the
		// container does not exist. Any other error is a real failure.
		if strings.Contains(out, "No such object") ||
			strings.Contains(out, "No such container") {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(out) == "true", nil
}

// StopContainer stops a running container. If the container is not running,
// this is a no-op.
func (c *Client) StopContainer(ctx context.Context, name string) error {
	running, err := c.ContainerIsRunning(ctx, name)
	if err != nil {
		return fmt.Errorf("check running state: %w", err)
	}
	if !running {
		c.Logger.Info("container %s is not running, skipping stop", name)
		return nil
	}
	c.Logger.Info("stopping container %s", name)
	_, err = c.Runner.Run(ctx, "docker", "stop", name)
	return err
}

// RemoveContainer removes a container. If the container does not exist,
// this is a no-op.
func (c *Client) RemoveContainer(ctx context.Context, name string) error {
	exists, err := c.ContainerExists(ctx, name)
	if err != nil {
		return fmt.Errorf("check existence: %w", err)
	}
	if !exists {
		c.Logger.Info("container %s does not exist, skipping remove", name)
		return nil
	}
	c.Logger.Info("removing container %s", name)
	_, err = c.Runner.Run(ctx, "docker", "rm", name)
	return err
}

// StopAndRemove stops and removes a container. Both operations are idempotent.
func (c *Client) StopAndRemove(ctx context.Context, name string) error {
	if err := c.StopContainer(ctx, name); err != nil {
		return fmt.Errorf("stop %s: %w", name, err)
	}
	if err := c.RemoveContainer(ctx, name); err != nil {
		return fmt.Errorf("remove %s: %w", name, err)
	}
	return nil
}

// PullImage pulls a Docker image using the runner's retry mechanism.
func (c *Client) PullImage(ctx context.Context, image, tag string) error {
	ref := image + ":" + tag
	c.Logger.Info("pulling image %s", ref)
	_, err := c.Runner.RunWithRetry(ctx, exec.DefaultRetryOpts(), "docker", "pull", ref)
	return err
}

// RunContainer starts a new container from the given config.
func (c *Client) RunContainer(ctx context.Context, cfg ContainerConfig) error {
	c.Logger.Info("running container %s from %s", cfg.Name, cfg.ImageRef())
	args := cfg.RunArgs()
	_, err := c.Runner.Run(ctx, "docker", args...)
	return err
}

// EnsureRunning stops and removes any existing container, runs a new one,
// and polls until it reports as running.
func (c *Client) EnsureRunning(ctx context.Context, cfg ContainerConfig) error {
	if err := c.StopAndRemove(ctx, cfg.Name); err != nil {
		return err
	}
	if err := c.RunContainer(ctx, cfg); err != nil {
		return err
	}

	// Poll until the container is running.
	opts := exec.DefaultRetryOpts()
	a := opts.InitialA
	b := opts.InitialB

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		running, err := c.ContainerIsRunning(ctx, cfg.Name)
		if err != nil {
			return fmt.Errorf("poll running state: %w", err)
		}
		if running {
			c.Logger.Info("container %s is running", cfg.Name)
			return nil
		}

		if attempt == opts.MaxAttempts {
			break
		}

		var delay time.Duration
		delay, a, b = exec.NextDelay(a, b)
		c.Logger.Warn("container %s not yet running, retry %d/%d in %s",
			cfg.Name, attempt, opts.MaxAttempts, delay)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled waiting for %s: %w", cfg.Name, ctx.Err())
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("container %s not running after %d attempts", cfg.Name, opts.MaxAttempts)
}
