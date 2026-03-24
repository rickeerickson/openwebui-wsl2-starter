package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve_DefaultsOnly(t *testing.T) {
	// No config file in cwd, no env vars, no flags.
	tmp := t.TempDir()
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	cfg, err := Resolve(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaults := Defaults()
	assertEqual(t, cfg.Ollama.Port, defaults.Ollama.Port, "ollama.port")
	assertEqual(t, cfg.Ollama.Host, defaults.Ollama.Host, "ollama.host")
	assertEqual(t, cfg.Ollama.Container, defaults.Ollama.Container, "ollama.container")
	assertEqual(t, cfg.OpenWebUI.Port, defaults.OpenWebUI.Port, "openwebui.port")
	assertEqual(t, cfg.WSL.Distro, defaults.WSL.Distro, "wsl.distro")
	assertEqual(t, cfg.Proxy.ListenPort, defaults.Proxy.ListenPort, "proxy.listen_port")
}

func TestResolve_FileOverridesDefaults(t *testing.T) {
	tmp := t.TempDir()
	writeYAML(t, filepath.Join(tmp, "ow.yaml"), `
ollama:
  port: 9999
  container: "custom-ollama"
openwebui:
  port: 4000
`)
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	cfg, err := Resolve(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, cfg.Ollama.Port, 9999, "ollama.port")
	assertEqual(t, cfg.Ollama.Container, "custom-ollama", "ollama.container")
	assertEqual(t, cfg.OpenWebUI.Port, 4000, "openwebui.port")
	// Non-overridden fields keep defaults.
	assertEqual(t, cfg.Ollama.Host, Defaults().Ollama.Host, "ollama.host")
}

func TestResolve_EnvOverridesFile(t *testing.T) {
	tmp := t.TempDir()
	writeYAML(t, filepath.Join(tmp, "ow.yaml"), `
ollama:
  port: 8888
`)
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	t.Setenv("OW_OLLAMA_PORT", "9999")

	cfg, err := Resolve(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, cfg.Ollama.Port, 9999, "ollama.port")
}

func TestResolve_FlagOverridesEnv(t *testing.T) {
	tmp := t.TempDir()
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	t.Setenv("OW_OLLAMA_PORT", "9999")

	flags := map[string]string{
		"ollama.port": "7777",
	}
	cfg, err := Resolve(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, cfg.Ollama.Port, 7777, "ollama.port")
}

func TestResolve_FullChain(t *testing.T) {
	// default < file < env < flag for the same field.
	tmp := t.TempDir()
	writeYAML(t, filepath.Join(tmp, "ow.yaml"), `
ollama:
  port: 1111
  host: "filehost"
  container: "file-ollama"
openwebui:
  port: 2222
`)
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	t.Setenv("OW_OLLAMA_HOST", "envhost")
	t.Setenv("OW_OPENWEBUI_PORT", "3333")

	flags := map[string]string{
		"openwebui.port": "4444",
	}

	cfg, err := Resolve(flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File overrides default.
	assertEqual(t, cfg.Ollama.Port, 1111, "ollama.port (file > default)")
	assertEqual(t, cfg.Ollama.Container, "file-ollama", "ollama.container (file > default)")
	// Env overrides file.
	assertEqual(t, cfg.Ollama.Host, "envhost", "ollama.host (env > file)")
	// Flag overrides env.
	assertEqual(t, cfg.OpenWebUI.Port, 4444, "openwebui.port (flag > env)")
}

func TestResolve_EnvNestedKey(t *testing.T) {
	tmp := t.TempDir()
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	t.Setenv("OW_OLLAMA_CONTAINER", "myollama")

	cfg, err := Resolve(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, cfg.Ollama.Container, "myollama", "ollama.container")
}

func TestResolve_UnsetEnvDoesNotOverrideFile(t *testing.T) {
	tmp := t.TempDir()
	writeYAML(t, filepath.Join(tmp, "ow.yaml"), `
ollama:
  container: "from-file"
`)
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	// Do not set OW_OLLAMA_CONTAINER in env.
	cfg, err := Resolve(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, cfg.Ollama.Container, "from-file", "ollama.container")
}

func TestResolve_EmptyFlagsIsNoop(t *testing.T) {
	tmp := t.TempDir()
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	cfg, err := Resolve(map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaults := Defaults()
	assertEqual(t, cfg.Ollama.Port, defaults.Ollama.Port, "ollama.port")
	assertEqual(t, cfg.OpenWebUI.Port, defaults.OpenWebUI.Port, "openwebui.port")
}

func TestResolve_InvalidEnvPortValue(t *testing.T) {
	tmp := t.TempDir()
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	t.Setenv("OW_OLLAMA_PORT", "abc")

	// Viper coerces "abc" to 0 for an int field, which fails port validation.
	_, err := Resolve(nil)
	if err == nil {
		t.Fatal("expected error for invalid port value from env")
	}
	assertContains(t, err.Error(), "ollama.port")
}

func TestResolve_ConfigFileNotFoundIsNotError(t *testing.T) {
	tmp := t.TempDir()
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	cfg, err := Resolve(nil)
	if err != nil {
		t.Fatalf("expected no error when config file is missing, got: %v", err)
	}

	defaults := Defaults()
	assertEqual(t, cfg.Ollama.Port, defaults.Ollama.Port, "ollama.port")
}

func TestResolve_ValidationAfterResolution(t *testing.T) {
	tmp := t.TempDir()
	restoreDir := chdir(t, tmp)
	defer restoreDir()

	flags := map[string]string{
		"ollama.port": "0",
	}

	_, err := Resolve(flags)
	if err == nil {
		t.Fatal("expected validation error for port 0")
	}
	assertContains(t, err.Error(), "ollama.port")
}

// chdir changes the working directory to dir and returns a function
// that restores the original directory.
func chdir(t *testing.T, dir string) func() {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("restore chdir %s: %v", orig, err)
		}
	}
}

// writeYAML writes content to path, trimming leading whitespace from the
// heredoc-style string.
func writeYAML(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
