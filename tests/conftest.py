"""Shared fixtures for the owui test suite."""

from __future__ import annotations

from pathlib import Path
from typing import Any
from unittest.mock import MagicMock, patch

import pytest


@pytest.fixture
def sample_config() -> dict[str, Any]:
    """Return a dict matching config.toml structure."""
    return {
        "ollama": {
            "host": "localhost",
            "port": 11434,
            "container_tag": "latest",
            "container_name": "ollama",
            "volume_name": "ollama",
            "image": "ollama/ollama",
        },
        "openwebui": {
            "host": "localhost",
            "port": 3000,
            "container_tag": "latest",
            "container_name": "open-webui",
            "volume_name": "open-webui",
            "image": "ghcr.io/open-webui/open-webui",
        },
        "models": {
            "default": ["llama3.2:1b"],
        },
    }


@pytest.fixture
def tmp_config_file(tmp_path: Path, sample_config: dict[str, Any]) -> Path:
    """Write a config.toml to tmp_path and return the path."""
    config_path = tmp_path / "config.toml"
    lines = [
        "[ollama]",
        f'host = "{sample_config["ollama"]["host"]}"',
        f"port = {sample_config['ollama']['port']}",
        f'container_tag = "{sample_config["ollama"]["container_tag"]}"',
        f'container_name = "{sample_config["ollama"]["container_name"]}"',
        f'volume_name = "{sample_config["ollama"]["volume_name"]}"',
        f'image = "{sample_config["ollama"]["image"]}"',
        "",
        "[openwebui]",
        f'host = "{sample_config["openwebui"]["host"]}"',
        f"port = {sample_config['openwebui']['port']}",
        f'container_tag = "{sample_config["openwebui"]["container_tag"]}"',
        f'container_name = "{sample_config["openwebui"]["container_name"]}"',
        f'volume_name = "{sample_config["openwebui"]["volume_name"]}"',
        f'image = "{sample_config["openwebui"]["image"]}"',
        "",
        "[models]",
        'default = ["llama3.2:1b"]',
        "",
    ]
    config_path.write_text("\n".join(lines))
    return config_path


@pytest.fixture
def mock_subprocess() -> Any:
    """Patch subprocess.run and return the mock."""
    with patch("subprocess.run") as mock_run:
        mock_run.return_value = MagicMock(
            returncode=0,
            stdout="",
            stderr="",
        )
        yield mock_run
