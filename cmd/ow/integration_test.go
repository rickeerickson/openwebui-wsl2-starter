//go:build linux && integration

package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"gopkg.in/yaml.v3"
)

// TestConfigShowRoundTrip builds the binary, runs `ow config show`, and
// verifies the YAML output unmarshals back to a config matching Defaults().
func TestConfigShowRoundTrip(t *testing.T) {
	binary := buildOW(t)

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
	if got.Ollama.Host != want.Ollama.Host {
		t.Errorf("ollama.host: got %q, want %q", got.Ollama.Host, want.Ollama.Host)
	}
	if got.Ollama.Container != want.Ollama.Container {
		t.Errorf("ollama.container: got %q, want %q", got.Ollama.Container, want.Ollama.Container)
	}
	if got.OpenWebUI.Port != want.OpenWebUI.Port {
		t.Errorf("openwebui.port: got %d, want %d", got.OpenWebUI.Port, want.OpenWebUI.Port)
	}
	if got.OpenWebUI.Container != want.OpenWebUI.Container {
		t.Errorf("openwebui.container: got %q, want %q", got.OpenWebUI.Container, want.OpenWebUI.Container)
	}
	if len(got.Ollama.Models) != len(want.Ollama.Models) {
		t.Fatalf("models count: got %d, want %d", len(got.Ollama.Models), len(want.Ollama.Models))
	}
	for i := range want.Ollama.Models {
		if got.Ollama.Models[i] != want.Ollama.Models[i] {
			t.Errorf("models[%d]: got %q, want %q", i, got.Ollama.Models[i], want.Ollama.Models[i])
		}
	}
}

// TestVersionOutput verifies that `ow version` runs and includes the OS/arch.
func TestVersionOutput(t *testing.T) {
	binary := buildOW(t)

	out, err := exec.Command(binary, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("ow version failed: %v\n%s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "linux") {
		t.Errorf("version output missing 'linux': %s", output)
	}
}

// TestSubcommandTree verifies all expected subcommands exist.
func TestSubcommandTree(t *testing.T) {
	binary := buildOW(t)

	tests := []struct {
		args []string
		want string
	}{
		{[]string{"help"}, "setup"},
		{[]string{"help"}, "containers"},
		{[]string{"help"}, "ollama"},
		{[]string{"help"}, "diagnose"},
		{[]string{"help"}, "config"},
		{[]string{"help"}, "version"},
		{[]string{"containers", "help"}, "up"},
		{[]string{"containers", "help"}, "down"},
		{[]string{"containers", "help"}, "status"},
		{[]string{"ollama", "help"}, "pull"},
		{[]string{"ollama", "help"}, "models"},
		{[]string{"ollama", "help"}, "run"},
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

// buildOW compiles the ow binary for the current platform and returns
// the path. It uses t.TempDir() for the output.
func buildOW(t *testing.T) string {
	t.Helper()
	binary := t.TempDir() + "/ow"
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/ow/")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return binary
}
