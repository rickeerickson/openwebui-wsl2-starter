"""System diagnostics.

Ports the diagnostic functions from bash/common/repo_lib.sh
(lines 755-854). Runs system info, network, port, Docker,
and connectivity checks.
"""

from __future__ import annotations

import socket
import subprocess
from typing import Any

from owui.log import get_logger

logger = get_logger(__name__)


def _run_and_log(
    cmd: list[str],
    label: str,
) -> subprocess.CompletedProcess[str]:
    """Run a command and log its output.

    Args:
        cmd: Command as a list.
        label: Label for the output section.

    Returns:
        CompletedProcess result.
    """
    logger.info("=== %s ===", label)
    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        check=False,
    )
    if result.stdout:
        for line in result.stdout.splitlines():
            logger.info("  %s", line)
    if result.returncode != 0 and result.stderr:
        for line in result.stderr.splitlines():
            logger.warning("  %s", line)
    return result


def show_system_info() -> None:
    """Display basic system info: user, home, distro."""
    logger.info("=== Basic System Info ===")
    _run_and_log(["whoami"], "User")
    _run_and_log(["bash", "-c", "echo $HOME"], "Home")

    # Try lsb_release first, fall back to /etc/os-release.
    result = subprocess.run(
        ["lsb_release", "-a"],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode == 0:
        for line in result.stdout.splitlines():
            logger.info("  %s", line)
    else:
        _run_and_log(
            ["bash", "-c", "cat /etc/os-release"],
            "Distro Info",
        )


def show_network() -> None:
    """Display network interfaces and IP addresses."""
    _run_and_log(["ip", "addr", "show"], "Network Interfaces & IPs")


def show_listening_ports() -> None:
    """Display listening TCP ports via lsof."""
    _run_and_log(
        ["sudo", "lsof", "-i", "-P", "-n"],
        "Listening Ports (TCP)",
    )


def test_port(host: str, port: int) -> None:
    """Test TCP connectivity and HTTP response on a port.

    Args:
        host: Host to connect to.
        port: Port number.
    """
    logger.info("=== Test Port %d ===", port)

    # TCP connect test.
    try:
        with socket.create_connection((host, port), timeout=5):
            logger.info("TCP connection to %s:%d succeeded.", host, port)
    except OSError:
        logger.warning("TCP connection to %s:%d failed.", host, port)

    # HTTP check.
    result = subprocess.run(
        [
            "curl",
            "-s",
            "-o",
            "/dev/null",
            "--write-out",
            "%{http_code}",
            f"http://{host}:{port}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    code = result.stdout.strip()
    if code.isdigit():
        logger.info("HTTP response code on port %d: %s", port, code)
    else:
        logger.warning("No valid HTTP response on port %d", port)


def show_docker_info() -> None:
    """Display Docker version, running containers, and images."""
    logger.info("=== Docker Diagnostics ===")
    _run_and_log(["docker", "--version"], "Docker version")
    _run_and_log(["docker", "ps"], "Running containers")
    _run_and_log(["docker", "images"], "Docker images")


def check_container_logs(name: str) -> None:
    """Check a container's logs for error, warn, and listen lines.

    Args:
        name: Container name.
    """
    logger.info("=== %s Container Logs ===", name)

    # Verify container is running.
    ps_result = subprocess.run(
        [
            "docker",
            "ps",
            "--filter",
            f"name={name}",
            "--format",
            "{{.Names}}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if name not in ps_result.stdout.splitlines():
        logger.warning("Container %r is not running.", name)
        return

    logger.info(
        "Container %r is running. Checking logs for error/warn/listen...",
        name,
    )
    logs_result = subprocess.run(
        ["docker", "logs", name],
        capture_output=True,
        text=True,
        check=False,
    )
    all_output = logs_result.stdout + logs_result.stderr
    found = False
    for line in all_output.splitlines():
        lower = line.lower()
        if "error" in lower or "warn" in lower or "listen" in lower:
            logger.info("  %s", line)
            found = True
    if not found:
        logger.info("No matches for error|warn|listen in logs.")


def check_routing() -> None:
    """Display default routes and test external connectivity."""
    logger.info("=== Routing & Connectivity ===")
    _run_and_log(["ip", "route", "show", "default"], "Default routes")
    _run_and_log(
        ["ping", "-c", "4", "google.com"],
        "External connectivity",
    )


def run_all(config: dict[str, Any]) -> None:
    """Run all diagnostics.

    Args:
        config: Full configuration dict with "ollama" and "openwebui"
                sections.
    """
    show_system_info()
    show_network()
    show_listening_ports()

    ollama_cfg = config.get("ollama", {})
    openwebui_cfg = config.get("openwebui", {})

    if ollama_cfg:
        test_port(
            str(ollama_cfg.get("host", "localhost")),
            int(ollama_cfg.get("port", 11434)),
        )
    if openwebui_cfg:
        test_port(
            str(openwebui_cfg.get("host", "localhost")),
            int(openwebui_cfg.get("port", 3000)),
        )

    show_docker_info()
    check_container_logs(str(ollama_cfg.get("container_name", "ollama")))
    check_container_logs(
        str(openwebui_cfg.get("container_name", "open-webui")),
    )
    check_routing()
