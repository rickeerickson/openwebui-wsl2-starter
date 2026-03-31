//go:build linux

package nvidia

import (
	"context"
	"fmt"
	"strings"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

const (
	gpgKeyURL   = "https://nvidia.github.io/libnvidia-container/gpgkey"
	repoListURL = "https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list"
	keyringPath = "/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg"
	repoFile    = "/etc/apt/sources.list.d/nvidia-container-toolkit.list"
)

// Installer manages NVIDIA Container Toolkit installation and verification.
type Installer struct {
	Runner exec.Runner
	Logger *logging.Logger
}

// NewInstaller creates an Installer with the given runner and logger.
func NewInstaller(runner exec.Runner, logger *logging.Logger) *Installer {
	return &Installer{Runner: runner, Logger: logger}
}

// Install installs the NVIDIA Container Toolkit if it is not already present.
// It checks dpkg for an existing installation, and if missing, downloads the
// GPG key, adds the apt repository, updates package lists, and installs.
func (i *Installer) Install(ctx context.Context) error {
	installed, err := i.isInstalled(ctx)
	if err != nil {
		return fmt.Errorf("check nvidia-container-toolkit: %w", err)
	}
	if installed {
		i.Logger.Info("nvidia-container-toolkit is already installed, skipping")
		return nil
	}

	i.Logger.Info("installing NVIDIA Container Toolkit")

	// Download GPG key and dearmor it in a single pipeline.
	// Avoids interpolating remote content into a shell command string.
	_, err = i.Runner.Run(ctx, "sh", "-c",
		fmt.Sprintf("curl -fsSL %s | gpg --dearmor --yes -o %s", gpgKeyURL, keyringPath))
	if err != nil {
		return fmt.Errorf("download and dearmor GPG key: %w", err)
	}

	// Download repo list, add signed-by directive, and write to sources.
	// All done in a single pipeline to avoid interpolating remote content.
	_, err = i.Runner.Run(ctx, "sh", "-c",
		fmt.Sprintf("curl -s -L %s | sed 's#deb https://#deb [signed-by=%s] https://#g' > %s",
			repoListURL, keyringPath, repoFile))
	if err != nil {
		return fmt.Errorf("write repo list: %w", err)
	}

	// Update package lists.
	_, err = i.Runner.RunWithRetry(ctx, exec.DefaultRetryOpts(), "apt-get", "update")
	if err != nil {
		return fmt.Errorf("apt-get update: %w", err)
	}

	// Install the toolkit.
	_, err = i.Runner.RunWithRetry(ctx, exec.DefaultRetryOpts(),
		"apt-get", "install", "-y", "nvidia-container-toolkit")
	if err != nil {
		return fmt.Errorf("apt-get install: %w", err)
	}

	i.Logger.Info("NVIDIA Container Toolkit installed")
	return nil
}

// Verify runs nvidia-smi to confirm GPU access.
func (i *Installer) Verify(ctx context.Context) error {
	i.Logger.Info("verifying GPU access via nvidia-smi")
	_, err := i.Runner.Run(ctx, "nvidia-smi")
	if err != nil {
		return fmt.Errorf("nvidia-smi failed: %w", err)
	}
	i.Logger.Info("GPU access verified")
	return nil
}

// isInstalled checks whether nvidia-container-toolkit is installed via dpkg.
func (i *Installer) isInstalled(ctx context.Context) (bool, error) {
	out, err := i.Runner.Run(ctx, "dpkg", "-l", "nvidia-container-toolkit")
	if err != nil {
		// dpkg -l exits non-zero when the package is not installed.
		return false, nil //nolint:nilerr // expected: dpkg exits 1 for missing packages
	}
	// A line starting with "ii" means the package is installed.
	return strings.Contains(out, "ii  nvidia-container-toolkit"), nil
}
