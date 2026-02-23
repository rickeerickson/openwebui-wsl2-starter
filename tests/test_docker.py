"""Tests for owui.docker module."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest

from owui import docker


class TestContainerExists:
    """Tests for container_exists()."""

    @patch("owui.docker.subprocess.run")
    def test_container_exists_true(self, mock_run: MagicMock) -> None:
        """Returns True when docker ps -a output contains the container name."""
        mock_run.return_value = MagicMock(returncode=0, stdout="ollama\n", stderr="")
        assert docker.container_exists("ollama") is True

    @patch("owui.docker.subprocess.run")
    def test_container_exists_false(self, mock_run: MagicMock) -> None:
        """Returns False when docker ps -a output is empty."""
        mock_run.return_value = MagicMock(returncode=0, stdout="", stderr="")
        assert docker.container_exists("ollama") is False


class TestContainerIsRunning:
    """Tests for container_is_running()."""

    @patch("owui.docker.subprocess.run")
    def test_container_is_running_true(self, mock_run: MagicMock) -> None:
        """Returns True when docker ps output contains the container."""
        mock_run.return_value = MagicMock(returncode=0, stdout="ollama\n", stderr="")
        assert docker.container_is_running("ollama") is True

    @patch("owui.docker.subprocess.run")
    def test_container_is_running_false(self, mock_run: MagicMock) -> None:
        """Returns False when docker ps output is empty."""
        mock_run.return_value = MagicMock(returncode=0, stdout="", stderr="")
        assert docker.container_is_running("ollama") is False


class TestStopContainer:
    """Tests for stop_container()."""

    @patch("owui.docker.wait_for_container_stop")
    @patch("owui.docker.run_with_retry")
    @patch("owui.docker.container_is_running")
    @patch("owui.docker.container_exists")
    def test_stop_container_running(
        self,
        mock_exists: MagicMock,
        mock_running: MagicMock,
        mock_retry: MagicMock,
        mock_wait_stop: MagicMock,
    ) -> None:
        """Calls run_with_retry with docker stop when container is running."""
        mock_exists.return_value = True
        mock_running.return_value = True
        docker.stop_container("ollama")
        mock_retry.assert_called_once_with(["docker", "stop", "ollama"])
        mock_wait_stop.assert_called_once_with("ollama")

    @patch("owui.docker.run_with_retry")
    @patch("owui.docker.container_is_running")
    @patch("owui.docker.container_exists")
    def test_stop_container_not_exists(
        self,
        mock_exists: MagicMock,
        mock_running: MagicMock,
        mock_retry: MagicMock,
    ) -> None:
        """Does not call docker stop when container does not exist."""
        mock_exists.return_value = False
        docker.stop_container("ollama")
        mock_retry.assert_not_called()
        mock_running.assert_not_called()

    @patch("owui.docker.run_with_retry")
    @patch("owui.docker.container_is_running")
    @patch("owui.docker.container_exists")
    def test_stop_container_not_running(
        self,
        mock_exists: MagicMock,
        mock_running: MagicMock,
        mock_retry: MagicMock,
    ) -> None:
        """Does not call docker stop when container exists but is not running."""
        mock_exists.return_value = True
        mock_running.return_value = False
        docker.stop_container("ollama")
        mock_retry.assert_not_called()


class TestRemoveContainer:
    """Tests for remove_container()."""

    @patch("owui.docker.run_with_retry")
    @patch("owui.docker.container_exists")
    @patch("owui.docker.stop_container")
    def test_remove_container(
        self,
        mock_stop: MagicMock,
        mock_exists: MagicMock,
        mock_retry: MagicMock,
    ) -> None:
        """Calls stop_container, then run_with_retry with docker rm -f."""
        mock_exists.return_value = True
        docker.remove_container("ollama")
        mock_stop.assert_called_once_with("ollama")
        mock_retry.assert_called_once_with(["docker", "rm", "-f", "ollama"])

    @patch("owui.docker.run_with_retry")
    @patch("owui.docker.container_exists")
    @patch("owui.docker.stop_container")
    def test_remove_container_not_exists(
        self,
        mock_stop: MagicMock,
        mock_exists: MagicMock,
        mock_retry: MagicMock,
    ) -> None:
        """Skips rm when container does not exist after stop."""
        mock_exists.return_value = False
        docker.remove_container("ollama")
        mock_stop.assert_called_once_with("ollama")
        mock_retry.assert_not_called()


class TestStopAndRemove:
    """Tests for stop_and_remove()."""

    @patch("owui.docker.remove_container")
    @patch("owui.docker.wait_for_container_stop")
    @patch("owui.docker.run_with_retry")
    @patch("owui.docker.container_is_running")
    @patch("owui.docker.container_exists")
    def test_stop_and_remove_running(
        self,
        mock_exists: MagicMock,
        mock_running: MagicMock,
        mock_retry: MagicMock,
        mock_wait_stop: MagicMock,
        mock_remove: MagicMock,
    ) -> None:
        """Stops running container then removes it."""
        mock_exists.return_value = True
        mock_running.return_value = True
        docker.stop_and_remove("ollama")
        mock_retry.assert_called_once_with(["docker", "stop", "ollama"])
        mock_wait_stop.assert_called_once_with("ollama")
        mock_remove.assert_called_once_with("ollama")

    @patch("owui.docker.remove_container")
    @patch("owui.docker.run_with_retry")
    @patch("owui.docker.container_is_running")
    @patch("owui.docker.container_exists")
    def test_stop_and_remove_not_exists(
        self,
        mock_exists: MagicMock,
        mock_running: MagicMock,
        mock_retry: MagicMock,
        mock_remove: MagicMock,
    ) -> None:
        """Skips stop and remove when container does not exist."""
        mock_exists.return_value = False
        docker.stop_and_remove("ollama")
        mock_running.assert_not_called()
        mock_retry.assert_not_called()
        mock_remove.assert_not_called()


class TestPullImage:
    """Tests for pull_image()."""

    @patch("owui.docker.run_with_retry")
    def test_pull_image(self, mock_retry: MagicMock) -> None:
        """Calls run_with_retry with docker pull image:tag."""
        docker.pull_image("ollama/ollama", "latest")
        mock_retry.assert_called_once_with(["docker", "pull", "ollama/ollama:latest"])

    @patch("owui.docker.run_with_retry")
    def test_pull_image_default_tag(self, mock_retry: MagicMock) -> None:
        """Uses 'latest' as default tag."""
        docker.pull_image("ollama/ollama")
        mock_retry.assert_called_once_with(["docker", "pull", "ollama/ollama:latest"])


class TestWaitForContainerUp:
    """Tests for wait_for_container_up()."""

    @patch("owui.docker.time.sleep")
    @patch("owui.docker._get_container_status")
    def test_wait_for_container_up_immediate(
        self, mock_status: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Returns immediately when container status starts with 'Up'."""
        mock_status.return_value = "Up 3 minutes"
        docker.wait_for_container_up("ollama")
        mock_sleep.assert_not_called()

    @patch("owui.docker.time.sleep")
    @patch("owui.docker._get_container_status")
    def test_wait_for_container_up_after_retries(
        self, mock_status: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Retries until container status starts with 'Up'."""
        mock_status.side_effect = ["", "", "Up 1 minute"]
        docker.wait_for_container_up("ollama", max_retries=5)
        assert mock_sleep.call_count == 2

    @patch("owui.docker.time.sleep")
    @patch("owui.docker._get_container_status")
    def test_wait_for_container_up_timeout(
        self, mock_status: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Raises RuntimeError when container never comes up."""
        mock_status.return_value = ""
        with pytest.raises(RuntimeError, match="did not reach"):
            docker.wait_for_container_up("ollama", max_retries=3)
