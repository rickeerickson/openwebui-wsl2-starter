package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Resolve loads configuration with the full resolution chain:
//  1. Compiled defaults (Defaults())
//  2. ow.yaml in the working directory
//  3. ~/.config/ow/ow.yaml (user-level)
//  4. Environment variables (OW_ prefix)
//  5. CLI flags (passed as map[string]string)
//
// Later sources override earlier ones.
// Returns the resolved and validated config.
//
// Note: ollama.models cannot be set via environment variables or CLI flags
// because viper and the flags map[string]string handle string slices poorly.
// Models should be set via YAML configuration files.
func Resolve(flags map[string]string) (Config, error) {
	v := viper.New()

	// Step 1: Set defaults from Defaults() struct.
	d := Defaults()
	v.SetDefault("ollama.port", d.Ollama.Port)
	v.SetDefault("ollama.host", d.Ollama.Host)
	v.SetDefault("ollama.image", d.Ollama.Image)
	v.SetDefault("ollama.tag", d.Ollama.Tag)
	v.SetDefault("ollama.container", d.Ollama.Container)
	v.SetDefault("ollama.volume", d.Ollama.Volume)
	v.SetDefault("ollama.models", d.Ollama.Models)
	v.SetDefault("openwebui.port", d.OpenWebUI.Port)
	v.SetDefault("openwebui.host", d.OpenWebUI.Host)
	v.SetDefault("openwebui.image", d.OpenWebUI.Image)
	v.SetDefault("openwebui.tag", d.OpenWebUI.Tag)
	v.SetDefault("openwebui.container", d.OpenWebUI.Container)
	v.SetDefault("openwebui.volume", d.OpenWebUI.Volume)
	v.SetDefault("wsl.distro", d.WSL.Distro)
	v.SetDefault("proxy.listen_address", d.Proxy.ListenAddress)
	v.SetDefault("proxy.listen_port", d.Proxy.ListenPort)
	v.SetDefault("proxy.connect_address", d.Proxy.ConnectAddress)
	v.SetDefault("proxy.connect_port", d.Proxy.ConnectPort)

	// Step 2: Config file discovery.
	v.SetConfigName("ow")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	home, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "ow"))
	}

	// Step 3: Read config file. Missing file is not an error.
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, fmt.Errorf("reading config file: %w", err)
		}
	}

	// Step 4: Environment variable binding.
	v.SetEnvPrefix("OW")

	// Bind each nested key explicitly so viper maps the flattened
	// env var name (e.g., OW_OLLAMA_PORT) to the nested key.
	envBindings := map[string]string{
		"ollama.port":           "OW_OLLAMA_PORT",
		"ollama.host":           "OW_OLLAMA_HOST",
		"ollama.image":          "OW_OLLAMA_IMAGE",
		"ollama.tag":            "OW_OLLAMA_TAG",
		"ollama.container":      "OW_OLLAMA_CONTAINER",
		"ollama.volume":         "OW_OLLAMA_VOLUME",
		"openwebui.port":        "OW_OPENWEBUI_PORT",
		"openwebui.host":        "OW_OPENWEBUI_HOST",
		"openwebui.image":       "OW_OPENWEBUI_IMAGE",
		"openwebui.tag":         "OW_OPENWEBUI_TAG",
		"openwebui.container":   "OW_OPENWEBUI_CONTAINER",
		"openwebui.volume":      "OW_OPENWEBUI_VOLUME",
		"wsl.distro":            "OW_WSL_DISTRO",
		"proxy.listen_address":  "OW_PROXY_LISTEN_ADDRESS",
		"proxy.listen_port":     "OW_PROXY_LISTEN_PORT",
		"proxy.connect_address": "OW_PROXY_CONNECT_ADDRESS",
		"proxy.connect_port":    "OW_PROXY_CONNECT_PORT",
	}
	for key, env := range envBindings {
		if bindErr := v.BindEnv(key, env); bindErr != nil {
			return Config{}, fmt.Errorf("binding env %s: %w", env, bindErr)
		}
	}

	// Step 5: Apply CLI flag overrides.
	for key, value := range flags {
		v.Set(key, value)
	}

	// Step 6: Unmarshal into Config struct.
	// Use the "yaml" struct tag for mapstructure field matching so that
	// keys like "proxy.listen_address" map to ProxyConfig.ListenAddress
	// (tagged `yaml:"listen_address"`) without adding mapstructure tags
	// to the Config structs.
	yamlTagOption := viper.DecoderConfigOption(func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	})
	var cfg Config
	if err := v.Unmarshal(&cfg, yamlTagOption); err != nil {
		return Config{}, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Step 7: Validate.
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}
