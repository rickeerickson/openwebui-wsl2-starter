//go:build linux

package apt

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// mockCall records a single command invocation.
type mockCall struct {
	Name string
	Args []string
}

// mockRunner records commands and returns preconfigured results.
type mockRunner struct {
	calls  []mockCall
	errors map[string]error // keyed by command name; nil means success
}

func (r *mockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	r.calls = append(r.calls, mockCall{Name: name, Args: args})
	if err, ok := r.errors[name]; ok && err != nil {
		return "", err
	}
	return "", nil
}

func (r *mockRunner) RunWithRetry(ctx context.Context, opts exec.RetryOpts, name string, args ...string) (string, error) {
	return r.Run(ctx, name, args...)
}

func newTestManager(t *testing.T, runner *mockRunner) *Manager {
	t.Helper()
	var buf bytes.Buffer
	logger, err := logging.NewLoggerWithWriter(&buf, "", logging.Debug2)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	return NewManager(runner, logger)
}

func TestUpdatePackagesRunsAllCommands(t *testing.T) {
	r := &mockRunner{errors: map[string]error{}}
	m := newTestManager(t, r)

	err := m.UpdatePackages(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"apt-get update",
		"DEBIAN_FRONTEND=noninteractive apt-get upgrade -y",
		"DEBIAN_FRONTEND=noninteractive apt-get dist-upgrade -y",
		"apt-get autoremove -y",
		"apt-get autoclean -y",
	}
	if len(r.calls) != len(expected) {
		t.Fatalf("got %d calls, want %d", len(r.calls), len(expected))
	}
	for i, want := range expected {
		if r.calls[i].Name != "sh" {
			t.Errorf("call %d: name = %q, want %q", i, r.calls[i].Name, "sh")
		}
		// Args should be ["-c", <shell command>]
		if len(r.calls[i].Args) != 2 || r.calls[i].Args[0] != "-c" {
			t.Errorf("call %d: args = %v, want [\"-c\", ...]", i, r.calls[i].Args)
			continue
		}
		if r.calls[i].Args[1] != want {
			t.Errorf("call %d: shell cmd = %q, want %q", i, r.calls[i].Args[1], want)
		}
	}
}

func TestInstallPackagesPassesNames(t *testing.T) {
	r := &mockRunner{errors: map[string]error{}}
	m := newTestManager(t, r)

	err := m.InstallPackages(context.Background(), "vim", "git", "curl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(r.calls))
	}
	call := r.calls[0]
	if call.Name != "sh" {
		t.Errorf("name = %q, want %q", call.Name, "sh")
	}
	// Args should be: -c "DEBIAN_FRONTEND=noninteractive apt-get install -y vim git curl"
	wantShellCmd := "DEBIAN_FRONTEND=noninteractive apt-get install -y vim git curl"
	if len(call.Args) != 2 || call.Args[0] != "-c" {
		t.Fatalf("args = %v, want [\"-c\", %q]", call.Args, wantShellCmd)
	}
	if call.Args[1] != wantShellCmd {
		t.Errorf("shell cmd = %q, want %q", call.Args[1], wantShellCmd)
	}
}

func TestIsInstalledReturnsTrueOnSuccess(t *testing.T) {
	r := &mockRunner{errors: map[string]error{}}
	m := newTestManager(t, r)

	installed, err := m.IsInstalled(context.Background(), "vim")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !installed {
		t.Error("expected true, got false")
	}

	if len(r.calls) != 1 || r.calls[0].Name != "dpkg" {
		t.Errorf("expected dpkg call, got %v", r.calls)
	}
}

func TestIsInstalledReturnsFalseOnFailure(t *testing.T) {
	r := &mockRunner{errors: map[string]error{
		"dpkg": fmt.Errorf("no packages found matching vim"),
	}}
	m := newTestManager(t, r)

	installed, err := m.IsInstalled(context.Background(), "vim")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if installed {
		t.Error("expected false, got true")
	}
}

func TestUpdatePackagesUsesRunWithRetry(t *testing.T) {
	// Verify that RunWithRetry is called (not Run) by checking that
	// the mock's RunWithRetry path is exercised. We use a specialized
	// mock that tracks which method was called.
	retryRunner := &retryTrackingRunner{errors: map[string]error{}}
	var buf bytes.Buffer
	logger, err := logging.NewLoggerWithWriter(&buf, "", logging.Debug2)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	m := NewManager(retryRunner, logger)

	err = m.UpdatePackages(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retryRunner.retryCount != 5 {
		t.Errorf("RunWithRetry called %d times, want 5", retryRunner.retryCount)
	}
	if retryRunner.runCount != 0 {
		t.Errorf("Run called %d times, want 0", retryRunner.runCount)
	}
}

func TestUpdatePackagesStopsOnError(t *testing.T) {
	r := &mockRunner{errors: map[string]error{
		"sh": fmt.Errorf("lock held"),
	}}
	m := newTestManager(t, r)

	err := m.UpdatePackages(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "lock held") {
		t.Errorf("error = %q, want it to contain 'lock held'", err.Error())
	}
	// Should stop after first failed command
	if len(r.calls) != 1 {
		t.Errorf("got %d calls, want 1 (should stop on first failure)", len(r.calls))
	}
}

// retryTrackingRunner counts calls to Run vs RunWithRetry.
type retryTrackingRunner struct {
	runCount   int
	retryCount int
	errors     map[string]error
}

func (r *retryTrackingRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	r.runCount++
	if err, ok := r.errors[name]; ok && err != nil {
		return "", err
	}
	return "", nil
}

func (r *retryTrackingRunner) RunWithRetry(ctx context.Context, opts exec.RetryOpts, name string, args ...string) (string, error) {
	r.retryCount++
	if err, ok := r.errors[name]; ok && err != nil {
		return "", err
	}
	return "", nil
}
