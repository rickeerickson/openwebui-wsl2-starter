package exec

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// Runner executes system commands.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
	RunWithRetry(ctx context.Context, opts RetryOpts, name string, args ...string) (string, error)
}

// RealRunner executes commands via os/exec.
type RealRunner struct {
	Logger      *logging.Logger
	AllowedBins map[string]bool // nil means use default allowlist
}

// Run executes a single command after validating it against the allowlist.
func (r *RealRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty command name")
	}

	if !r.allowedBins()[name] {
		return "", fmt.Errorf("binary %q is not in the allowlist", name)
	}

	r.Logger.Info("exec: %s %v", name, args)

	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // G204: name is checked against allowlist above
	out, err := cmd.CombinedOutput()
	output := string(out)

	r.Logger.Debug1("output: %s", output)

	if err != nil {
		return output, fmt.Errorf("command %q failed: %s: %w", name, output, err)
	}
	return output, nil
}

// RunWithRetry calls Run in a loop with Fibonacci backoff per RetryOpts.
func (r *RealRunner) RunWithRetry(ctx context.Context, opts RetryOpts, name string, args ...string) (string, error) {
	a := opts.InitialA
	b := opts.InitialB

	var lastOut string
	var lastErr error

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		lastOut, lastErr = r.Run(ctx, name, args...)
		if lastErr == nil {
			return lastOut, nil
		}

		if attempt == opts.MaxAttempts {
			break
		}

		var delay time.Duration
		delay, a, b = NextDelay(a, b)

		r.Logger.Warn("retry %d/%d in %s: %s", attempt, opts.MaxAttempts, delay, lastErr)

		select {
		case <-ctx.Done():
			return lastOut, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		case <-time.After(delay):
		}
	}

	return lastOut, fmt.Errorf("after %d attempts: %w", opts.MaxAttempts, lastErr)
}
