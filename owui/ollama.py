"""Ollama container management.

Ports the Ollama orchestration functions from
bash/common/repo_lib.sh (lines 536-753).
"""

from __future__ import annotations

import subprocess
from typing import Any

from owui.docker import (
    container_is_running,
    stop_and_remove,
    wait_for_container_up,
)
from owui.log import get_logger
from owui.retry import run_with_retry

logger = get_logger(__name__)


def ensure_running(config: dict[str, Any]) -> None:
    """Start the Ollama container if not already running.

    Runs with --gpus all, --network=host, and a named volume
    for model storage.

    Args:
        config: Ollama config dict from config.load_config()["ollama"].
    """
    name = str(config["container_name"])
    tag = str(config["container_tag"])
    host = str(config["host"])
    volume = str(config["volume_name"])
    image = str(config.get("image", "ollama/ollama"))

    if container_is_running(name):
        logger.info("Ollama container is already running.")
        return

    logger.info("Starting Ollama container...")
    result = subprocess.run(
        [
            "docker",
            "run",
            "-d",
            "--gpus",
            "all",
            "--network=host",
            "--volume",
            f"{volume}:/root/.ollama",
            "--env",
            f"OLLAMA_HOST={host}",
            "--restart",
            "always",
            "--name",
            name,
            f"{image}:{tag}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        logger.error("Failed to start Ollama container: %s", result.stderr)
        msg = f"docker run failed with exit code {result.returncode}"
        raise RuntimeError(msg)

    wait_for_container_up(name)
    logger.info("Ollama container started.")


def stop_remove_run(config: dict[str, Any]) -> None:
    """Stop, remove, then start the Ollama container.

    Args:
        config: Ollama config dict.
    """
    name = str(config["container_name"])
    logger.info("Stopping and removing Ollama container...")
    stop_and_remove(name)
    ensure_running(config)
    wait_for_container_up(name)
    logger.info("Ollama container restarted.")


def verify_setup(host: str, port: int) -> None:
    """Verify the Ollama service is responding.

    Checks HTTP endpoint, lists models, checks port binding,
    and prints container logs.

    Args:
        host: Ollama host address.
        port: Ollama port number.
    """
    url = f"http://{host}:{port}"
    logger.info("Verifying Ollama setup at %s...", url)

    run_with_retry(
        [
            "curl",
            "-s",
            "-o",
            "/dev/null",
            "--write-out",
            "%{response_code}\\n",
            url,
        ],
    )
    run_with_retry(["ollama", "list"])
    run_with_retry(["ollama", "ps"])

    # Check port binding via ss.
    ss_result = subprocess.run(
        ["ss", "-tuln"],
        capture_output=True,
        text=True,
        check=False,
    )
    if str(port) in ss_result.stdout:
        logger.info("Port %d is bound.", port)
    else:
        logger.warning("Port %d not found in ss output.", port)

    # Show container logs.
    subprocess.run(
        ["docker", "logs", "ollama"],
        capture_output=True,
        text=True,
        check=False,
    )
    logger.info("Ollama setup verification completed.")


def pull_models(models: list[str]) -> None:
    """Pull Ollama models, merging with any already installed.

    Fetches the current list of installed models, merges with the
    requested list (deduplicating), and pulls each model.

    Args:
        models: List of model names to pull.
    """
    logger.info("Pulling Ollama models...")

    # Get installed models.
    installed_result = subprocess.run(
        ["ollama", "list"],
        capture_output=True,
        text=True,
        check=False,
    )
    installed: set[str] = set()
    if installed_result.returncode == 0:
        for line in installed_result.stdout.splitlines()[1:]:
            parts = line.split()
            if parts:
                installed.add(parts[0])

    # Merge requested and installed, preserving order.
    all_models: list[str] = list(models)
    for model in installed:
        if model not in all_models:
            all_models.append(model)

    for model in all_models:
        logger.info("Pulling model: %s", model)
        run_with_retry(["ollama", "pull", model])

    logger.info("Model pulling completed.")
