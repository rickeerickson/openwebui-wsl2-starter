"""Docker container lifecycle management.

Ports the container management functions from bash/common/repo_lib.sh
(lines 325-534). Each function checks state before acting to remain
idempotent.
"""

from __future__ import annotations

import subprocess
import time
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from collections.abc import Iterator

from owui.log import get_logger
from owui.retry import run_with_retry

logger = get_logger(__name__)


def _fibonacci_poll_delays(
    initial: int = 10,
    max_retries: int = 5,
) -> Iterator[int]:
    """Yield Fibonacci delays for polling loops."""
    fib1 = initial
    fib2 = initial
    for _ in range(max_retries):
        yield fib1
        fib1, fib2 = fib2, fib1 + fib2


def container_exists(name: str) -> bool:
    """Check if a container exists (running or stopped).

    Args:
        name: Container name.

    Returns:
        True if the container exists.
    """
    result = subprocess.run(
        [
            "docker",
            "ps",
            "-a",
            "--filter",
            f"name=^{name}$",
            "--format",
            "{{.Names}}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    return name in result.stdout.splitlines()


def container_is_running(name: str) -> bool:
    """Check if a container is currently running.

    Args:
        name: Container name.

    Returns:
        True if the container is running.
    """
    result = subprocess.run(
        [
            "docker",
            "ps",
            "--filter",
            f"name=^{name}$",
            "--format",
            "{{.Names}}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    return name in result.stdout.splitlines()


def _get_container_status(name: str) -> str:
    """Get the status string for a container.

    Args:
        name: Container name.

    Returns:
        Status string (e.g. "Up 3 minutes") or empty string.
    """
    result = subprocess.run(
        [
            "docker",
            "ps",
            "--filter",
            f"name={name}",
            "--format",
            "{{.Status}}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    lines = result.stdout.strip().splitlines()
    if lines:
        return lines[0]
    return ""


def _container_is_exited(name: str) -> bool:
    """Check if a container is in the exited state.

    Args:
        name: Container name.

    Returns:
        True if the container exists and is exited.
    """
    result = subprocess.run(
        [
            "docker",
            "ps",
            "-a",
            "--filter",
            f"name=^{name}$",
            "--filter",
            "status=exited",
            "--format",
            "{{.Names}}",
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    return name in result.stdout.splitlines()


def wait_for_container_up(name: str, max_retries: int = 5) -> None:
    """Poll until a container's status starts with "Up".

    Args:
        name: Container name.
        max_retries: Maximum number of poll attempts.

    Raises:
        RuntimeError: If the container does not come up within
            max_retries.
    """
    logger.info("Waiting for container %r to start...", name)
    for attempt, delay in enumerate(_fibonacci_poll_delays(10, max_retries)):
        status = _get_container_status(name)
        if status.startswith("Up"):
            logger.info(
                "Container %r is running with status %r.",
                name,
                status,
            )
            return
        logger.warning(
            "Waiting for container %r to start. Current status: %r. Retry %d/%d",
            name,
            status,
            attempt + 1,
            max_retries,
        )
        time.sleep(delay)

    msg = f"Container {name!r} did not reach 'Up' status after {max_retries} retries"
    raise RuntimeError(msg)


def wait_for_container_stop(name: str, max_retries: int = 5) -> None:
    """Poll until a container reaches the exited state.

    Args:
        name: Container name.
        max_retries: Maximum number of poll attempts.

    Raises:
        RuntimeError: If the container does not stop within
            max_retries.
    """
    logger.info("Waiting for container %r to stop...", name)
    for attempt, delay in enumerate(_fibonacci_poll_delays(10, max_retries)):
        if _container_is_exited(name):
            logger.info(
                "Container %r has stopped and is in exited state.",
                name,
            )
            return
        logger.warning(
            "Waiting for container %r to stop... Retry %d/%d",
            name,
            attempt + 1,
            max_retries,
        )
        time.sleep(delay)

    msg = f"Container {name!r} did not stop after {max_retries} retries"
    raise RuntimeError(msg)


def stop_container(name: str) -> None:
    """Stop a running container.

    Checks existence and running state before issuing stop. Waits
    for exited state after stopping.

    Args:
        name: Container name.
    """
    logger.info("Stopping container %r...", name)
    if not container_exists(name):
        logger.warning(
            "Container %r does not exist. Skipping stop.",
            name,
        )
        return
    if not container_is_running(name):
        logger.warning(
            "Container %r is not running. Skipping stop.",
            name,
        )
        return
    run_with_retry(["docker", "stop", name])
    wait_for_container_stop(name)


def remove_container(name: str) -> None:
    """Stop (if running) then remove a container.

    Args:
        name: Container name.
    """
    logger.info("Removing container %r...", name)
    stop_container(name)
    if not container_exists(name):
        logger.warning(
            "Container %r not found. Skipping remove.",
            name,
        )
        return
    run_with_retry(["docker", "rm", "-f", name])
    logger.info("Container %r removed.", name)


def stop_and_remove(name: str) -> None:
    """Stop and remove a container (orchestrator).

    Args:
        name: Container name.
    """
    logger.info("Stopping and removing container %r...", name)
    if not container_exists(name):
        logger.warning(
            "Container %r does not exist. Skipping stop and remove.",
            name,
        )
        return
    if container_is_running(name):
        run_with_retry(["docker", "stop", name])
        wait_for_container_stop(name)
    remove_container(name)


def pull_image(image: str, tag: str = "latest") -> None:
    """Pull a Docker image with retry.

    Args:
        image: Image name (e.g. "ollama/ollama").
        tag: Image tag. Defaults to "latest".
    """
    full_name = f"{image}:{tag}"
    logger.info("Pulling Docker image: %s...", full_name)
    run_with_retry(["docker", "pull", full_name])
    logger.info("Docker image %s pulled.", full_name)
