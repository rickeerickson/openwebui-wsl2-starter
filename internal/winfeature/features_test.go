//go:build windows

package winfeature

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

func (r *mockRunner) callCount() int {
	return len(r.calls)
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

func TestIsEnabledTrue(t *testing.T) {
	r := newMockRunner()
	checkKey := r.key("powershell.exe", "-Command",
		"(Get-WindowsOptionalFeature -FeatureName Microsoft-Windows-Subsystem-Linux -Online).State")
	r.outputs[checkKey] = "Enabled\n"

	mgr := NewManager(r, newTestLogger(t))
	enabled, err := mgr.IsEnabled(context.Background(), featureWSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected true")
	}
}

func TestIsEnabledFalse(t *testing.T) {
	r := newMockRunner()
	checkKey := r.key("powershell.exe", "-Command",
		"(Get-WindowsOptionalFeature -FeatureName VirtualMachinePlatform -Online).State")
	r.outputs[checkKey] = "Disabled\n"

	mgr := NewManager(r, newTestLogger(t))
	enabled, err := mgr.IsEnabled(context.Background(), featureVMP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enabled {
		t.Error("expected false")
	}
}

func TestEnableIfNeededSkipsWhenEnabled(t *testing.T) {
	r := newMockRunner()
	checkKey := r.key("powershell.exe", "-Command",
		"(Get-WindowsOptionalFeature -FeatureName Microsoft-Windows-Subsystem-Linux -Online).State")
	r.outputs[checkKey] = "Enabled\n"

	mgr := NewManager(r, newTestLogger(t))
	changed, err := mgr.EnableIfNeeded(context.Background(), featureWSL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Error("expected false (already enabled)")
	}
	// Should only check, not enable.
	if r.callCount() != 1 {
		t.Errorf("expected 1 call (check only), got %d", r.callCount())
	}
}

func TestEnableIfNeededEnablesWhenDisabled(t *testing.T) {
	r := newMockRunner()
	checkKey := r.key("powershell.exe", "-Command",
		"(Get-WindowsOptionalFeature -FeatureName VirtualMachinePlatform -Online).State")
	r.outputs[checkKey] = "Disabled\n"

	mgr := NewManager(r, newTestLogger(t))
	changed, err := mgr.EnableIfNeeded(context.Background(), featureVMP)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected true (newly enabled)")
	}
	// Should check then enable.
	if r.callCount() != 2 {
		t.Errorf("expected 2 calls (check + enable), got %d", r.callCount())
	}
}

func TestEnableWSLFeaturesAllEnabled(t *testing.T) {
	r := newMockRunner()
	wslKey := r.key("powershell.exe", "-Command",
		"(Get-WindowsOptionalFeature -FeatureName Microsoft-Windows-Subsystem-Linux -Online).State")
	vmpKey := r.key("powershell.exe", "-Command",
		"(Get-WindowsOptionalFeature -FeatureName VirtualMachinePlatform -Online).State")
	r.outputs[wslKey] = "Enabled\n"
	r.outputs[vmpKey] = "Enabled\n"

	mgr := NewManager(r, newTestLogger(t))
	changed, err := mgr.EnableWSLFeatures(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Error("expected false (both already enabled)")
	}
}

func TestEnableWSLFeaturesOneDisabled(t *testing.T) {
	r := newMockRunner()
	wslKey := r.key("powershell.exe", "-Command",
		"(Get-WindowsOptionalFeature -FeatureName Microsoft-Windows-Subsystem-Linux -Online).State")
	vmpKey := r.key("powershell.exe", "-Command",
		"(Get-WindowsOptionalFeature -FeatureName VirtualMachinePlatform -Online).State")
	r.outputs[wslKey] = "Enabled\n"
	r.outputs[vmpKey] = "Disabled\n"

	mgr := NewManager(r, newTestLogger(t))
	changed, err := mgr.EnableWSLFeatures(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected true (one was disabled)")
	}
}

func TestIsEnabledError(t *testing.T) {
	r := newMockRunner()
	checkKey := r.key("powershell.exe", "-Command",
		"(Get-WindowsOptionalFeature -FeatureName BadFeature -Online).State")
	r.errors[checkKey] = fmt.Errorf("feature not found")

	mgr := NewManager(r, newTestLogger(t))
	_, err := mgr.IsEnabled(context.Background(), "BadFeature")
	if err == nil {
		t.Fatal("expected error")
	}
}
