"""Tests for owui.config module."""

from __future__ import annotations

from pathlib import Path
from typing import Any

import pytest

from owui.config import get_config_value, load_config


class TestLoadConfig:
    """Tests for load_config()."""

    def test_load_config_default(self) -> None:
        """Loading config.toml from repo root returns a dict with expected sections."""
        cfg = load_config()
        assert "ollama" in cfg
        assert "openwebui" in cfg
        assert "models" in cfg

    def test_load_config_custom_path(self, tmp_config_file: Path) -> None:
        """Loading from a custom path returns valid config."""
        cfg = load_config(tmp_config_file)
        assert cfg["ollama"]["port"] == 11434
        assert cfg["openwebui"]["container_name"] == "open-webui"

    def test_load_config_missing_file(self, tmp_path: Path) -> None:
        """Loading a nonexistent file raises FileNotFoundError."""
        missing = tmp_path / "nonexistent.toml"
        with pytest.raises(FileNotFoundError):
            load_config(missing)


class TestGetConfigValue:
    """Tests for get_config_value()."""

    def test_get_config_value_simple(self, sample_config: dict[str, Any]) -> None:
        """Dotted key 'ollama.port' returns '11434' as a string."""
        result = get_config_value("ollama.port", sample_config)
        assert result == "11434"

    def test_get_config_value_nested(self, sample_config: dict[str, Any]) -> None:
        """Dotted key 'models.default' returns a string representation."""
        result = get_config_value("models.default", sample_config)
        assert isinstance(result, str)
        assert "llama3.2:1b" in result

    def test_get_config_value_missing_key(self, sample_config: dict[str, Any]) -> None:
        """Missing key raises KeyError."""
        with pytest.raises(KeyError):
            get_config_value("nonexistent.key", sample_config)
