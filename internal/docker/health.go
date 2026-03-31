//go:build linux

package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
)

// CheckHTTP runs a single HTTP health check using curl via the runner.
// It succeeds if curl exits 0, meaning the endpoint returned HTTP 2xx.
func (c *Client) CheckHTTP(ctx context.Context, host string, port int, path string) error {
	url := fmt.Sprintf("http://%s:%d%s", host, port, path)
	_, err := c.Runner.Run(ctx, "curl", "-sf", url)
	return err
}

// WaitForHTTP retries CheckHTTP with Fibonacci backoff until it succeeds
// or the retry attempts are exhausted.
func (c *Client) WaitForHTTP(ctx context.Context, host string, port int, path string, opts exec.RetryOpts) error {
	if opts.MaxAttempts < 1 {
		return fmt.Errorf("MaxAttempts must be >= 1, got %d", opts.MaxAttempts)
	}

	url := fmt.Sprintf("http://%s:%d%s", host, port, path)
	c.Logger.Info("waiting for %s (max %d attempts)", url, opts.MaxAttempts)

	a := opts.InitialA
	b := opts.InitialB

	var lastErr error
	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		lastErr = c.CheckHTTP(ctx, host, port, path)
		if lastErr == nil {
			c.Logger.Info("health check passed: %s", url)
			return nil
		}

		if attempt == opts.MaxAttempts {
			break
		}

		var delay time.Duration
		delay, a, b = exec.NextDelay(a, b)
		c.Logger.Warn("health check %s failed, retry %d/%d in %s: %s",
			url, attempt, opts.MaxAttempts, delay, lastErr)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled waiting for %s: %w", url, ctx.Err())
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("health check %s failed after %d attempts: %w", url, opts.MaxAttempts, lastErr)
}
