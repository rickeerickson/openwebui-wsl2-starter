//go:build linux

package docker

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// mockCall represents one expected call to the mock runner.
type mockCall struct {
	wantName string
	wantArgs []string // nil means don't check args
	output   string
	err      error
}

// mockRunner replays predefined responses for sequential calls.
type mockRunner struct {
	calls   []mockCall
	current int
	t       *testing.T
}

func (m *mockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	m.t.Helper()
	if m.current >= len(m.calls) {
		m.t.Fatalf("unexpected call #%d: %s %v", m.current, name, args)
	}
	c := m.calls[m.current]
	m.current++

	if c.wantName != "" && c.wantName != name {
		m.t.Errorf("call #%d: name = %q, want %q", m.current-1, name, c.wantName)
	}
	if c.wantArgs != nil {
		got := strings.Join(args, " ")
		want := strings.Join(c.wantArgs, " ")
		if got != want {
			m.t.Errorf("call #%d: args = %q, want %q", m.current-1, got, want)
		}
	}
	return c.output, c.err
}

func (m *mockRunner) RunWithRetry(ctx context.Context, opts exec.RetryOpts, name string, args ...string) (string, error) {
	// For test purposes, just delegate to Run (retry logic is tested in exec).
	return m.Run(ctx, name, args...)
}

func (m *mockRunner) verify() {
	m.t.Helper()
	if m.current != len(m.calls) {
		m.t.Errorf("expected %d calls, got %d", len(m.calls), m.current)
	}
}

func newTestClient(t *testing.T, calls []mockCall) (*Client, *mockRunner) {
	t.Helper()
	var buf bytes.Buffer
	logger, err := logging.NewLoggerWithWriter(&buf, "", logging.Debug2)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	mr := &mockRunner{calls: calls, t: t}
	return NewClient(mr, logger), mr
}

func TestContainerExistsTrue(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		{wantName: "docker", wantArgs: []string{"inspect", "ollama"}, output: "{}", err: nil},
	})
	exists, err := c.ContainerExists(context.Background(), "ollama")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("ContainerExists = false, want true")
	}
	m.verify()
}

func TestContainerExistsFalse(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		{wantName: "docker", wantArgs: []string{"inspect", "missing"},
			output: "Error: No such container: missing", err: fmt.Errorf("command \"docker\" failed: exit status 1")},
	})
	exists, err := c.ContainerExists(context.Background(), "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("ContainerExists = true, want false")
	}
	m.verify()
}

func TestContainerIsRunningTrue(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		{wantName: "docker", wantArgs: []string{"inspect", "-f", "{{.State.Running}}", "ollama"},
			output: "true\n", err: nil},
	})
	running, err := c.ContainerIsRunning(context.Background(), "ollama")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !running {
		t.Error("ContainerIsRunning = false, want true")
	}
	m.verify()
}

func TestContainerIsRunningFalse(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		{wantName: "docker", wantArgs: []string{"inspect", "-f", "{{.State.Running}}", "ollama"},
			output: "false\n", err: nil},
	})
	running, err := c.ContainerIsRunning(context.Background(), "ollama")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if running {
		t.Error("ContainerIsRunning = true, want false")
	}
	m.verify()
}

func TestStopContainerSkipsWhenNotRunning(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		// ContainerIsRunning check: returns false.
		{wantName: "docker", wantArgs: []string{"inspect", "-f", "{{.State.Running}}", "myapp"},
			output: "false\n", err: nil},
	})
	err := c.StopContainer(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestStopContainerCallsDockerStop(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		// ContainerIsRunning check: returns true.
		{wantName: "docker", wantArgs: []string{"inspect", "-f", "{{.State.Running}}", "myapp"},
			output: "true\n", err: nil},
		// docker stop.
		{wantName: "docker", wantArgs: []string{"stop", "myapp"},
			output: "myapp\n", err: nil},
	})
	err := c.StopContainer(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestRemoveContainerSkipsWhenNotExists(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		// ContainerExists check: returns false.
		{wantName: "docker", wantArgs: []string{"inspect", "gone"},
			output: "Error: No such container: missing", err: fmt.Errorf("command \"docker\" failed: exit status 1")},
	})
	err := c.RemoveContainer(context.Background(), "gone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestRemoveContainerCallsDockerRm(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		// ContainerExists check: returns true.
		{wantName: "docker", wantArgs: []string{"inspect", "myapp"},
			output: "{}", err: nil},
		// docker rm.
		{wantName: "docker", wantArgs: []string{"rm", "myapp"},
			output: "myapp\n", err: nil},
	})
	err := c.RemoveContainer(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestStopAndRemoveCallsStopThenRemove(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		// StopContainer -> ContainerIsRunning: true.
		{wantName: "docker", output: "true\n", err: nil},
		// StopContainer -> docker stop.
		{wantName: "docker", output: "myapp\n", err: nil},
		// RemoveContainer -> ContainerExists: true.
		{wantName: "docker", output: "{}", err: nil},
		// RemoveContainer -> docker rm.
		{wantName: "docker", output: "myapp\n", err: nil},
	})
	err := c.StopAndRemove(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestStopAndRemoveSucceedsWhenContainerGone(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		// StopContainer -> ContainerIsRunning: inspect fails (not found).
		{wantName: "docker", output: "Error: No such container: missing", err: fmt.Errorf("command \"docker\" failed: exit status 1")},
		// RemoveContainer -> ContainerExists: inspect fails (not found).
		{wantName: "docker", output: "Error: No such container: missing", err: fmt.Errorf("command \"docker\" failed: exit status 1")},
	})
	err := c.StopAndRemove(context.Background(), "gone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestPullImageCallsDockerPull(t *testing.T) {
	c, m := newTestClient(t, []mockCall{
		{wantName: "docker", wantArgs: []string{"pull", "ollama/ollama:latest"},
			output: "Pulling...\n", err: nil},
	})
	err := c.PullImage(context.Background(), "ollama/ollama", "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestRunContainerBuildsCorrectCommand(t *testing.T) {
	cfg := ContainerConfig{
		Name:    "ollama",
		Image:   "ollama/ollama",
		Tag:     "latest",
		Volume:  "ollama",
		VolPath: "/root/.ollama",
		Env:     map[string]string{"OLLAMA_HOST": "localhost"},
		GPUs:    "all",
		Network: "host",
		Restart: "always",
	}

	expectedArgs := cfg.RunArgs()

	c, m := newTestClient(t, []mockCall{
		{wantName: "docker", wantArgs: expectedArgs, output: "abc123\n", err: nil},
	})
	err := c.RunContainer(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestEnsureRunningStopsRemovesRunsPolls(t *testing.T) {
	cfg := ContainerConfig{
		Name:  "myapp",
		Image: "img",
		Tag:   "v1",
	}

	c, m := newTestClient(t, []mockCall{
		// StopAndRemove -> StopContainer -> ContainerIsRunning: not running.
		{wantName: "docker", output: "false\n", err: nil},
		// StopAndRemove -> RemoveContainer -> ContainerExists: not found.
		{wantName: "docker", output: "Error: No such container: missing", err: fmt.Errorf("command \"docker\" failed: exit status 1")},
		// RunContainer -> docker run.
		{wantName: "docker", output: "abc123\n", err: nil},
		// Poll -> ContainerIsRunning: true (immediate success).
		{wantName: "docker", output: "true\n", err: nil},
	})
	err := c.EnsureRunning(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestEnsureRunningPollsUntilRunning(t *testing.T) {
	cfg := ContainerConfig{
		Name:  "slow",
		Image: "img",
		Tag:   "v1",
	}

	// Override the default retry opts by using a context with short timeout.
	// The client uses DefaultRetryOpts internally, but the poll loop has
	// a retry delay. For this test, the mock returns false once then true.
	// The actual sleep in the poll loop uses time.After, so this test will
	// take ~10s with real delays. We accept this since the mock runner is
	// fast and the sleep is the bottleneck.
	//
	// To keep the test fast, we rely on the fact that the first poll returns
	// false and the second returns true. The delay between is 10s from
	// DefaultRetryOpts. We use a generous context timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, m := newTestClient(t, []mockCall{
		// StopAndRemove -> StopContainer -> ContainerIsRunning: not running.
		{wantName: "docker", output: "false\n", err: nil},
		// StopAndRemove -> RemoveContainer -> ContainerExists: not found.
		{wantName: "docker", output: "Error: No such container: missing", err: fmt.Errorf("command \"docker\" failed: exit status 1")},
		// RunContainer -> docker run.
		{wantName: "docker", output: "abc123\n", err: nil},
		// Poll 1 -> ContainerIsRunning: false.
		{wantName: "docker", output: "false\n", err: nil},
		// Poll 2 -> ContainerIsRunning: true.
		{wantName: "docker", output: "true\n", err: nil},
	})
	err := c.EnsureRunning(ctx, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}
