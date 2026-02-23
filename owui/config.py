"""TOML configuration loader.

Loads config.toml from the repo root and provides dot-notation access
to configuration values.
"""

from __future__ import annotations

import tomllib
from pathlib import Path
from typing import Any


def _find_repo_root() -> Path:
    """Walk up from this file's directory to find config.toml."""
    current = Path(__file__).resolve().parent
    while current != current.parent:
        if (current / "config.toml").exists():
            return current
        current = current.parent
    msg = "config.toml not found in any parent directory"
    raise FileNotFoundError(msg)


def load_config(path: Path | None = None) -> dict[str, Any]:
    """Load configuration from a TOML file.

    Args:
        path: Path to the config file. Defaults to config.toml in the
              repo root.

    Returns:
        Parsed configuration dictionary.
    """
    if path is None:
        path = _find_repo_root() / "config.toml"
    with path.open("rb") as f:
        return tomllib.load(f)


def get_config_value(key: str, config: dict[str, Any] | None = None) -> str:
    """Access a config value using dot-notation.

    Args:
        key: Dot-separated key path, e.g. "ollama.port" or
             "models.default".
        config: Configuration dict. If None, loads from default path.

    Returns:
        The config value as a string.

    Raises:
        KeyError: If the key path does not exist in the config.
    """
    if config is None:
        config = load_config()
    parts = key.split(".")
    current: Any = config
    for part in parts:
        if not isinstance(current, dict):
            msg = f"key path {key!r} is invalid: {part!r} is not a section"
            raise KeyError(msg)
        if part not in current:
            msg = f"key {key!r} not found (missing {part!r})"
            raise KeyError(msg)
        current = current[part]
    return str(current)
