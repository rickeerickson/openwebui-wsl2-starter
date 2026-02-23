"""Tests for owui.openwebui module."""

from __future__ import annotations

from typing import Any
from unittest.mock import MagicMock, patch

from owui import openwebui


class TestEnsureRunning:
    """Tests for ensure_running()."""

    @patch("owui.openwebui.subprocess.run")
    @patch("owui.openwebui.container_is_running")
    def test_ensure_running_already_running(
        self,
        mock_running: MagicMock,
        mock_run: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """No docker run when container is already running."""
        mock_running.return_value = True
        openwebui.ensure_running(sample_config["openwebui"], sample_config["ollama"])
        mock_run.assert_not_called()

    @patch("owui.openwebui.wait_for_container_up")
    @patch("owui.openwebui.subprocess.run")
    @patch("owui.openwebui.container_is_running")
    def test_ensure_running_not_running(
        self,
        mock_running: MagicMock,
        mock_run: MagicMock,
        mock_wait: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """Runs container with OLLAMA_BASE_URL env when not running."""
        mock_running.return_value = False
        mock_run.return_value = MagicMock(returncode=0, stdout="", stderr="")
        openwebui.ensure_running(sample_config["openwebui"], sample_config["ollama"])
        mock_run.assert_called_once()
        args = mock_run.call_args.args[0]
        assert "docker" in args
        assert "run" in args
        # Check OLLAMA_BASE_URL is in the args
        assert "OLLAMA_BASE_URL=http://localhost:11434" in args
        mock_wait.assert_called_once_with("open-webui")


class TestStopRemoveRun:
    """Tests for stop_remove_run()."""

    @patch("owui.openwebui.wait_for_container_up")
    @patch("owui.openwebui.ensure_running")
    @patch("owui.openwebui.stop_and_remove")
    def test_stop_remove_run(
        self,
        mock_stop_remove: MagicMock,
        mock_ensure: MagicMock,
        mock_wait: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """Calls stop_and_remove, ensure_running, then wait_for_container_up."""
        openwebui.stop_remove_run(sample_config["openwebui"], sample_config["ollama"])
        mock_stop_remove.assert_called_once_with("open-webui")
        mock_ensure.assert_called_once_with(
            sample_config["openwebui"], sample_config["ollama"]
        )
        mock_wait.assert_called_once_with("open-webui")


class TestVerifySetup:
    """Tests for verify_setup()."""

    @patch("owui.openwebui.time.sleep")
    @patch("owui.openwebui.subprocess.run")
    @patch("owui.openwebui.run_with_retry")
    def test_verify_setup(
        self,
        mock_retry: MagicMock,
        mock_run: MagicMock,
        mock_sleep: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """Polls for port via ss, then curls and checks docker logs."""
        # ss returns port 3000 immediately
        mock_run.return_value = MagicMock(returncode=0, stdout="3000", stderr="")
        openwebui.verify_setup(sample_config["openwebui"])
        # 1 run_with_retry call: curl
        mock_retry.assert_called_once()
        curl_cmd = mock_retry.call_args.args[0]
        assert "curl" in curl_cmd
        # subprocess.run calls: ss + docker logs
        assert mock_run.call_count == 2
        mock_sleep.assert_not_called()
