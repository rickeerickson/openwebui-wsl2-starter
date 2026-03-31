package config

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestDefaultsMatchBashConfig parses update_open-webui.config.sh and asserts
// that every Bash default matches the Go Defaults() struct. This is a guardrail
// test that catches config drift between the two systems during side-by-side
// operation.
//
// This test has NO runtime dependency on Bash. It parses the config file as
// text using simple regex matching.
func TestDefaultsMatchBashConfig(t *testing.T) {
	bashCfgPath := "../../update_open-webui.config.sh"
	data, err := os.ReadFile(bashCfgPath)
	if err != nil {
		t.Fatalf("reading bash config: %v", err)
	}

	bashVars := parseBashConfig(string(data))
	defaults := Defaults()

	// Ollama settings.
	assertIntMatch(t, bashVars, "OLLAMA_PORT", defaults.Ollama.Port)
	assertStringMatch(t, bashVars, "OLLAMA_HOST", defaults.Ollama.Host)
	assertStringMatch(t, bashVars, "OLLAMA_CONTAINER_TAG", defaults.Ollama.Tag)
	assertStringMatch(t, bashVars, "OLLAMA_CONTAINER_NAME", defaults.Ollama.Container)
	assertStringMatch(t, bashVars, "OLLAMA_VOLUME_NAME", defaults.Ollama.Volume)

	// OpenWebUI settings.
	assertIntMatch(t, bashVars, "OPEN_WEBUI_PORT", defaults.OpenWebUI.Port)
	assertStringMatch(t, bashVars, "OPEN_WEBUI_HOST", defaults.OpenWebUI.Host)
	assertStringMatch(t, bashVars, "OPEN_WEBUI_CONTAINER_TAG", defaults.OpenWebUI.Tag)
	assertStringMatch(t, bashVars, "OPEN_WEBUI_CONTAINER_NAME", defaults.OpenWebUI.Container)
	assertStringMatch(t, bashVars, "OPEN_WEBUI_VOLUME_NAME", defaults.OpenWebUI.Volume)

	// Models.
	bashModels := parseBashArray(string(data), "DEFAULT_OLLAMA_MODELS")
	if len(bashModels) != len(defaults.Ollama.Models) {
		t.Fatalf("model count mismatch: bash=%d, go=%d", len(bashModels), len(defaults.Ollama.Models))
	}
	for i, model := range bashModels {
		if model != defaults.Ollama.Models[i] {
			t.Errorf("model[%d]: bash=%q, go=%q", i, model, defaults.Ollama.Models[i])
		}
	}
}

// parseBashConfig extracts KEY=VALUE pairs from a Bash config file.
// Handles both quoted and unquoted values.
func parseBashConfig(content string) map[string]string {
	vars := make(map[string]string)
	re := regexp.MustCompile(`^([A-Z_]+)=["']?([^"'\n]*)["']?`)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if m := re.FindStringSubmatch(line); m != nil {
			vars[m[1]] = m[2]
		}
	}
	return vars
}

// parseBashArray extracts values from a Bash array declaration like:
//
//	ARRAY_NAME=(
//	    "value1"
//	    "value2"
//	)
func parseBashArray(content string, name string) []string {
	re := regexp.MustCompile(name + `=\(\s*\n([\s\S]*?)\)`)
	m := re.FindStringSubmatch(content)
	if m == nil {
		return nil
	}

	var values []string
	valRe := regexp.MustCompile(`"([^"]+)"`)
	for _, match := range valRe.FindAllStringSubmatch(m[1], -1) {
		values = append(values, match[1])
	}
	return values
}

func assertStringMatch(t *testing.T, bashVars map[string]string, key string, goVal string) {
	t.Helper()
	bashVal, ok := bashVars[key]
	if !ok {
		t.Errorf("bash config missing key %s", key)
		return
	}
	if bashVal != goVal {
		t.Errorf("%s: bash=%q, go=%q", key, bashVal, goVal)
	}
}

func assertIntMatch(t *testing.T, bashVars map[string]string, key string, goVal int) {
	t.Helper()
	bashVal, ok := bashVars[key]
	if !ok {
		t.Errorf("bash config missing key %s", key)
		return
	}
	bashInt, err := strconv.Atoi(bashVal)
	if err != nil {
		t.Errorf("%s: bash value %q is not an integer", key, bashVal)
		return
	}
	if bashInt != goVal {
		t.Errorf("%s: bash=%d, go=%d", key, bashInt, goVal)
	}
}
