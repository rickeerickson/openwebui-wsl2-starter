//go:build linux

package main

import (
	"reflect"
	"testing"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
)

// TestOllamaContainerConfigMatchesBash verifies that the Go container config
// produces docker run arguments identical to the Bash scripts.
//
// Bash (repo_lib.sh lines 572-579):
//
//	docker run -d \
//	    --gpus all \
//	    --network=host \
//	    --volume ollama:/root/.ollama \
//	    --env OLLAMA_HOST=localhost \
//	    --restart always \
//	    --name ollama \
//	    ollama/ollama:latest
func TestOllamaContainerConfigMatchesBash(t *testing.T) {
	cfg := config.Defaults()
	cc := ollamaContainerConfig(cfg)

	got := cc.RunArgs()
	want := []string{
		"run", "-d",
		"--name", "ollama",
		"--gpus", "all",
		"--network", "host",
		"--restart", "always",
		"--volume", "ollama:/root/.ollama",
		"--env", "OLLAMA_HOST=localhost",
		"ollama/ollama:latest",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ollama RunArgs() =\n  %v\nwant\n  %v", got, want)
	}
}

// TestOpenWebUIContainerConfigMatchesBash verifies that the Go container config
// produces docker run arguments identical to the Bash scripts.
//
// Bash (repo_lib.sh lines 647-655):
//
//	docker run -d \
//	    --gpus all \
//	    --network=host \
//	    --volume open-webui:/app/backend/data \
//	    --env OLLAMA_BASE_URL=http://localhost:11434 \
//	    --env PORT=3000 \
//	    --name open-webui \
//	    --restart always \
//	    ghcr.io/open-webui/open-webui:latest
func TestOpenWebUIContainerConfigMatchesBash(t *testing.T) {
	cfg := config.Defaults()
	cc := openWebUIContainerConfig(cfg)

	got := cc.RunArgs()
	want := []string{
		"run", "-d",
		"--name", "open-webui",
		"--gpus", "all",
		"--network", "host",
		"--restart", "always",
		"--volume", "open-webui:/app/backend/data",
		"--env", "OLLAMA_BASE_URL=http://localhost:11434",
		"--env", "PORT=3000",
		"ghcr.io/open-webui/open-webui:latest",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("openwebui RunArgs() =\n  %v\nwant\n  %v", got, want)
	}
}

func TestOllamaContainerConfigCustomValues(t *testing.T) {
	cfg := config.Defaults()
	cfg.Ollama.Container = "my-ollama"
	cfg.Ollama.Tag = "0.3.0"
	cfg.Ollama.Volume = "my-vol"
	cfg.Ollama.Host = "0.0.0.0"

	cc := ollamaContainerConfig(cfg)

	if cc.Name != "my-ollama" {
		t.Errorf("Name = %q, want %q", cc.Name, "my-ollama")
	}
	if cc.Tag != "0.3.0" {
		t.Errorf("Tag = %q, want %q", cc.Tag, "0.3.0")
	}
	if cc.Volume != "my-vol" {
		t.Errorf("Volume = %q, want %q", cc.Volume, "my-vol")
	}
	if cc.Env["OLLAMA_HOST"] != "0.0.0.0" {
		t.Errorf("Env[OLLAMA_HOST] = %q, want %q", cc.Env["OLLAMA_HOST"], "0.0.0.0")
	}
	if cc.VolPath != ollamaVolPath {
		t.Errorf("VolPath = %q, want %q", cc.VolPath, ollamaVolPath)
	}
}

func TestOpenWebUIContainerConfigCustomValues(t *testing.T) {
	cfg := config.Defaults()
	cfg.OpenWebUI.Port = 8080
	cfg.Ollama.Port = 11435
	cfg.Ollama.Host = "192.168.1.10"

	cc := openWebUIContainerConfig(cfg)

	if cc.Env["PORT"] != "8080" {
		t.Errorf("Env[PORT] = %q, want %q", cc.Env["PORT"], "8080")
	}
	if cc.Env["OLLAMA_BASE_URL"] != "http://192.168.1.10:11435" {
		t.Errorf("Env[OLLAMA_BASE_URL] = %q, want %q",
			cc.Env["OLLAMA_BASE_URL"], "http://192.168.1.10:11435")
	}
	if cc.VolPath != openWebUIVolPath {
		t.Errorf("VolPath = %q, want %q", cc.VolPath, openWebUIVolPath)
	}
}
