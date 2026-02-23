"""Leveled logger mirroring the Bash log_message system.

Bash levels: ERROR=0, WARNING=1, INFO=2, DEBUG1=3, DEBUG2=4.
Maps to Python logging: ERROR, WARNING, INFO, DEBUG (for both
DEBUG1 and DEBUG2, with a custom filter for DEBUG2).
"""

from __future__ import annotations

import logging
import sys
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from pathlib import Path

# Bash-compatible verbosity levels.
LEVEL_ERROR = 0
LEVEL_WARNING = 1
LEVEL_INFO = 2
LEVEL_DEBUG_1 = 3
LEVEL_DEBUG_2 = 4

# Custom log level for DEBUG2 (below standard DEBUG=10).
DEBUG2 = logging.DEBUG - 1
logging.addLevelName(DEBUG2, "DEBUG2")

# Map bash verbosity integers to Python log levels.
_VERBOSITY_TO_LOG_LEVEL: dict[int, int] = {
    LEVEL_ERROR: logging.ERROR,
    LEVEL_WARNING: logging.WARNING,
    LEVEL_INFO: logging.INFO,
    LEVEL_DEBUG_1: logging.DEBUG,
    LEVEL_DEBUG_2: DEBUG2,
}

_LOG_FORMAT = "%(asctime)s - %(levelname)-8s %(message)s"
_LOG_DATEFMT = "%Y.%m.%d:%H:%M:%S"


def setup_logger(
    name: str,
    verbosity: int = LEVEL_INFO,
    log_file: Path | None = None,
) -> logging.Logger:
    """Configure and return a logger.

    Args:
        name: Logger name, typically __name__ or "owui".
        verbosity: Bash-compatible verbosity level (0-4).
        log_file: Optional file path to write logs to.

    Returns:
        Configured logger instance.
    """
    logger = logging.getLogger(name)

    # Clamp verbosity to valid range.
    verbosity = max(LEVEL_ERROR, min(verbosity, LEVEL_DEBUG_2))
    log_level = _VERBOSITY_TO_LOG_LEVEL.get(verbosity, logging.INFO)
    logger.setLevel(log_level)

    # Avoid duplicate handlers on repeated calls.
    if logger.handlers:
        logger.handlers.clear()

    formatter = logging.Formatter(_LOG_FORMAT, datefmt=_LOG_DATEFMT)

    stderr_handler = logging.StreamHandler(sys.stderr)
    stderr_handler.setFormatter(formatter)
    stderr_handler.setLevel(log_level)
    logger.addHandler(stderr_handler)

    if log_file is not None:
        file_handler = logging.FileHandler(log_file)
        file_handler.setFormatter(formatter)
        file_handler.setLevel(log_level)
        logger.addHandler(file_handler)

    return logger


def get_logger(name: str = "owui") -> logging.Logger:
    """Return an existing logger by name.

    If the logger has no handlers, sets up a default INFO-level logger.

    Args:
        name: Logger name.

    Returns:
        Logger instance.
    """
    logger = logging.getLogger(name)
    if not logger.handlers:
        return setup_logger(name)
    return logger
