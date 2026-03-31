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

	// Download GPG key.
	gpgKey, err := i.Runner.Run(ctx, "curl", "-fsSL", gpgKeyURL)
	if err != nil {
		return fmt.Errorf("download GPG key: %w", err)
	}

	// Dearmor and write the keyring. We pass the key via stdin by writing
	// it to a temp approach. Since we cannot pipe, we use sh -c.
	// Actually, the runner doesn't support stdin. Write key to a temp file
	// approach would need filesystem access. Instead, use sh -c for the
	// gpg dearmor step.
	_, err = i.Runner.Run(ctx, "sh", "-c",
		fmt.Sprintf("echo '%s' | gpg --dearmor --yes -o %s", gpgKey, keyringPath))
	if err != nil {
		return fmt.Errorf("dearmor GPG key: %w", err)
	}

	// Download and configure the apt repository list.
	repoList, err := i.Runner.Run(ctx, "curl", "-s", "-L", repoListURL)
	if err != nil {
		return fmt.Errorf("download repo list: %w", err)
	}

	// Add signed-by to the repo entries and write to the sources list.
	signedRepo := strings.ReplaceAll(repoList, "deb https://",
		fmt.Sprintf("deb [signed-by=%s] https://", keyringPath))
	_, err = i.Runner.Run(ctx, "sh", "-c",
		fmt.Sprintf("echo '%s' > %s", signedRepo, repoFile))
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
