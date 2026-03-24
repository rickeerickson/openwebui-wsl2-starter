package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	// Ollama defaults
	assertEqual(t, cfg.Ollama.Port, 11434, "ollama.port")
	assertEqual(t, cfg.Ollama.Host, "localhost", "ollama.host")
	assertEqual(t, cfg.Ollama.Image, "ollama/ollama", "ollama.image")
	assertEqual(t, cfg.Ollama.Tag, "latest", "ollama.tag")
	assertEqual(t, cfg.Ollama.Container, "ollama", "ollama.container")
	assertEqual(t, cfg.Ollama.Volume, "ollama", "ollama.volume")
	if len(cfg.Ollama.Models) != 1 || cfg.Ollama.Models[0] != "llama3.2:1b" {
		t.Errorf("ollama.models: got %v, want [llama3.2:1b]", cfg.Ollama.Models)
	}

	// OpenWebUI defaults
	assertEqual(t, cfg.OpenWebUI.Port, 3000, "openwebui.port")
	assertEqual(t, cfg.OpenWebUI.Host, "localhost", "openwebui.host")
	assertEqual(t, cfg.OpenWebUI.Image, "ghcr.io/open-webui/open-webui", "openwebui.image")
	assertEqual(t, cfg.OpenWebUI.Tag, "latest", "openwebui.tag")
	assertEqual(t, cfg.OpenWebUI.Container, "open-webui", "openwebui.container")
	assertEqual(t, cfg.OpenWebUI.Volume, "open-webui", "openwebui.volume")

	// WSL defaults
	assertEqual(t, cfg.WSL.Distro, "Ubuntu", "wsl.distro")

	// Proxy defaults
	assertEqual(t, cfg.Proxy.ListenAddress, "0.0.0.0", "proxy.listen_address")
	assertEqual(t, cfg.Proxy.ListenPort, 3000, "proxy.listen_port")
	assertEqual(t, cfg.Proxy.ConnectAddress, "127.0.0.1", "proxy.connect_address")
	assertEqual(t, cfg.Proxy.ConnectPort, 3000, "proxy.connect_port")
}

func TestLoadFromFile_Valid(t *testing.T) {
	cfg, err := LoadFromFile(filepath.Join("testdata", "valid.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, cfg.Ollama.Port, 11434, "ollama.port")
	assertEqual(t, cfg.Ollama.Host, "localhost", "ollama.host")
	assertEqual(t, cfg.Ollama.Image, "ollama/ollama", "ollama.image")
	assertEqual(t, cfg.OpenWebUI.Port, 3000, "openwebui.port")
	assertEqual(t, cfg.OpenWebUI.Image, "ghcr.io/open-webui/open-webui", "openwebui.image")
	assertEqual(t, cfg.WSL.Distro, "Ubuntu", "wsl.distro")
	assertEqual(t, cfg.Proxy.ListenAddress, "0.0.0.0", "proxy.listen_address")
	assertEqual(t, cfg.Proxy.ConnectPort, 3000, "proxy.connect_port")
}

func TestLoadFromFile_Minimal(t *testing.T) {
	cfg, err := LoadFromFile(filepath.Join("testdata", "minimal.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overridden field
	assertEqual(t, cfg.Ollama.Port, 9999, "ollama.port")

	// All other fields keep defaults
	defaults := Defaults()
	assertEqual(t, cfg.Ollama.Host, defaults.Ollama.Host, "ollama.host")
	assertEqual(t, cfg.Ollama.Image, defaults.Ollama.Image, "ollama.image")
	assertEqual(t, cfg.Ollama.Tag, defaults.Ollama.Tag, "ollama.tag")
	assertEqual(t, cfg.Ollama.Container, defaults.Ollama.Container, "ollama.container")
	assertEqual(t, cfg.Ollama.Volume, defaults.Ollama.Volume, "ollama.volume")
	if len(cfg.Ollama.Models) != 1 || cfg.Ollama.Models[0] != "llama3.2:1b" {
		t.Errorf("ollama.models: got %v, want [llama3.2:1b]", cfg.Ollama.Models)
	}
	assertEqual(t, cfg.OpenWebUI.Port, defaults.OpenWebUI.Port, "openwebui.port")
	assertEqual(t, cfg.OpenWebUI.Image, defaults.OpenWebUI.Image, "openwebui.image")
	assertEqual(t, cfg.WSL.Distro, defaults.WSL.Distro, "wsl.distro")
	assertEqual(t, cfg.Proxy.ListenAddress, defaults.Proxy.ListenAddress, "proxy.listen_address")
}

func TestLoadFromFile_NonexistentPath(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/to/config.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestLoadFromFile_MalformedYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{{{not yaml"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestValidate_Defaults(t *testing.T) {
	cfg := Defaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("defaults should be valid, got: %v", err)
	}
}

func TestValidate_PortZero(t *testing.T) {
	cfg := Defaults()
	cfg.Ollama.Port = 0
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for port 0")
	}
	assertContains(t, err.Error(), "ollama.port")
}

func TestValidate_Port65536(t *testing.T) {
	cfg := Defaults()
	cfg.Ollama.Port = 65536
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for port 65536")
	}
	assertContains(t, err.Error(), "ollama.port")
}

func TestValidate_PortNegative(t *testing.T) {
	cfg := Defaults()
	cfg.Ollama.Port = -1
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for port -1")
	}
	assertContains(t, err.Error(), "ollama.port")
}

func TestValidate_ContainerNameInjection(t *testing.T) {
	cfg := Defaults()
	cfg.Ollama.Container = "; rm -rf /"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for injected container name")
	}
	assertContains(t, err.Error(), "ollama.container")
}

func TestValidate_ContainerNameBacktick(t *testing.T) {
	cfg := Defaults()
	cfg.Ollama.Container = "test`cmd`"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for backtick in container name")
	}
	assertContains(t, err.Error(), "ollama.container")
}

func TestValidate_ContainerNameDollar(t *testing.T) {
	cfg := Defaults()
	cfg.Ollama.Container = "test$var"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for $ in container name")
	}
	assertContains(t, err.Error(), "ollama.container")
}

func TestValidate_InvalidHostname(t *testing.T) {
	cfg := Defaults()
	cfg.Ollama.Host = ".starts-with-dot"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for hostname starting with dot")
	}
	assertContains(t, err.Error(), "ollama.host")
}

func TestValidate_ImageWithRegistryPath(t *testing.T) {
	cfg := Defaults()
	cfg.OpenWebUI.Image = "ghcr.io/open-webui/open-webui"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("registry image path should be valid, got: %v", err)
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := Defaults()
	cfg.Ollama.Port = 0
	cfg.Ollama.Container = ";bad"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation errors")
	}
	errStr := err.Error()
	assertContains(t, errStr, "ollama.port")
	assertContains(t, errStr, "ollama.container")
}

func TestValidate_EmptyModelName(t *testing.T) {
	cfg := Defaults()
	cfg.Ollama.Models = []string{"llama3.2:1b", ""}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty model name")
	}
	assertContains(t, err.Error(), "must not be empty")
}

func TestRoundTrip(t *testing.T) {
	original := Defaults()

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var restored Config
	if err := yaml.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Compare all fields
	assertEqual(t, restored.Ollama.Port, original.Ollama.Port, "ollama.port")
	assertEqual(t, restored.Ollama.Host, original.Ollama.Host, "ollama.host")
	assertEqual(t, restored.Ollama.Image, original.Ollama.Image, "ollama.image")
	assertEqual(t, restored.Ollama.Tag, original.Ollama.Tag, "ollama.tag")
	assertEqual(t, restored.Ollama.Container, original.Ollama.Container, "ollama.container")
	assertEqual(t, restored.Ollama.Volume, original.Ollama.Volume, "ollama.volume")
	if len(restored.Ollama.Models) != len(original.Ollama.Models) {
		t.Fatalf("models length: got %d, want %d", len(restored.Ollama.Models), len(original.Ollama.Models))
	}
	for i := range original.Ollama.Models {
		assertEqual(t, restored.Ollama.Models[i], original.Ollama.Models[i], "ollama.models")
	}
	assertEqual(t, restored.OpenWebUI.Port, original.OpenWebUI.Port, "openwebui.port")
	assertEqual(t, restored.OpenWebUI.Host, original.OpenWebUI.Host, "openwebui.host")
	assertEqual(t, restored.OpenWebUI.Image, original.OpenWebUI.Image, "openwebui.image")
	assertEqual(t, restored.OpenWebUI.Tag, original.OpenWebUI.Tag, "openwebui.tag")
	assertEqual(t, restored.OpenWebUI.Container, original.OpenWebUI.Container, "openwebui.container")
	assertEqual(t, restored.OpenWebUI.Volume, original.OpenWebUI.Volume, "openwebui.volume")
	assertEqual(t, restored.WSL.Distro, original.WSL.Distro, "wsl.distro")
	assertEqual(t, restored.Proxy.ListenAddress, original.Proxy.ListenAddress, "proxy.listen_address")
	assertEqual(t, restored.Proxy.ListenPort, original.Proxy.ListenPort, "proxy.listen_port")
	assertEqual(t, restored.Proxy.ConnectAddress, original.Proxy.ConnectAddress, "proxy.connect_address")
	assertEqual(t, restored.Proxy.ConnectPort, original.Proxy.ConnectPort, "proxy.connect_port")
}

// assertEqual is a generic test helper for comparable types.
func assertEqual[T comparable](t *testing.T, got, want T, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", field, got, want)
	}
}

// assertContains checks that s contains substr.
func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
