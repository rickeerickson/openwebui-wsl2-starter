//go:build linux

package ollama

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// mockCall records a single Runner invocation.
type mockCall struct {
	Name string
	Args []string
}

// mockRunner records calls and returns preconfigured responses.
type mockRunner struct {
	calls   []mockCall
	outputs map[string]string
	errors  map[string]error
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
	}
}

func (m *mockRunner) key(name string, args ...string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}

func (m *mockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	m.calls = append(m.calls, mockCall{Name: name, Args: args})
	k := m.key(name, args...)
	if err, ok := m.errors[k]; ok {
		return m.outputs[k], err
	}
	return m.outputs[k], nil
}

func (m *mockRunner) RunWithRetry(ctx context.Context, _ exec.RetryOpts, name string, args ...string) (string, error) {
	return m.Run(ctx, name, args...)
}

func (m *mockRunner) called(name string, args ...string) bool {
	k := m.key(name, args...)
	for _, c := range m.calls {
		ck := m.key(c.Name, c.Args...)
		if ck == k {
			return true
		}
	}
	return false
}

func (m *mockRunner) callCount() int {
	return len(m.calls)
}

// calledPrefix returns true if any call starts with the given name and args.
func (m *mockRunner) calledPrefix(name string, args ...string) bool {
	for _, c := range m.calls {
		if c.Name != name {
			continue
		}
		if len(c.Args) < len(args) {
			continue
		}
		match := true
		for i, a := range args {
			if c.Args[i] != a {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func newTestLogger(t *testing.T) *logging.Logger {
	t.Helper()
	var buf bytes.Buffer
	l, err := logging.NewLoggerWithWriter(&buf, "", logging.Debug2)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	return l
}

func TestInstallSkipsWhenAlreadyInstalled(t *testing.T) {
	m := newMockRunner()
	m.outputs[m.key("ollama", "--version")] = "ollama version 0.1.0"

	mgr := NewManager(m, newTestLogger(t))
	err := mgr.Install(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the version check should be called.
	if m.callCount() != 1 {
		t.Errorf("expected 1 call, got %d: %+v", m.callCount(), m.calls)
	}
}

func TestInstallRunsCurlAndShWhenMissing(t *testing.T) {
	m := newMockRunner()
	m.errors[m.key("ollama", "--version")] = fmt.Errorf("not found")

	mgr := NewManager(m, newTestLogger(t))

	// We need --version to fail first time, succeed second time.
	// Replace the mock with a stateful one.
	sm := &statefulMockRunner{
		mockRunner:    m,
		versionCalls: 0,
	}
	mgr.Runner = sm

	err := mgr.Install(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have called curl and sh.
	if !sm.calledPrefix("curl", "-fsSL", "-o") {
		t.Error("expected curl call to download install script")
	}
	if !sm.calledPrefix("sh") {
		t.Error("expected sh call to run install script")
	}
}

// statefulMockRunner tracks version call count so the first fails and
// subsequent succeed.
type statefulMockRunner struct {
	*mockRunner
	versionCalls int
}

func (s *statefulMockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	if name == "ollama" && len(args) > 0 && args[0] == "--version" {
		s.versionCalls++
		if s.versionCalls == 1 {
			s.mockRunner.calls = append(s.mockRunner.calls, mockCall{Name: name, Args: args})
			return "", fmt.Errorf("not found")
		}
		s.mockRunner.calls = append(s.mockRunner.calls, mockCall{Name: name, Args: args})
		return "ollama version 0.1.0", nil
	}
	return s.mockRunner.Run(ctx, name, args...)
}

func (s *statefulMockRunner) RunWithRetry(ctx context.Context, opts exec.RetryOpts, name string, args ...string) (string, error) {
	return s.Run(ctx, name, args...)
}

func TestIsInstalledReturnsTrueOnSuccess(t *testing.T) {
	m := newMockRunner()
	m.outputs[m.key("ollama", "--version")] = "ollama version 0.1.0"

	mgr := NewManager(m, newTestLogger(t))
	installed, err := mgr.IsInstalled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !installed {
		t.Error("expected IsInstalled to return true")
	}
}

func TestIsInstalledReturnsFalseOnFailure(t *testing.T) {
	m := newMockRunner()
	m.errors[m.key("ollama", "--version")] = fmt.Errorf("not found")

	mgr := NewManager(m, newTestLogger(t))
	installed, err := mgr.IsInstalled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected IsInstalled to return false")
	}
}
