//go:build linux

package docker

import (
	"reflect"
	"testing"
)

func TestRunArgsAllFields(t *testing.T) {
	cfg := ContainerConfig{
		Name:    "ollama",
		Image:   "ollama/ollama",
		Tag:     "latest",
		Volume:  "ollama",
		VolPath: "/root/.ollama",
		Env:     map[string]string{"OLLAMA_HOST": "localhost"},
		GPUs:    "all",
		Network: "host",
		Restart: "always",
	}

	got := cfg.RunArgs()
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
		t.Errorf("RunArgs() =\n  %v\nwant\n  %v", got, want)
	}
}

func TestRunArgsMinimalFields(t *testing.T) {
	cfg := ContainerConfig{
		Name:  "myapp",
		Image: "myimage",
		Tag:   "v1",
	}

	got := cfg.RunArgs()
	want := []string{
		"run", "-d",
		"--name", "myapp",
		"myimage:v1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("RunArgs() =\n  %v\nwant\n  %v", got, want)
	}
}

func TestRunArgsMultipleEnvVars(t *testing.T) {
	cfg := ContainerConfig{
		Name:  "webui",
		Image: "ghcr.io/open-webui/open-webui",
		Tag:   "latest",
		Env: map[string]string{
			"PORT":            "3000",
			"OLLAMA_BASE_URL": "http://localhost:11434",
		},
	}

	got := cfg.RunArgs()
	want := []string{
		"run", "-d",
		"--name", "webui",
		"--env", "OLLAMA_BASE_URL=http://localhost:11434",
		"--env", "PORT=3000",
		"ghcr.io/open-webui/open-webui:latest",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("RunArgs() =\n  %v\nwant\n  %v", got, want)
	}
}

func TestImageRefDefaultTag(t *testing.T) {
	cfg := ContainerConfig{Image: "myimage"}
	got := cfg.ImageRef()
	if got != "myimage:latest" {
		t.Errorf("ImageRef() = %q, want %q", got, "myimage:latest")
	}
}

func TestImageRefExplicitTag(t *testing.T) {
	cfg := ContainerConfig{Image: "myimage", Tag: "v2"}
	got := cfg.ImageRef()
	if got != "myimage:v2" {
		t.Errorf("ImageRef() = %q, want %q", got, "myimage:v2")
	}
}

func TestRunArgsVolumeRequiresBothFields(t *testing.T) {
	// Volume set but VolPath empty: no --volume arg.
	cfg := ContainerConfig{
		Name:   "test",
		Image:  "img",
		Tag:    "v1",
		Volume: "vol",
	}
	for _, arg := range cfg.RunArgs() {
		if arg == "--volume" {
			t.Error("RunArgs() should not include --volume when VolPath is empty")
		}
	}

	// VolPath set but Volume empty: no --volume arg.
	cfg2 := ContainerConfig{
		Name:    "test",
		Image:   "img",
		Tag:     "v1",
		VolPath: "/data",
	}
	for _, arg := range cfg2.RunArgs() {
		if arg == "--volume" {
			t.Error("RunArgs() should not include --volume when Volume is empty")
		}
	}
}
