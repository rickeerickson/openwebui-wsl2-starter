"""Tests for owui.retry module."""

from __future__ import annotations

import subprocess
from unittest.mock import MagicMock, patch

import pytest

from owui.retry import fibonacci_delays, run_with_retry


class TestFibonacciDelays:
    """Tests for fibonacci_delays()."""

    def test_fibonacci_delays_default(self) -> None:
        """Default parameters yield 6 delays: [10, 10, 20, 30, 50, 80]."""
        delays = list(fibonacci_delays())
        assert delays == [10, 10, 20, 30, 50, 80]

    def test_fibonacci_delays_custom(self) -> None:
        """initial=5, default max_retries=5 yields [5, 5, 10, 15, 25, 40]."""
        delays = list(fibonacci_delays(initial=5))
        assert delays == [5, 5, 10, 15, 25, 40]

    def test_fibonacci_delays_single_retry(self) -> None:
        """max_retries=1 yields two delays."""
        delays = list(fibonacci_delays(max_retries=1))
        assert delays == [10, 10]

    def test_fibonacci_delays_zero_retries(self) -> None:
        """max_retries=0 yields a single delay (the initial attempt)."""
        delays = list(fibonacci_delays(max_retries=0))
        assert delays == [10]


class TestRunWithRetry:
    """Tests for run_with_retry()."""

    @patch("owui.retry.time.sleep")
    @patch("owui.retry.subprocess.run")
    def test_run_with_retry_success_first(
        self, mock_run: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """No retry on first success."""
        mock_run.return_value = MagicMock(returncode=0, stdout="ok", stderr="")
        result = run_with_retry(["echo", "hello"])
        assert result.returncode == 0
        mock_run.assert_called_once()
        mock_sleep.assert_not_called()

    @patch("owui.retry.time.sleep")
    @patch("owui.retry.subprocess.run")
    def test_run_with_retry_success_after_failure(
        self, mock_run: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Retries once then succeeds."""
        failure = MagicMock(returncode=1, stdout="", stderr="error")
        success = MagicMock(returncode=0, stdout="ok", stderr="")
        mock_run.side_effect = [failure, success]
        result = run_with_retry(["cmd"], max_retries=3)
        assert result.returncode == 0
        assert mock_run.call_count == 2
        mock_sleep.assert_called_once()

    @patch("owui.retry.time.sleep")
    @patch("owui.retry.subprocess.run")
    def test_run_with_retry_all_fail(
        self, mock_run: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Raises CalledProcessError after exhausting all retries.

        max_retries=3 means 4 total attempts (1 initial + 3 retries).
        Sleep is called between attempts, so 3 times.
        """
        mock_run.return_value = MagicMock(returncode=1, stdout="", stderr="err")
        with pytest.raises(subprocess.CalledProcessError):
            run_with_retry(["fail"], max_retries=3)
        assert mock_run.call_count == 4
        assert mock_sleep.call_count == 3

    @patch("owui.retry.time.sleep")
    @patch("owui.retry.subprocess.run")
    def test_run_with_retry_check_false(
        self, mock_run: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Returns last result even on failure when check=False.

        All max_retries+1 attempts run, but no exception is raised.
        """
        mock_run.return_value = MagicMock(returncode=1, stdout="", stderr="error")
        result = run_with_retry(["cmd"], check=False)
        assert result.returncode == 1
        # Default max_retries=5 -> 6 total attempts
        assert mock_run.call_count == 6
        assert mock_sleep.call_count == 5

    @patch("owui.retry.time.sleep")
    @patch("owui.retry.subprocess.run")
    def test_run_with_retry_fibonacci_delays_used(
        self, mock_run: MagicMock, mock_sleep: MagicMock
    ) -> None:
        """Sleep delays follow Fibonacci sequence.

        max_retries=5 -> 6 total attempts. Sleep called before
        attempts 1-5 (5 times). The delays from fibonacci_delays
        are [10, 10, 20, 30, 50, 80]. Attempt 0 skips sleep,
        so sleep receives delays at indices 1-5: [10, 20, 30, 50, 80].
        """
        mock_run.return_value = MagicMock(returncode=1, stdout="", stderr="err")
        with pytest.raises(subprocess.CalledProcessError):
            run_with_retry(["fail"], max_retries=5, initial_delay=10)
        sleep_args = [call.args[0] for call in mock_sleep.call_args_list]
        assert sleep_args == [10, 20, 30, 50, 80]
