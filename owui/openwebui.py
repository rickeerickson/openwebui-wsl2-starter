"""OpenWebUI container management.

Ports the OpenWebUI orchestration functions from
bash/common/repo_lib.sh (lines 600-703).
"""

from __future__ import annotations

import subprocess
import time
from typing import Any

from owui.docker import (
    container_is_running,
    stop_and_remove,
    wait_for_container_up,
)
from owui.log import get_logger
from owui.retry import run_with_retry

logger = get_logger(__name__)


def ensure_running(
    config: dict[str, Any],
    ollama_config: dict[str, Any],
) -> None:
    """Start the OpenWebUI container if not already running.

    Runs with --gpus all, --network=host, OLLAMA_BASE_URL env,
    and a named volume for backend data.

    Args:
        config: OpenWebUI config dict from
                config.load_config()["openwebui"].
        ollama_config: Ollama config dict for building the
                       OLLAMA_BASE_URL.
    """
    name = str(config["container_name"])
    tag = str(config["container_tag"])
    port = str(config["port"])
    volume = str(config["volume_name"])
    image = str(config.get("image", "ghcr.io/open-webui/open-webui"))

    ollama_host = str(ollama_config["host"])
    ollama_port = str(ollama_config["port"])
    ollama_url = f"http://{ollama_host}:{ollama_port}"

    if container_is_running(name):
        logger.info("OpenWebUI container is already running.")
        return

    logger.info("Starting OpenWebUI container...")
    result = subprocess.run(
        [
            "docker",
            "run",
            "-d",
            "--gpus",
            "all",
            "--network=host",
            "--volume",
            f"{volume}:/app/backend/data",
            "--env",
            f"OLLAMA_BASE_URL={ollama_url}",
            "--env",
            f"PORT={port}",
            "--name",
            name,
            "--restart",
            "always",
            f"{image}:{tag}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        logger.error("Failed to start OpenWebUI container: %s", result.stderr)
        msg = f"docker run failed with exit code {result.returncode}"
        raise RuntimeError(msg)

    wait_for_container_up(name)
    logger.info("OpenWebUI container started.")


def stop_remove_run(
    config: dict[str, Any],
    ollama_config: dict[str, Any],
) -> None:
    """Stop, remove, then start the OpenWebUI container.

    Args:
        config: OpenWebUI config dict.
        ollama_config: Ollama config dict.
    """
    name = str(config["container_name"])
    logger.info("Stopping and removing OpenWebUI container...")
    stop_and_remove(name)
    ensure_running(config, ollama_config)
    wait_for_container_up(name)
    logger.info("OpenWebUI container restarted.")


def verify_setup(config: dict[str, Any]) -> None:
    """Verify the OpenWebUI service is responding.

    Waits for the port to appear in ss output, then checks HTTP
    response and container logs.

    Args:
        config: OpenWebUI config dict.
    """
    host = str(config["host"])
    port = int(config["port"])
    name = str(config["container_name"])
    url = f"http://{host}:{port}"

    logger.info("Verifying OpenWebUI setup at %s...", url)

    # Wait for port to appear.
    max_retries = 5
    for attempt in range(max_retries):
        ss_result = subprocess.run(
            ["ss", "-tuln"],
            capture_output=True,
            text=True,
            check=False,
        )
        if str(port) in ss_result.stdout:
            break
        logger.info(
            "Waiting for OpenWebUI on port %d... Retry %d/%d",
            port,
            attempt + 1,
            max_retries,
        )
        time.sleep(2**attempt)
    else:
        msg = f"OpenWebUI is not listening on port {port} after {max_retries} attempts"
        raise RuntimeError(msg)

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
    subprocess.run(
        ["docker", "logs", name],
        capture_output=True,
        text=True,
        check=False,
    )
    logger.info("OpenWebUI setup verified.")
