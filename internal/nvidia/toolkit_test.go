//go:build linux

package nvidia

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
	outputs map[string]string // key = "name arg1 arg2 ..."
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
	m.outputs[m.key("dpkg", "-l", "nvidia-container-toolkit")] =
		"ii  nvidia-container-toolkit  1.14.0  amd64  NVIDIA Container Toolkit"

	inst := NewInstaller(m, newTestLogger(t))
	err := inst.Install(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have called dpkg, nothing else.
	if m.callCount() != 1 {
		t.Errorf("expected 1 call (dpkg check), got %d: %+v", m.callCount(), m.calls)
	}
}

func TestInstallRunsFullSequenceWhenMissing(t *testing.T) {
	m := newMockRunner()
	// dpkg check fails (package not installed).
	m.errors[m.key("dpkg", "-l", "nvidia-container-toolkit")] =
		fmt.Errorf("dpkg-query: no packages found")
	m.outputs[m.key("curl", "-fsSL", gpgKeyURL)] = "FAKE-GPG-KEY"
	m.outputs[m.key("curl", "-s", "-L", repoListURL)] = "deb https://nvidia.github.io/libnvidia-container/stable/deb amd64/"

	inst := NewInstaller(m, newTestLogger(t))
	err := inst.Install(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify full install sequence: dpkg, curl gpg, sh gpg, curl repo, sh repo, apt-get update, apt-get install.
	if m.callCount() < 6 {
		t.Errorf("expected at least 6 calls, got %d: %+v", m.callCount(), m.calls)
	}

	if !m.called("apt-get", "install", "-y", "nvidia-container-toolkit") {
		t.Error("expected apt-get install call")
	}
}

func TestInstallGPGKeyUsesCorrectURL(t *testing.T) {
	m := newMockRunner()
	m.errors[m.key("dpkg", "-l", "nvidia-container-toolkit")] =
		fmt.Errorf("not installed")
	m.outputs[m.key("curl", "-fsSL", gpgKeyURL)] = "KEY"
	m.outputs[m.key("curl", "-s", "-L", repoListURL)] = "deb https://example.com/ amd64/"

	inst := NewInstaller(m, newTestLogger(t))
	_ = inst.Install(context.Background())

	if !m.called("curl", "-fsSL", gpgKeyURL) {
		t.Errorf("expected curl call with GPG key URL %s", gpgKeyURL)
	}
}

func TestInstallAddsCorrectAptRepository(t *testing.T) {
	m := newMockRunner()
	m.errors[m.key("dpkg", "-l", "nvidia-container-toolkit")] =
		fmt.Errorf("not installed")
	m.outputs[m.key("curl", "-fsSL", gpgKeyURL)] = "KEY"
	m.outputs[m.key("curl", "-s", "-L", repoListURL)] =
		"deb https://nvidia.github.io/libnvidia-container/stable/deb amd64/"

	inst := NewInstaller(m, newTestLogger(t))
	_ = inst.Install(context.Background())

	if !m.called("curl", "-s", "-L", repoListURL) {
		t.Errorf("expected curl call with repo list URL %s", repoListURL)
	}

	// The sh -c call should contain signed-by.
	found := false
	for _, c := range m.calls {
		if c.Name == "sh" && len(c.Args) >= 2 {
			combined := strings.Join(c.Args, " ")
			if strings.Contains(combined, "signed-by=") &&
				strings.Contains(combined, repoFile) {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected sh -c call that writes signed-by repo to sources list")
	}
}

func TestVerifySucceeds(t *testing.T) {
	m := newMockRunner()
	m.outputs[m.key("nvidia-smi")] = "GPU 0: Tesla T4"

	inst := NewInstaller(m, newTestLogger(t))
	err := inst.Verify(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyFailsOnNvidiaSmiError(t *testing.T) {
	m := newMockRunner()
	m.errors[m.key("nvidia-smi")] = fmt.Errorf("command failed: exit 1")

	inst := NewInstaller(m, newTestLogger(t))
	err := inst.Verify(context.Background())
	if err == nil {
		t.Fatal("expected error when nvidia-smi fails")
	}
	if !strings.Contains(err.Error(), "nvidia-smi failed") {
		t.Errorf("error = %q, want it to contain 'nvidia-smi failed'", err.Error())
	}
}

func TestInstallAptGetUpdateFailure(t *testing.T) {
	m := newMockRunner()
	m.errors[m.key("dpkg", "-l", "nvidia-container-toolkit")] =
		fmt.Errorf("not installed")
	m.outputs[m.key("curl", "-fsSL", gpgKeyURL)] = "KEY"
	m.outputs[m.key("curl", "-s", "-L", repoListURL)] = "deb https://example.com/ amd64/"
	m.errors[m.key("apt-get", "update")] = fmt.Errorf("apt-get update failed")

	inst := NewInstaller(m, newTestLogger(t))
	err := inst.Install(context.Background())
	if err == nil {
		t.Fatal("expected error when apt-get update fails")
	}
	if !strings.Contains(err.Error(), "apt-get update") {
		t.Errorf("error = %q, want it to contain 'apt-get update'", err.Error())
	}
}
