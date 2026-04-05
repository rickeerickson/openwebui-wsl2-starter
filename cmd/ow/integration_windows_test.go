//go:build windows && integration

package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"gopkg.in/yaml.v3"
)

// TestConfigShowRoundTripWindows builds the binary, runs `ow config show`,
// and verifies the YAML output unmarshals back to a config matching Defaults().
func TestConfigShowRoundTripWindows(t *testing.T) {
	binary := buildOWWindows(t)

	out, err := exec.Command(binary, "config", "show").CombinedOutput()
	if err != nil {
		t.Fatalf("ow config show failed: %v\n%s", err, out)
	}

	var got config.Config
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal config show output: %v", err)
	}

	want := config.Defaults()

	if got.Ollama.Port != want.Ollama.Port {
		t.Errorf("ollama.port: got %d, want %d", got.Ollama.Port, want.Ollama.Port)
	}
	if got.WSL.Distro != want.WSL.Distro {
		t.Errorf("wsl.distro: got %q, want %q", got.WSL.Distro, want.WSL.Distro)
	}
	if got.Proxy.ListenAddress != want.Proxy.ListenAddress {
		t.Errorf("proxy.listen_address: got %q, want %q",
			got.Proxy.ListenAddress, want.Proxy.ListenAddress)
	}
}

// TestVersionOutputWindows verifies that `ow version` runs and includes windows.
func TestVersionOutputWindows(t *testing.T) {
	binary := buildOWWindows(t)

	out, err := exec.Command(binary, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("ow version failed: %v\n%s", err, out)
	}

	if !strings.Contains(string(out), "windows") {
		t.Errorf("version output missing 'windows': %s", out)
	}
}

// TestSubcommandTreeWindows verifies all expected Windows subcommands exist.
func TestSubcommandTreeWindows(t *testing.T) {
	binary := buildOWWindows(t)

	tests := []struct {
		args []string
		want string
	}{
		{[]string{"help"}, "setup"},
		{[]string{"help"}, "wsl"},
		{[]string{"help"}, "proxy"},
		{[]string{"help"}, "diagnose"},
		{[]string{"help"}, "config"},
		{[]string{"help"}, "version"},
		{[]string{"wsl", "help"}, "install"},
		{[]string{"wsl", "help"}, "remove"},
		{[]string{"wsl", "help"}, "stop"},
		{[]string{"proxy", "help"}, "enable"},
		{[]string{"proxy", "help"}, "remove"},
		{[]string{"proxy", "help"}, "show"},
	}

	for _, tt := range tests {
		out, err := exec.Command(binary, tt.args...).CombinedOutput()
		if err != nil {
			t.Errorf("ow %s failed: %v", strings.Join(tt.args, " "), err)
			continue
		}
		if !strings.Contains(string(out), tt.want) {
			t.Errorf("ow %s output missing %q:\n%s",
				strings.Join(tt.args, " "), tt.want, out)
		}
	}
}

func buildOWWindows(t *testing.T) string {
	t.Helper()
	binary := t.TempDir() + "\\ow.exe"
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/ow/")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return binary
}
