"""Tests for owui.log module."""

from __future__ import annotations

import logging
from pathlib import Path

from owui.log import get_logger, setup_logger


class TestSetupLogger:
    """Tests for setup_logger()."""

    def test_setup_logger_default(self) -> None:
        """Default verbosity=2 sets INFO level."""
        logger = setup_logger("test_default")
        assert logger.level == logging.INFO

    def test_setup_logger_verbose(self) -> None:
        """Verbosity=3 sets DEBUG level."""
        logger = setup_logger("test_verbose", verbosity=3)
        assert logger.level == logging.DEBUG

    def test_setup_logger_file(self, tmp_path: Path) -> None:
        """Passing log_file creates a file handler."""
        log_file = tmp_path / "test.log"
        logger = setup_logger("test_file_handler", log_file=log_file)
        file_handlers = [
            h for h in logger.handlers if isinstance(h, logging.FileHandler)
        ]
        assert len(file_handlers) >= 1

    def test_setup_logger_error_level(self) -> None:
        """Verbosity=0 sets ERROR level."""
        logger = setup_logger("test_error", verbosity=0)
        assert logger.level == logging.ERROR


class TestGetLogger:
    """Tests for get_logger()."""

    def test_get_logger_returns_same(self) -> None:
        """get_logger returns the same logger instance for the same name."""
        setup_logger("test_same_instance")
        logger1 = get_logger("test_same_instance")
        logger2 = get_logger("test_same_instance")
        assert logger1 is logger2

    def test_log_format(self) -> None:
        """Logger formatter includes timestamp pattern."""
        logger = setup_logger("test_format_check")
        stream_handlers = [
            h for h in logger.handlers if isinstance(h, logging.StreamHandler)
        ]
        assert len(stream_handlers) >= 1
        formatter = stream_handlers[0].formatter
        assert formatter is not None
        assert "asctime" in (formatter._fmt or "")
