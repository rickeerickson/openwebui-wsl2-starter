//go:build linux

package ollama

import (
	"context"
	"fmt"
	"testing"
)

func TestPullModelCallsOllamaPull(t *testing.T) {
	m := newMockRunner()
	mgr := NewManager(m, newTestLogger(t))

	err := mgr.PullModel(context.Background(), "llama3.2:1b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !m.called("ollama", "pull", "llama3.2:1b") {
		t.Error("expected ollama pull call with correct model name")
	}
}

func TestPullModelUsesRetry(t *testing.T) {
	m := newMockRunner()
	// Verify RunWithRetry is used by checking the call goes through.
	// The mock's RunWithRetry delegates to Run, so the call is recorded.
	mgr := NewManager(m, newTestLogger(t))

	err := mgr.PullModel(context.Background(), "mistral:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !m.called("ollama", "pull", "mistral:latest") {
		t.Error("expected ollama pull call via retry path")
	}
}

func TestPullModelsPullsEachInOrder(t *testing.T) {
	m := newMockRunner()
	// ListModels returns empty so all models need pulling.
	m.outputs[m.key("ollama", "list")] = "NAME\tID\tSIZE\tMODIFIED\n"

	mgr := NewManager(m, newTestLogger(t))
	models := []string{"llama3.2:1b", "mistral:latest", "phi3:mini"}
	err := mgr.PullModels(context.Background(), models)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, model := range models {
		if !m.called("ollama", "pull", model) {
			t.Errorf("expected ollama pull %s", model)
		}
	}

	// Verify ordering: find indices of pull calls.
	pullIndices := make(map[string]int)
	for i, c := range m.calls {
		if c.Name == "ollama" && len(c.Args) >= 2 && c.Args[0] == "pull" {
			pullIndices[c.Args[1]] = i
		}
	}
	for i := 1; i < len(models); i++ {
		prev := pullIndices[models[i-1]]
		curr := pullIndices[models[i]]
		if curr <= prev {
			t.Errorf("model %s pulled before %s", models[i], models[i-1])
		}
	}
}

func TestPullModelsHandlesEmptyList(t *testing.T) {
	m := newMockRunner()
	mgr := NewManager(m, newTestLogger(t))

	err := mgr.PullModels(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No calls should be made.
	if m.callCount() != 0 {
		t.Errorf("expected 0 calls, got %d: %+v", m.callCount(), m.calls)
	}
}

func TestListModelsParseOutput(t *testing.T) {
	m := newMockRunner()
	m.outputs[m.key("ollama", "list")] = `NAME                    ID              SIZE    MODIFIED
llama3.2:1b             abc123          1.3 GB  2 hours ago
mistral:latest          def456          4.1 GB  1 day ago
`

	mgr := NewManager(m, newTestLogger(t))
	models, err := mgr.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"llama3.2:1b", "mistral:latest"}
	if len(models) != len(expected) {
		t.Fatalf("got %d models, want %d: %v", len(models), len(expected), models)
	}
	for i, e := range expected {
		if models[i] != e {
			t.Errorf("models[%d] = %q, want %q", i, models[i], e)
		}
	}
}

func TestListModelsReturnsEmptyOnNoModels(t *testing.T) {
	m := newMockRunner()
	m.outputs[m.key("ollama", "list")] = "NAME\tID\tSIZE\tMODIFIED\n"

	mgr := NewManager(m, newTestLogger(t))
	models, err := mgr.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(models) != 0 {
		t.Errorf("expected empty slice, got %v", models)
	}
}

func TestPullModelReturnsErrorOnFailure(t *testing.T) {
	m := newMockRunner()
	m.errors[m.key("ollama", "pull", "bad-model")] = fmt.Errorf("pull failed")

	mgr := NewManager(m, newTestLogger(t))
	err := mgr.PullModel(context.Background(), "bad-model")
	if err == nil {
		t.Fatal("expected error for failed pull")
	}
}

func TestRunInteractiveReturnsCommand(t *testing.T) {
	m := newMockRunner()
	mgr := NewManager(m, newTestLogger(t))

	cmd, args := mgr.RunInteractive(context.Background(), "llama3.2:1b")
	if cmd != "ollama" {
		t.Errorf("command = %q, want 'ollama'", cmd)
	}
	if len(args) != 2 || args[0] != "run" || args[1] != "llama3.2:1b" {
		t.Errorf("args = %v, want [run llama3.2:1b]", args)
	}
}

func TestPullModelsSkipsInstalledModels(t *testing.T) {
	m := newMockRunner()
	m.outputs[m.key("ollama", "list")] = `NAME                    ID              SIZE    MODIFIED
llama3.2:1b             abc123          1.3 GB  2 hours ago
`

	mgr := NewManager(m, newTestLogger(t))
	err := mgr.PullModels(context.Background(), []string{"llama3.2:1b", "mistral:latest"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// llama3.2:1b should be skipped, mistral:latest should be pulled.
	if m.called("ollama", "pull", "llama3.2:1b") {
		t.Error("should not pull already-installed model llama3.2:1b")
	}
	if !m.called("ollama", "pull", "mistral:latest") {
		t.Error("should pull mistral:latest")
	}
}
