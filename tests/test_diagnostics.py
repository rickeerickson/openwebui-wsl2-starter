"""Tests for owui.diagnostics module."""

from __future__ import annotations

from typing import Any
from unittest.mock import MagicMock, patch

from owui import diagnostics


class TestShowSystemInfo:
    """Tests for show_system_info()."""

    @patch("owui.diagnostics.subprocess.run")
    def test_show_system_info(self, mock_run: MagicMock) -> None:
        """Calls subprocess.run for whoami, echo $HOME, and lsb_release."""
        mock_run.return_value = MagicMock(returncode=0, stdout="ok\n", stderr="")
        diagnostics.show_system_info()
        # 3 calls: whoami (via _run_and_log), echo $HOME (via _run_and_log),
        # lsb_release -a (direct)
        assert mock_run.call_count == 3
        cmds = [c.args[0] for c in mock_run.call_args_list]
        assert "whoami" in cmds[0]
        assert "lsb_release" in cmds[2]

    @patch("owui.diagnostics.subprocess.run")
    def test_show_system_info_no_lsb(self, mock_run: MagicMock) -> None:
        """Falls back to /etc/os-release when lsb_release fails."""

        def side_effect(cmd: list[str], **kwargs: Any) -> MagicMock:
            if "lsb_release" in cmd:
                return MagicMock(returncode=1, stdout="", stderr="not found")
            return MagicMock(returncode=0, stdout="ok\n", stderr="")

        mock_run.side_effect = side_effect
        diagnostics.show_system_info()
        # 4 calls: whoami, echo $HOME, lsb_release (fails),
        # cat /etc/os-release (fallback)
        assert mock_run.call_count == 4


class TestShowDockerInfo:
    """Tests for show_docker_info()."""

    @patch("owui.diagnostics.subprocess.run")
    def test_show_docker_info(self, mock_run: MagicMock) -> None:
        """Calls docker --version, docker ps, docker images."""
        mock_run.return_value = MagicMock(returncode=0, stdout="ok\n", stderr="")
        diagnostics.show_docker_info()
        assert mock_run.call_count == 3
        cmds = [c.args[0] for c in mock_run.call_args_list]
        assert ["docker", "--version"] in cmds
        assert ["docker", "ps"] in cmds
        assert ["docker", "images"] in cmds


class TestTestPort:
    """Tests for test_port()."""

    @patch("owui.diagnostics.subprocess.run")
    @patch("owui.diagnostics.socket.create_connection")
    def test_test_port_success(
        self,
        mock_socket: MagicMock,
        mock_run: MagicMock,
    ) -> None:
        """Tests TCP connection and HTTP check via curl."""
        mock_socket.return_value.__enter__ = MagicMock()
        mock_socket.return_value.__exit__ = MagicMock(return_value=False)
        mock_run.return_value = MagicMock(returncode=0, stdout="200", stderr="")
        diagnostics.test_port("localhost", 8080)
        mock_socket.assert_called_once_with(("localhost", 8080), timeout=5)
        mock_run.assert_called_once()

    @patch("owui.diagnostics.subprocess.run")
    @patch("owui.diagnostics.socket.create_connection")
    def test_test_port_tcp_failure(
        self,
        mock_socket: MagicMock,
        mock_run: MagicMock,
    ) -> None:
        """Logs warning when TCP connection fails, still runs curl."""
        mock_socket.side_effect = OSError("Connection refused")
        mock_run.return_value = MagicMock(returncode=0, stdout="", stderr="")
        diagnostics.test_port("localhost", 9999)
        mock_socket.assert_called_once()
        mock_run.assert_called_once()


class TestCheckContainerLogs:
    """Tests for check_container_logs()."""

    @patch("owui.diagnostics.subprocess.run")
    def test_check_container_logs_running(self, mock_run: MagicMock) -> None:
        """Calls docker ps then docker logs when container is running."""
        mock_run.side_effect = [
            # docker ps check
            MagicMock(returncode=0, stdout="ollama\n", stderr=""),
            # docker logs
            MagicMock(returncode=0, stdout="listening\n", stderr=""),
        ]
        diagnostics.check_container_logs("ollama")
        assert mock_run.call_count == 2
        ps_cmd = mock_run.call_args_list[0].args[0]
        assert "docker" in ps_cmd
        assert "ps" in ps_cmd
        logs_cmd = mock_run.call_args_list[1].args[0]
        assert "docker" in logs_cmd
        assert "logs" in logs_cmd

    @patch("owui.diagnostics.subprocess.run")
    def test_check_container_logs_not_running(self, mock_run: MagicMock) -> None:
        """Skips docker logs when container is not in docker ps output."""
        mock_run.return_value = MagicMock(returncode=0, stdout="", stderr="")
        diagnostics.check_container_logs("ollama")
        # Only the docker ps check, no docker logs
        mock_run.assert_called_once()


class TestRunAll:
    """Tests for run_all()."""

    @patch("owui.diagnostics.check_routing")
    @patch("owui.diagnostics.check_container_logs")
    @patch("owui.diagnostics.show_docker_info")
    @patch("owui.diagnostics.test_port")
    @patch("owui.diagnostics.show_listening_ports")
    @patch("owui.diagnostics.show_network")
    @patch("owui.diagnostics.show_system_info")
    def test_run_all(
        self,
        mock_sys: MagicMock,
        mock_net: MagicMock,
        mock_ports: MagicMock,
        mock_test_port: MagicMock,
        mock_docker: MagicMock,
        mock_logs: MagicMock,
        mock_routing: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """Calls all diagnostic functions."""
        diagnostics.run_all(sample_config)
        mock_sys.assert_called_once()
        mock_net.assert_called_once()
        mock_ports.assert_called_once()
        mock_docker.assert_called_once()
        mock_routing.assert_called_once()
        # test_port called for ollama and openwebui
        assert mock_test_port.call_count == 2
        # check_container_logs called for ollama and openwebui
        assert mock_logs.call_count == 2
