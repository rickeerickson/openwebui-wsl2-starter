"""Fibonacci retry logic mirroring bash run_command_with_retry.

Delays: 10, 10, 20, 30, 50, 80 seconds (Fibonacci sequence starting
from initial_delay).
"""

from __future__ import annotations

import subprocess
import time
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from collections.abc import Iterator

from owui.log import get_logger

logger = get_logger(__name__)


def fibonacci_delays(
    initial: int = 10,
    max_retries: int = 5,
) -> Iterator[int]:
    """Yield Fibonacci-spaced delays.

    With defaults, yields: 10, 10, 20, 30, 50, 80.

    Args:
        initial: Starting delay in seconds.
        max_retries: Number of delays to yield.

    Yields:
        Delay in seconds.
    """
    fib1 = initial
    fib2 = initial
    for _ in range(max_retries + 1):
        yield fib1
        fib1, fib2 = fib2, fib1 + fib2


def run_with_retry(
    cmd: list[str],
    *,
    max_retries: int = 5,
    initial_delay: int = 10,
    check: bool = True,
) -> subprocess.CompletedProcess[str]:
    """Run a command with Fibonacci backoff retries.

    Args:
        cmd: Command and arguments as a list.
        max_retries: Maximum number of retry attempts.
        initial_delay: Initial delay in seconds for backoff.
        check: If True, raise CalledProcessError on non-zero exit
               after all retries are exhausted. If False, return
               the result regardless.

    Returns:
        CompletedProcess from the successful (or final) run.

    Raises:
        subprocess.CalledProcessError: If all retries fail and
            check is True.
    """
    delays = fibonacci_delays(initial_delay, max_retries)
    last_result: subprocess.CompletedProcess[str] | None = None
    cmd_str = " ".join(cmd)

    for attempt, delay in enumerate(delays):
        if attempt > 0:
            logger.warning(
                "Retrying command %r in %d seconds (attempt %d/%d)...",
                cmd_str,
                delay,
                attempt,
                max_retries,
            )
            time.sleep(delay)

        logger.info("Executing: %s", cmd_str)
        last_result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            check=False,
        )

        if last_result.returncode == 0:
            if last_result.stdout:
                for line in last_result.stdout.splitlines():
                    logger.info("  %s", line)
            return last_result

        logger.warning(
            "Command %r failed with exit code %d",
            cmd_str,
            last_result.returncode,
        )
        if last_result.stderr:
            for line in last_result.stderr.splitlines():
                logger.warning("  %s", line)

    # All retries exhausted.
    assert last_result is not None  # noqa: S101
    if check:
        logger.error(
            "Failed to execute %r after %d attempts",
            cmd_str,
            max_retries + 1,
        )
        raise subprocess.CalledProcessError(
            last_result.returncode,
            cmd,
            last_result.stdout,
            last_result.stderr,
        )

    return last_result
