"""Tests for owui.ollama module."""

from __future__ import annotations

from typing import Any
from unittest.mock import MagicMock, patch

from owui import ollama


class TestEnsureRunning:
    """Tests for ensure_running()."""

    @patch("owui.ollama.subprocess.run")
    @patch("owui.ollama.container_is_running")
    def test_ensure_running_already_running(
        self,
        mock_running: MagicMock,
        mock_run: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """No docker run when container is already running."""
        mock_running.return_value = True
        ollama.ensure_running(sample_config["ollama"])
        mock_run.assert_not_called()

    @patch("owui.ollama.wait_for_container_up")
    @patch("owui.ollama.subprocess.run")
    @patch("owui.ollama.container_is_running")
    def test_ensure_running_not_running(
        self,
        mock_running: MagicMock,
        mock_run: MagicMock,
        mock_wait: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """Runs container with docker run when not running."""
        mock_running.return_value = False
        mock_run.return_value = MagicMock(returncode=0, stdout="", stderr="")
        ollama.ensure_running(sample_config["ollama"])
        mock_run.assert_called_once()
        args = mock_run.call_args.args[0]
        assert "docker" in args
        assert "run" in args
        assert "--name" in args
        assert "ollama" in args
        assert "--gpus" in args
        mock_wait.assert_called_once_with("ollama")


class TestStopRemoveRun:
    """Tests for stop_remove_run()."""

    @patch("owui.ollama.wait_for_container_up")
    @patch("owui.ollama.ensure_running")
    @patch("owui.ollama.stop_and_remove")
    def test_stop_remove_run(
        self,
        mock_stop_remove: MagicMock,
        mock_ensure: MagicMock,
        mock_wait: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """Calls stop_and_remove, ensure_running, then wait_for_container_up."""
        ollama.stop_remove_run(sample_config["ollama"])
        mock_stop_remove.assert_called_once_with("ollama")
        mock_ensure.assert_called_once_with(sample_config["ollama"])
        mock_wait.assert_called_once_with("ollama")


class TestPullModels:
    """Tests for pull_models()."""

    @patch("owui.ollama.run_with_retry")
    @patch("owui.ollama.subprocess.run")
    def test_pull_models_new(
        self,
        mock_run: MagicMock,
        mock_retry: MagicMock,
    ) -> None:
        """Calls run_with_retry for each model to pull."""
        mock_run.return_value = MagicMock(returncode=0, stdout="", stderr="")
        ollama.pull_models(["llama3.2:1b"])
        mock_run.assert_called_once()  # ollama list
        mock_retry.assert_called_once_with(["ollama", "pull", "llama3.2:1b"])

    @patch("owui.ollama.run_with_retry")
    @patch("owui.ollama.subprocess.run")
    def test_pull_models_already_installed(
        self,
        mock_run: MagicMock,
        mock_retry: MagicMock,
    ) -> None:
        """Still pulls models even if already installed (merges lists)."""
        mock_run.return_value = MagicMock(
            returncode=0,
            stdout="NAME\tSIZE\nllama3.2:1b\t1.3GB\n",
            stderr="",
        )
        ollama.pull_models(["llama3.2:1b"])
        mock_run.assert_called_once()  # ollama list
        # Model appears in both requested and installed, still pulled
        mock_retry.assert_called_once_with(["ollama", "pull", "llama3.2:1b"])


class TestVerifySetup:
    """Tests for verify_setup()."""

    @patch("owui.ollama.subprocess.run")
    @patch("owui.ollama.run_with_retry")
    def test_verify_setup(
        self,
        mock_retry: MagicMock,
        mock_run: MagicMock,
    ) -> None:
        """Calls run_with_retry for curl, ollama list, ollama ps.

        Also calls subprocess.run for ss and docker logs.
        """
        mock_run.return_value = MagicMock(returncode=0, stdout="11434", stderr="")
        ollama.verify_setup("localhost", 11434)
        # 3 run_with_retry: curl, ollama list, ollama ps
        assert mock_retry.call_count == 3
        curl_cmd = mock_retry.call_args_list[0].args[0]
        assert "curl" in curl_cmd
        # 2 subprocess.run: ss -tuln, docker logs
        assert mock_run.call_count == 2
