//go:build linux

package apt

import (
	"context"
	"fmt"
	"os"
	"os/user"
)

const (
	dockerKeyringPath = "/etc/apt/keyrings/docker.asc"
	dockerGPGURL      = "https://download.docker.com/linux/ubuntu/gpg"
	dockerSourcesList = "/etc/apt/sources.list.d/docker.list"

	nvidiaKeyringPath = "/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg"
	nvidiaGPGURL      = "https://nvidia.github.io/libnvidia-container/gpgkey"
	nvidiaSourcesList = "/etc/apt/sources.list.d/nvidia-container-toolkit.list"
	nvidiaRepoListURL = "https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list"
)

// SetupDockerKeyring downloads the Docker GPG key and adds the Docker apt
// repository. Skips if the keyring file already exists.
func (m *Manager) SetupDockerKeyring(ctx context.Context) error {
	if _, err := os.Stat(dockerKeyringPath); err == nil {
		m.Logger.Info("docker keyring already exists at %s, skipping", dockerKeyringPath)
		return nil
	}

	m.Logger.Info("setting up Docker GPG keyring and repository")

	// Ensure keyrings directory exists
	_, err := m.Runner.RunWithRetry(ctx, retryOpts(), "install", "-m", "0755", "-d", "/etc/apt/keyrings")
	if err != nil {
		return fmt.Errorf("create keyrings dir: %w", err)
	}

	// Download Docker GPG key
	_, err = m.Runner.RunWithRetry(ctx, retryOpts(), "curl", "-fsSL", dockerGPGURL, "-o", dockerKeyringPath)
	if err != nil {
		return fmt.Errorf("download docker gpg key: %w", err)
	}

	// Set permissions on key file
	_, err = m.Runner.Run(ctx, "chmod", "a+r", dockerKeyringPath)
	if err != nil {
		return fmt.Errorf("chmod docker keyring: %w", err)
	}

	// Add Docker repository to sources list.
	// The repo line includes architecture and codename resolved at runtime
	// via shell command substitution. Double quotes allow expansion.
	repoLine := `deb [arch=$(dpkg --print-architecture) signed-by=` + dockerKeyringPath + `] ` +
		`https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable`
	_, err = m.Runner.Run(ctx, "sh", "-c", `echo "`+repoLine+`" > `+dockerSourcesList)
	if err != nil {
		return fmt.Errorf("write docker sources list: %w", err)
	}

	m.Logger.Info("Docker GPG keyring and repository setup complete")
	return nil
}

// SetupNvidiaKeyring downloads the NVIDIA Container Toolkit GPG key and adds
// the NVIDIA apt repository. Skips if the keyring file already exists.
func (m *Manager) SetupNvidiaKeyring(ctx context.Context) error {
	if _, err := os.Stat(nvidiaKeyringPath); err == nil {
		m.Logger.Info("nvidia keyring already exists at %s, skipping", nvidiaKeyringPath)
		return nil
	}

	m.Logger.Info("setting up NVIDIA Container Toolkit GPG keyring")

	// Download and dearmor the NVIDIA GPG key
	_, err := m.Runner.RunWithRetry(ctx, retryOpts(),
		"sh", "-c",
		"curl -fsSL "+nvidiaGPGURL+" | gpg --dearmor --yes -o "+nvidiaKeyringPath)
	if err != nil {
		return fmt.Errorf("download nvidia gpg key: %w", err)
	}

	// Download repo list and add signed-by directive
	_, err = m.Runner.RunWithRetry(ctx, retryOpts(),
		"sh", "-c",
		"curl -s -L "+nvidiaRepoListURL+
			" | sed 's#deb https://#deb [signed-by="+nvidiaKeyringPath+"] https://#g'"+
			" > "+nvidiaSourcesList)
	if err != nil {
		return fmt.Errorf("write nvidia sources list: %w", err)
	}

	m.Logger.Info("NVIDIA GPG keyring and repository setup complete")
	return nil
}

// InstallDocker installs Docker Engine and related packages.
func (m *Manager) InstallDocker(ctx context.Context) error {
	m.Logger.Info("installing Docker")

	// Update package lists first
	_, err := m.Runner.RunWithRetry(ctx, retryOpts(), "apt-get", "update")
	if err != nil {
		return fmt.Errorf("apt-get update: %w", err)
	}

	return m.InstallPackages(ctx,
		"docker-ce",
		"docker-ce-cli",
		"containerd.io",
		"docker-buildx-plugin",
		"docker-compose-plugin",
	)
}

// ConfigureDockerNvidia configures the NVIDIA runtime for Docker by running
// nvidia-ctk and restarting the docker service.
func (m *Manager) ConfigureDockerNvidia(ctx context.Context) error {
	m.Logger.Info("configuring Docker NVIDIA runtime")

	_, err := m.Runner.RunWithRetry(ctx, retryOpts(),
		"nvidia-ctk", "runtime", "configure", "--runtime=docker")
	if err != nil {
		return fmt.Errorf("nvidia-ctk configure: %w", err)
	}

	_, err = m.Runner.RunWithRetry(ctx, retryOpts(), "systemctl", "restart", "docker")
	if err != nil {
		return fmt.Errorf("restart docker: %w", err)
	}

	m.Logger.Info("Docker NVIDIA runtime configured")
	return nil
}

// AddUserToDockerGroup adds the current user to the docker group.
func (m *Manager) AddUserToDockerGroup(ctx context.Context) error {
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}

	m.Logger.Info("adding user %s to docker group", u.Username)

	_, err = m.Runner.Run(ctx, "usermod", "-aG", "docker", u.Username)
	if err != nil {
		return fmt.Errorf("usermod: %w", err)
	}

	return nil
}
