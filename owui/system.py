"""System setup functions.

Ports the system configuration functions from
bash/common/repo_lib.sh (lines 192-323). Handles package updates,
Docker installation, NVIDIA toolkit, and environment verification.
"""

from __future__ import annotations

import shutil
import subprocess

from owui.log import get_logger
from owui.retry import run_with_retry

logger = get_logger(__name__)


def update_system_packages() -> None:
    """Run apt-get update, upgrade, dist-upgrade, autoremove, autoclean."""
    logger.info("Updating system packages...")
    run_with_retry(["sudo", "apt-get", "update"])
    run_with_retry(["sudo", "apt-get", "upgrade", "-y"])
    run_with_retry(["sudo", "apt-get", "dist-upgrade", "-y"])
    run_with_retry(["sudo", "apt-get", "autoremove", "-y"])
    run_with_retry(["sudo", "apt-get", "autoclean", "-y"])
    logger.info("System packages updated.")


def setup_docker_keyring() -> None:
    """Install Docker GPG key and add the Docker apt repository.

    Installs ca-certificates and curl, downloads the Docker GPG key,
    and configures the apt source list for Docker packages.
    """
    logger.info("Setting up Docker GPG keyring and repository...")
    run_with_retry(
        ["sudo", "apt-get", "install", "-y", "ca-certificates", "curl"],
    )
    run_with_retry(
        ["sudo", "install", "-m", "0755", "-d", "/etc/apt/keyrings"],
    )
    run_with_retry(
        [
            "sudo",
            "curl",
            "-fsSL",
            "https://download.docker.com/linux/ubuntu/gpg",
            "-o",
            "/etc/apt/keyrings/docker.asc",
        ],
    )
    run_with_retry(
        ["sudo", "chmod", "a+r", "/etc/apt/keyrings/docker.asc"],
    )

    # Build the apt source line. Requires dpkg and /etc/os-release.
    arch_result = subprocess.run(
        ["dpkg", "--print-architecture"],
        capture_output=True,
        text=True,
        check=True,
    )
    arch = arch_result.stdout.strip()

    codename_result = subprocess.run(
        ["bash", "-c", ". /etc/os-release && echo $VERSION_CODENAME"],
        capture_output=True,
        text=True,
        check=True,
    )
    codename = codename_result.stdout.strip()

    source_line = (
        f"deb [arch={arch} "
        f"signed-by=/etc/apt/keyrings/docker.asc] "
        f"https://download.docker.com/linux/ubuntu "
        f"{codename} stable"
    )
    subprocess.run(
        ["sudo", "tee", "/etc/apt/sources.list.d/docker.list"],
        input=source_line,
        capture_output=True,
        text=True,
        check=True,
    )
    logger.info("Docker GPG keyring and repository setup completed.")


def install_nvidia_toolkit() -> None:
    """Add NVIDIA repo, GPG key, and install nvidia-container-toolkit."""
    logger.info("Installing NVIDIA Container Toolkit...")

    # Download NVIDIA GPG key.
    gpg_key_result = subprocess.run(
        [
            "curl",
            "-fsSL",
            "https://nvidia.github.io/libnvidia-container/gpgkey",
        ],
        capture_output=True,
        check=True,
    )
    subprocess.run(
        [
            "sudo",
            "gpg",
            "--dearmor",
            "--yes",
            "-o",
            "/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg",
        ],
        input=gpg_key_result.stdout,
        capture_output=True,
        check=True,
    )

    # Add NVIDIA repository with signed-by.
    repo_list_result = subprocess.run(
        [
            "curl",
            "-s",
            "-L",
            "https://nvidia.github.io/libnvidia-container/stable/deb/"
            "nvidia-container-toolkit.list",
        ],
        capture_output=True,
        text=True,
        check=True,
    )
    # Replace unsigned repo line with signed version.
    signed_list = repo_list_result.stdout.replace(
        "deb https://",
        "deb [signed-by=/usr/share/keyrings/"
        "nvidia-container-toolkit-keyring.gpg] https://",
    )
    subprocess.run(
        [
            "sudo",
            "tee",
            "/etc/apt/sources.list.d/nvidia-container-toolkit.list",
        ],
        input=signed_list,
        capture_output=True,
        text=True,
        check=True,
    )

    # Import GPG key.
    run_with_retry(
        [
            "sudo",
            "apt-key",
            "adv",
            "--keyserver",
            "keyserver.ubuntu.com",
            "--recv-keys",
            "DDCAE044F796ECB0",
        ],
    )

    run_with_retry(["sudo", "apt-get", "update"])
    run_with_retry(
        ["sudo", "apt-get", "install", "-y", "nvidia-container-toolkit"],
    )
    verify_nvidia_environment()


def install_docker() -> None:
    """Install Docker CE with NVIDIA runtime support.

    Installs docker-ce, docker-ce-cli, containerd.io, and
    docker-compose-plugin. Configures NVIDIA container runtime,
    adds the current user to the docker group, restarts the
    service, and runs a hello-world verification.
    """
    logger.info("Installing and configuring Docker...")
    run_with_retry(["sudo", "apt-get", "update"])
    run_with_retry(
        [
            "sudo",
            "apt-get",
            "install",
            "-y",
            "docker-ce",
            "docker-ce-cli",
            "containerd.io",
            "docker-compose-plugin",
        ],
    )
    run_with_retry(
        ["sudo", "nvidia-ctk", "runtime", "configure", "--runtime=docker"],
    )

    # Create docker group if it does not exist.
    group_check = subprocess.run(
        ["getent", "group", "docker"],
        capture_output=True,
        text=True,
        check=False,
    )
    if group_check.returncode != 0:
        run_with_retry(["sudo", "groupadd", "docker"])
    else:
        logger.info("Group 'docker' already exists. Skipping creation.")

    # Get current username without using $USER shell variable.
    user_result = subprocess.run(
        ["whoami"],
        capture_output=True,
        text=True,
        check=True,
    )
    username = user_result.stdout.strip()

    run_with_retry(["sudo", "usermod", "-aG", "docker", username])
    run_with_retry(["sudo", "systemctl", "restart", "docker"])
    run_with_retry(["sudo", "docker", "run", "hello-world"])
    logger.info("Docker installation and configuration completed.")


def install_ollama() -> None:
    """Install Ollama if not already present.

    Checks if the ollama binary is on PATH. If not, downloads and
    runs the official install script.
    """
    if shutil.which("ollama") is not None:
        logger.info("Ollama is already installed.")
        return

    logger.info("Installing Ollama...")
    # Download installer, then pipe to sh.
    installer = subprocess.run(
        ["curl", "-fsSL", "https://ollama.com/install.sh"],
        capture_output=True,
        check=True,
    )
    result = subprocess.run(
        ["sh"],
        input=installer.stdout,
        capture_output=True,
        check=False,
    )
    if result.returncode != 0:
        logger.error("Ollama installation failed.")
        logger.error("stderr: %s", result.stderr.decode())
        msg = "Ollama installation failed"
        raise RuntimeError(msg)

    if shutil.which("ollama") is None:
        msg = "Ollama installation completed but 'ollama' not found on PATH"
        raise RuntimeError(msg)

    logger.info("Ollama installed.")


def ensure_port_available(port: int) -> None:
    """Check if a port is in use and kill the occupying process.

    Args:
        port: TCP port number to check.
    """
    logger.info("Checking if port %d is available...", port)
    result = subprocess.run(
        ["sudo", "lsof", f"-ti:{port}"],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode == 0 and result.stdout.strip():
        pid = result.stdout.strip()
        logger.warning(
            "Port %d is in use by PID %s. Stopping process...",
            port,
            pid,
        )
        run_with_retry(["sudo", "kill", "-9", pid])
        logger.info("Freed up port %d.", port)
    else:
        logger.info("Port %d is available.", port)


def verify_docker_environment() -> None:
    """Verify Docker group membership, service status, and version.

    Raises:
        RuntimeError: If the user is not in the docker group or
            Docker is not running.
    """
    logger.info("Checking docker group membership...")
    groups_result = subprocess.run(
        ["id", "-nG"],
        capture_output=True,
        text=True,
        check=True,
    )
    if "docker" not in groups_result.stdout.split():
        msg = (
            "Current shell does not reflect docker group membership. "
            "Re-run the setup script after logging out and back in."
        )
        raise RuntimeError(msg)

    logger.info("Checking Docker service...")
    status_result = subprocess.run(
        ["systemctl", "is-active", "--quiet", "docker"],
        capture_output=True,
        check=False,
    )
    if status_result.returncode != 0:
        msg = "Docker is not running. Please start Docker."
        raise RuntimeError(msg)

    logger.info("Verifying Docker environment...")
    run_with_retry(["docker", "--version"])
    run_with_retry(["docker", "context", "ls"])


def verify_nvidia_environment() -> None:
    """Run nvidia-smi to verify NVIDIA drivers are accessible."""
    logger.info("Verifying NVIDIA environment...")
    run_with_retry(["nvidia-smi"])
