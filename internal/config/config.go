package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config holds the full application configuration.
type Config struct {
	Ollama    OllamaConfig    `yaml:"ollama"`
	OpenWebUI OpenWebUIConfig `yaml:"openwebui"`
	WSL       WSLConfig       `yaml:"wsl"`
	Proxy     ProxyConfig     `yaml:"proxy"`
}

// OllamaConfig holds Ollama container settings.
type OllamaConfig struct {
	Port      int      `yaml:"port"`
	Host      string   `yaml:"host"`
	Image     string   `yaml:"image"`
	Tag       string   `yaml:"tag"`
	Container string   `yaml:"container"`
	Volume    string   `yaml:"volume"`
	Models    []string `yaml:"models"`
}

// OpenWebUIConfig holds OpenWebUI container settings.
type OpenWebUIConfig struct {
	Port      int    `yaml:"port"`
	Host      string `yaml:"host"`
	Image     string `yaml:"image"`
	Tag       string `yaml:"tag"`
	Container string `yaml:"container"`
	Volume    string `yaml:"volume"`
}

// WSLConfig holds WSL distribution settings.
type WSLConfig struct {
	Distro string `yaml:"distro"`
}

// ProxyConfig holds port proxy settings.
type ProxyConfig struct {
	ListenAddress  string `yaml:"listen_address"`
	ListenPort     int    `yaml:"listen_port"`
	ConnectAddress string `yaml:"connect_address"`
	ConnectPort    int    `yaml:"connect_port"`
}

var (
	namePattern     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.\-]*$`)
	imagePattern    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.\-/:]*$`)
	hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.\-]*[a-zA-Z0-9])?$`)
	modelPattern    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:/\-]*$`)
)

// Defaults returns a Config populated with hardcoded default values.
func Defaults() Config {
	return Config{
		Ollama: OllamaConfig{
			Port:      11434,
			Host:      "localhost",
			Image:     "ollama/ollama",
			Tag:       "latest",
			Container: "ollama",
			Volume:    "ollama",
			Models:    []string{"llama3.2:1b"},
		},
		OpenWebUI: OpenWebUIConfig{
			Port:      3000,
			Host:      "localhost",
			Image:     "ghcr.io/open-webui/open-webui",
			Tag:       "latest",
			Container: "open-webui",
			Volume:    "open-webui",
		},
		WSL: WSLConfig{
			Distro: "Ubuntu",
		},
		Proxy: ProxyConfig{
			ListenAddress:  "0.0.0.0",
			ListenPort:     3000,
			ConnectAddress: "127.0.0.1",
			ConnectPort:    3000,
		},
	}
}

// LoadFromFile reads a YAML config file and merges it onto the defaults.
// Fields not specified in the file keep their default values.
func LoadFromFile(path string) (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path) //nolint:gosec // G304: path comes from trusted caller, validated externally
	if err != nil {
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// Validate checks all config fields and returns an error containing
// every validation failure found. Returns nil if the config is valid.
func (c Config) Validate() error {
	var errs []error

	// Port validation
	validatePort(&errs, c.Ollama.Port, "ollama.port")
	validatePort(&errs, c.OpenWebUI.Port, "openwebui.port")
	validatePort(&errs, c.Proxy.ListenPort, "proxy.listen_port")
	validatePort(&errs, c.Proxy.ConnectPort, "proxy.connect_port")

	// Container name validation
	validateName(&errs, c.Ollama.Container, "ollama.container")
	validateName(&errs, c.OpenWebUI.Container, "openwebui.container")

	// Volume name validation
	validateName(&errs, c.Ollama.Volume, "ollama.volume")
	validateName(&errs, c.OpenWebUI.Volume, "openwebui.volume")

	// Image name validation
	validateImage(&errs, c.Ollama.Image, "ollama.image")
	validateImage(&errs, c.OpenWebUI.Image, "openwebui.image")

	// Tag validation (same pattern as name)
	validateName(&errs, c.Ollama.Tag, "ollama.tag")
	validateName(&errs, c.OpenWebUI.Tag, "openwebui.tag")

	// Hostname validation
	validateHostname(&errs, c.Ollama.Host, "ollama.host")
	validateHostname(&errs, c.OpenWebUI.Host, "openwebui.host")

	// WSL distro name validation
	validateName(&errs, c.WSL.Distro, "wsl.distro")

	// Proxy IP validation
	validateIP(&errs, c.Proxy.ListenAddress, "proxy.listen_address")
	validateIP(&errs, c.Proxy.ConnectAddress, "proxy.connect_address")

	// Models validation
	for i, model := range c.Ollama.Models {
		field := fmt.Sprintf("ollama.models[%d]", i)
		if model == "" {
			errs = append(errs, fmt.Errorf("%s: must not be empty", field))
			continue
		}
		if !modelPattern.MatchString(model) {
			errs = append(errs, fmt.Errorf("%s: %q does not match %s", field, model, modelPattern.String()))
		}
	}

	return errors.Join(errs...)
}

func validatePort(errs *[]error, port int, field string) {
	if port < 1 || port > 65535 {
		*errs = append(*errs, fmt.Errorf("%s: %d is not in range 1-65535", field, port))
	}
}

func validateName(errs *[]error, name, field string) {
	if !namePattern.MatchString(name) {
		*errs = append(*errs, fmt.Errorf("%s: %q does not match %s", field, name, namePattern.String()))
	}
}

func validateImage(errs *[]error, image, field string) {
	if !imagePattern.MatchString(image) {
		*errs = append(*errs, fmt.Errorf("%s: %q does not match %s", field, image, imagePattern.String()))
	}
}

func validateHostname(errs *[]error, hostname, field string) {
	if !hostnamePattern.MatchString(hostname) {
		*errs = append(*errs, fmt.Errorf("%s: %q does not match %s", field, hostname, hostnamePattern.String()))
	}
}

func validateIP(errs *[]error, ip, field string) {
	if net.ParseIP(ip) == nil {
		*errs = append(*errs, fmt.Errorf("%s: %q is not a valid IP address", field, ip))
	}
}
