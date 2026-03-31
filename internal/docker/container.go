//go:build linux

package docker

import "fmt"

// ContainerConfig describes the settings for a Docker container.
type ContainerConfig struct {
	Name    string
	Image   string
	Tag     string
	Volume  string
	VolPath string // mount path inside container
	Env     map[string]string
	GPUs    string // "all" or empty
	Network string // "host" or empty
	Restart string // "always" or empty
}

// ImageRef returns "image:tag", falling back to "latest" if Tag is empty.
func (c ContainerConfig) ImageRef() string {
	tag := c.Tag
	if tag == "" {
		tag = "latest"
	}
	return c.Image + ":" + tag
}

// RunArgs builds the argument list for `docker run -d` from the config fields.
// The returned slice does not include the "docker" binary name itself.
func (c ContainerConfig) RunArgs() []string {
	args := []string{"run", "-d"}

	if c.Name != "" {
		args = append(args, "--name", c.Name)
	}
	if c.GPUs != "" {
		args = append(args, "--gpus", c.GPUs)
	}
	if c.Network != "" {
		args = append(args, "--network", c.Network)
	}
	if c.Restart != "" {
		args = append(args, "--restart", c.Restart)
	}
	if c.Volume != "" && c.VolPath != "" {
		args = append(args, "--volume", fmt.Sprintf("%s:%s", c.Volume, c.VolPath))
	}

	// Sort env keys for deterministic output in tests.
	// Use a simple insertion sort since env maps are small.
	keys := make([]string, 0, len(c.Env))
	for k := range c.Env {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	for _, k := range keys {
		args = append(args, "--env", fmt.Sprintf("%s=%s", k, c.Env[k]))
	}

	args = append(args, c.ImageRef())
	return args
}
