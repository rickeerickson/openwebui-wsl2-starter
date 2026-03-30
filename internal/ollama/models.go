//go:build linux

package ollama

import (
	"context"
	"fmt"
	"strings"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
)

// PullModel pulls a single model via `ollama pull` with retry.
func (m *Manager) PullModel(ctx context.Context, model string) error {
	m.Logger.Info("pulling model: %s", model)
	_, err := m.Runner.RunWithRetry(ctx, exec.DefaultRetryOpts(), "ollama", "pull", model)
	if err != nil {
		return fmt.Errorf("pull model %s: %w", model, err)
	}
	return nil
}

// PullModels pulls each model in the list. It skips models that are already
// installed (present in ListModels output).
func (m *Manager) PullModels(ctx context.Context, models []string) error {
	if len(models) == 0 {
		m.Logger.Info("no models to pull")
		return nil
	}

	installed, err := m.ListModels(ctx)
	if err != nil {
		// If listing fails (e.g., ollama not running), proceed with pulls anyway.
		m.Logger.Warn("could not list installed models: %v", err)
		installed = nil
	}

	installedSet := make(map[string]bool, len(installed))
	for _, name := range installed {
		installedSet[name] = true
	}

	for _, model := range models {
		if installedSet[model] {
			m.Logger.Info("model %s already installed, skipping", model)
			continue
		}
		if err := m.PullModel(ctx, model); err != nil {
			return err
		}
	}

	m.Logger.Info("model pulling completed")
	return nil
}

// ListModels runs `ollama list` and parses the output to return model names.
// The output format is a header line followed by rows where the first column
// is the model name.
func (m *Manager) ListModels(ctx context.Context) ([]string, error) {
	out, err := m.Runner.Run(ctx, "ollama", "list")
	if err != nil {
		return nil, fmt.Errorf("ollama list: %w", err)
	}
	return parseModelList(out), nil
}

// RunInteractive returns the command and arguments needed to run a model
// interactively. Actual interactive execution requires terminal wiring that
// will be handled by the cobra command layer.
func (m *Manager) RunInteractive(_ context.Context, model string) (string, []string) {
	return "ollama", []string{"run", model}
}

// parseModelList extracts model names from `ollama list` output.
// It skips the header line and extracts the first whitespace-delimited field.
func parseModelList(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= 1 {
		return nil
	}

	var models []string
	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) > 0 {
			models = append(models, fields[0])
		}
	}
	return models
}
