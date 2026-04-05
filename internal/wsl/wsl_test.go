//go:build windows

package wsl

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

type mockCall struct {
	Name string
	Args []string
}

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

func (r *mockRunner) key(name string, args ...string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}

func (r *mockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	r.calls = append(r.calls, mockCall{Name: name, Args: args})
	k := r.key(name, args...)
	if err, ok := r.errors[k]; ok {
		return r.outputs[k], err
	}
	return r.outputs[k], nil
}

func (r *mockRunner) RunWithRetry(ctx context.Context, _ exec.RetryOpts, name string, args ...string) (string, error) {
	return r.Run(ctx, name, args...)
}

func (r *mockRunner) called(name string, args ...string) bool {
	k := r.key(name, args...)
	for _, c := range r.calls {
		ck := r.key(c.Name, c.Args...)
		if ck == k {
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

func TestIsInstalledTrue(t *testing.T) {
	r := newMockRunner()
	mgr := NewManager(r, newTestLogger(t))

	installed, err := mgr.IsInstalled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !installed {
		t.Error("expected true")
	}
}

func TestIsInstalledFalse(t *testing.T) {
	r := newMockRunner()
	r.errors[r.key("wsl.exe", "--status")] = fmt.Errorf("not installed")

	mgr := NewManager(r, newTestLogger(t))
	installed, err := mgr.IsInstalled(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected false")
	}
}

func TestInstallCallsWslInstall(t *testing.T) {
	r := newMockRunner()
	mgr := NewManager(r, newTestLogger(t))

	err := mgr.Install(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("wsl.exe", "--install", "--no-distribution") {
		t.Error("expected wsl.exe --install --no-distribution call")
	}
}

func TestUpdateCallsWslUpdate(t *testing.T) {
	r := newMockRunner()
	mgr := NewManager(r, newTestLogger(t))

	err := mgr.Update(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("wsl.exe", "--update") {
		t.Error("expected wsl.exe --update call")
	}
}

func TestSetDefaultVersion(t *testing.T) {
	r := newMockRunner()
	mgr := NewManager(r, newTestLogger(t))

	err := mgr.SetDefaultVersion(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("wsl.exe", "--set-default-version", "2") {
		t.Error("expected wsl.exe --set-default-version 2 call")
	}
}

func TestStopCallsShutdown(t *testing.T) {
	r := newMockRunner()
	mgr := NewManager(r, newTestLogger(t))

	err := mgr.Stop(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("wsl.exe", "--shutdown") {
		t.Error("expected wsl.exe --shutdown call")
	}
}

func TestIsDistroInstalledTrue(t *testing.T) {
	r := newMockRunner()
	r.outputs[r.key("wsl.exe", "--list", "--verbose")] =
		"  NAME      STATE     VERSION\n* Ubuntu    Running   2\n  Debian    Stopped   2\n"

	mgr := NewManager(r, newTestLogger(t))
	installed, err := mgr.IsDistroInstalled(context.Background(), "Ubuntu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !installed {
		t.Error("expected true for Ubuntu")
	}
}

func TestIsDistroInstalledFalse(t *testing.T) {
	r := newMockRunner()
	r.outputs[r.key("wsl.exe", "--list", "--verbose")] =
		"  NAME      STATE     VERSION\n* Debian    Running   2\n"

	mgr := NewManager(r, newTestLogger(t))
	installed, err := mgr.IsDistroInstalled(context.Background(), "Ubuntu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected false for Ubuntu")
	}
}

func TestIsDistroInstalledCaseInsensitive(t *testing.T) {
	r := newMockRunner()
	r.outputs[r.key("wsl.exe", "--list", "--verbose")] =
		"  NAME      STATE     VERSION\n* ubuntu    Running   2\n"

	mgr := NewManager(r, newTestLogger(t))
	installed, err := mgr.IsDistroInstalled(context.Background(), "Ubuntu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !installed {
		t.Error("expected true (case insensitive)")
	}
}

func TestInstallDistro(t *testing.T) {
	r := newMockRunner()
	mgr := NewManager(r, newTestLogger(t))

	err := mgr.InstallDistro(context.Background(), "Ubuntu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("wsl.exe", "--install", "-d", "Ubuntu") {
		t.Error("expected wsl.exe --install -d Ubuntu call")
	}
	if !r.called("wsl.exe", "--setdefault", "Ubuntu") {
		t.Error("expected wsl.exe --setdefault Ubuntu call")
	}
}

func TestRemoveDistro(t *testing.T) {
	r := newMockRunner()
	mgr := NewManager(r, newTestLogger(t))

	err := mgr.RemoveDistro(context.Background(), "Ubuntu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("wsl.exe", "--unregister", "Ubuntu") {
		t.Error("expected wsl.exe --unregister Ubuntu call")
	}
}

func TestRunCommand(t *testing.T) {
	r := newMockRunner()
	r.outputs[r.key("wsl.exe", "-d", "Ubuntu", "--", "ls", "-la")] = "total 0\n"

	mgr := NewManager(r, newTestLogger(t))
	out, err := mgr.RunCommand(context.Background(), "Ubuntu", "ls", "-la")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "total 0\n" {
		t.Errorf("output = %q, want %q", out, "total 0\n")
	}
}

func TestContainsDistroWithDefaultMarker(t *testing.T) {
	output := "  NAME      STATE     VERSION\n* Ubuntu    Running   2\n"
	if !containsDistro(output, "Ubuntu") {
		t.Error("should find Ubuntu with * marker")
	}
}

func TestContainsDistroEmptyOutput(t *testing.T) {
	if containsDistro("", "Ubuntu") {
		t.Error("should not find Ubuntu in empty output")
	}
}

func TestIsDistroInstalledWhenWslFails(t *testing.T) {
	r := newMockRunner()
	r.errors[r.key("wsl.exe", "--list", "--verbose")] = fmt.Errorf("wsl not installed")

	mgr := NewManager(r, newTestLogger(t))
	installed, err := mgr.IsDistroInstalled(context.Background(), "Ubuntu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected false when wsl fails")
	}
}
