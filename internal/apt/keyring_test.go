//go:build linux

package apt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// keyringMockRunner extends mockRunner to support per-call error configuration.
type keyringMockRunner struct {
	calls     []mockCall
	failCalls map[string]error // keyed by "name arg0" or just "name"
}

func (r *keyringMockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	r.calls = append(r.calls, mockCall{Name: name, Args: args})
	key := name
	if len(args) > 0 {
		key = name + " " + args[0]
	}
	if err, ok := r.failCalls[key]; ok && err != nil {
		return "", err
	}
	if err, ok := r.failCalls[name]; ok && err != nil {
		return "", err
	}
	return "", nil
}

func (r *keyringMockRunner) RunWithRetry(ctx context.Context, opts exec.RetryOpts, name string, args ...string) (string, error) {
	return r.Run(ctx, name, args...)
}

func newKeyringTestManager(t *testing.T, runner exec.Runner) *Manager {
	t.Helper()
	var buf bytes.Buffer
	logger, err := logging.NewLoggerWithWriter(&buf, "", logging.Debug2)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	return NewManager(runner, logger)
}

func TestSetupDockerKeyringSkipsWhenExists(t *testing.T) {
	// Create a temp file to stand in for the keyring
	tmpDir := t.TempDir()
	fakeKeyring := filepath.Join(tmpDir, "docker.asc")
	if err := os.WriteFile(fakeKeyring, []byte("key"), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	// Patch the constant by overriding the check. Since we can't easily
	// override the const, we test via a helper that takes the path.
	// Instead, we test the actual function but only when the file exists
	// at the real path. For unit testing, we verify the mock isn't called.
	//
	// We'll test the skip logic indirectly: if the file doesn't exist at
	// the real path (/etc/apt/keyrings/docker.asc), all commands run.
	// If it does exist, no commands run.
	r := &keyringMockRunner{failCalls: map[string]error{}}
	m := newKeyringTestManager(t, r)

	// Check if the real keyring file exists
	_, err := os.Stat(dockerKeyringPath)
	if err == nil {
		// File exists on this system; setup should be a no-op
		err = m.SetupDockerKeyring(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(r.calls) != 0 {
			t.Errorf("expected 0 calls when keyring exists, got %d", len(r.calls))
		}
	} else {
		// File doesn't exist; setup should run commands
		err = m.SetupDockerKeyring(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(r.calls) == 0 {
			t.Error("expected commands to run when keyring is missing")
		}
	}
}

func TestSetupDockerKeyringDownloadsWhenMissing(t *testing.T) {
	// Skip if the keyring actually exists on this system
	if _, err := os.Stat(dockerKeyringPath); err == nil {
		t.Skip("docker keyring exists on this system")
	}

	r := &keyringMockRunner{failCalls: map[string]error{}}
	m := newKeyringTestManager(t, r)

	err := m.SetupDockerKeyring(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect: install (keyrings dir), curl (download key), chmod, sh (write sources)
	if len(r.calls) < 4 {
		t.Fatalf("expected at least 4 calls, got %d: %v", len(r.calls), r.calls)
	}

	// First call: create keyrings directory
	if r.calls[0].Name != "install" {
		t.Errorf("call 0: name = %q, want %q", r.calls[0].Name, "install")
	}

	// Second call: curl to download GPG key
	if r.calls[1].Name != "curl" {
		t.Errorf("call 1: name = %q, want %q", r.calls[1].Name, "curl")
	}

	// Third call: chmod
	if r.calls[2].Name != "chmod" {
		t.Errorf("call 2: name = %q, want %q", r.calls[2].Name, "chmod")
	}

	// Fourth call: sh to write sources list
	if r.calls[3].Name != "sh" {
		t.Errorf("call 3: name = %q, want %q", r.calls[3].Name, "sh")
	}
}

func TestSetupNvidiaKeyringSkipsWhenExists(t *testing.T) {
	if _, err := os.Stat(nvidiaKeyringPath); err != nil {
		t.Skip("nvidia keyring does not exist on this system")
	}

	r := &keyringMockRunner{failCalls: map[string]error{}}
	m := newKeyringTestManager(t, r)

	err := m.SetupNvidiaKeyring(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.calls) != 0 {
		t.Errorf("expected 0 calls when keyring exists, got %d", len(r.calls))
	}
}

func TestSetupNvidiaKeyringDownloadsWhenMissing(t *testing.T) {
	if _, err := os.Stat(nvidiaKeyringPath); err == nil {
		t.Skip("nvidia keyring exists on this system")
	}

	r := &keyringMockRunner{failCalls: map[string]error{}}
	m := newKeyringTestManager(t, r)

	err := m.SetupNvidiaKeyring(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect: sh (curl + gpg dearmor), sh (curl + sed for repo list)
	if len(r.calls) < 2 {
		t.Fatalf("expected at least 2 calls, got %d: %v", len(r.calls), r.calls)
	}

	// Both calls should be via sh -c
	for i, call := range r.calls {
		if call.Name != "sh" {
			t.Errorf("call %d: name = %q, want %q", i, call.Name, "sh")
		}
	}
}

func TestInstallDockerPassesCorrectPackages(t *testing.T) {
	r := &keyringMockRunner{failCalls: map[string]error{}}
	m := newKeyringTestManager(t, r)

	err := m.InstallDocker(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect: apt-get update, sh -c "DEBIAN_FRONTEND=noninteractive apt-get install ..."
	if len(r.calls) < 2 {
		t.Fatalf("expected at least 2 calls, got %d", len(r.calls))
	}

	// First: apt-get update
	if r.calls[0].Name != "apt-get" || r.calls[0].Args[0] != "update" {
		t.Errorf("call 0: got %s %v, want apt-get update", r.calls[0].Name, r.calls[0].Args)
	}

	// Second: sh -c with DEBIAN_FRONTEND and all docker packages
	installCall := r.calls[1]
	if installCall.Name != "sh" {
		t.Errorf("call 1: name = %q, want %q", installCall.Name, "sh")
	}

	wantPkgs := []string{"docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"}
	argsStr := strings.Join(installCall.Args, " ")
	for _, pkg := range wantPkgs {
		if !strings.Contains(argsStr, pkg) {
			t.Errorf("install args missing package %q, got: %v", pkg, installCall.Args)
		}
	}
}

func TestConfigureDockerNvidiaWritesConfigAndRestarts(t *testing.T) {
	r := &keyringMockRunner{failCalls: map[string]error{}}
	m := newKeyringTestManager(t, r)

	err := m.ConfigureDockerNvidia(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(r.calls), r.calls)
	}

	// First: nvidia-ctk configure
	if r.calls[0].Name != "nvidia-ctk" {
		t.Errorf("call 0: name = %q, want %q", r.calls[0].Name, "nvidia-ctk")
	}
	argsStr := strings.Join(r.calls[0].Args, " ")
	if !strings.Contains(argsStr, "--runtime=docker") {
		t.Errorf("nvidia-ctk args missing --runtime=docker, got: %v", r.calls[0].Args)
	}

	// Second: systemctl restart docker
	if r.calls[1].Name != "systemctl" {
		t.Errorf("call 1: name = %q, want %q", r.calls[1].Name, "systemctl")
	}
	if len(r.calls[1].Args) < 2 || r.calls[1].Args[0] != "restart" || r.calls[1].Args[1] != "docker" {
		t.Errorf("call 1: args = %v, want [restart docker]", r.calls[1].Args)
	}
}

func TestAddUserToDockerGroupRunsUsermod(t *testing.T) {
	r := &keyringMockRunner{failCalls: map[string]error{}}
	m := newKeyringTestManager(t, r)

	err := m.AddUserToDockerGroup(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(r.calls))
	}

	call := r.calls[0]
	if call.Name != "usermod" {
		t.Errorf("name = %q, want %q", call.Name, "usermod")
	}
	if len(call.Args) < 3 || call.Args[0] != "-aG" || call.Args[1] != "docker" {
		t.Errorf("args = %v, want [-aG docker <username>]", call.Args)
	}
}

func TestConfigureDockerNvidiaErrorOnConfigure(t *testing.T) {
	r := &keyringMockRunner{failCalls: map[string]error{
		"nvidia-ctk": fmt.Errorf("nvidia-ctk not found"),
	}}
	m := newKeyringTestManager(t, r)

	err := m.ConfigureDockerNvidia(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "nvidia-ctk") {
		t.Errorf("error = %q, want it to contain 'nvidia-ctk'", err.Error())
	}
}

func TestInstallDockerErrorOnUpdate(t *testing.T) {
	r := &keyringMockRunner{failCalls: map[string]error{
		"apt-get": fmt.Errorf("network error"),
	}}
	m := newKeyringTestManager(t, r)

	err := m.InstallDocker(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("error = %q, want it to contain 'network error'", err.Error())
	}
}
