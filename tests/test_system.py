"""Tests for owui.system module."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

from owui import system


class TestUpdateSystemPackages:
    """Tests for update_system_packages()."""

    @patch("owui.system.run_with_retry")
    def test_update_system_packages(self, mock_retry: MagicMock) -> None:
        """Calls run_with_retry 5 times: update, upgrade, dist-upgrade,
        autoremove, autoclean."""
        system.update_system_packages()
        assert mock_retry.call_count == 5
        cmds = [c.args[0] for c in mock_retry.call_args_list]
        assert "update" in cmds[0]
        assert "upgrade" in cmds[1]
        assert "dist-upgrade" in cmds[2]
        assert "autoremove" in cmds[3]
        assert "autoclean" in cmds[4]


class TestSetupDockerKeyring:
    """Tests for setup_docker_keyring()."""

    @patch("owui.system.subprocess.run")
    @patch("owui.system.run_with_retry")
    def test_setup_docker_keyring(
        self,
        mock_retry: MagicMock,
        mock_run: MagicMock,
    ) -> None:
        """Uses run_with_retry for installs and subprocess.run for dpkg/tee."""
        mock_run.return_value = MagicMock(returncode=0, stdout="amd64\n", stderr="")
        system.setup_docker_keyring()
        # 4 run_with_retry calls: apt install, install dir, curl key, chmod
        assert mock_retry.call_count == 4
        # 3 subprocess.run calls: dpkg --print-architecture,
        # bash -c os-release, sudo tee
        assert mock_run.call_count == 3


class TestInstallDocker:
    """Tests for install_docker()."""

    @patch("owui.system.subprocess.run")
    @patch("owui.system.run_with_retry")
    def test_install_docker(
        self,
        mock_retry: MagicMock,
        mock_run: MagicMock,
    ) -> None:
        """Calls run_with_retry for apt install, nvidia-ctk, and more.

        Also calls subprocess.run for getent group check and whoami.
        """
        # getent group docker succeeds (group exists)
        mock_run.side_effect = [
            MagicMock(returncode=0, stdout="docker:x:999:\n", stderr=""),
            MagicMock(returncode=0, stdout="testuser\n", stderr=""),
        ]
        system.install_docker()
        # 6 run_with_retry: apt-get update, apt-get install, nvidia-ctk,
        # usermod, systemctl restart, docker run hello-world
        assert mock_retry.call_count == 6
        retry_cmds = [c.args[0] for c in mock_retry.call_args_list]
        assert "docker-ce" in retry_cmds[1]
        # 2 subprocess.run: getent group docker, whoami
        assert mock_run.call_count == 2


class TestInstallOllama:
    """Tests for install_ollama()."""

    @patch("owui.system.subprocess.run")
    @patch("owui.system.shutil.which")
    def test_install_ollama_already_installed(
        self, mock_which: MagicMock, mock_run: MagicMock
    ) -> None:
        """Skips install when ollama is already on PATH."""
        mock_which.return_value = "/usr/bin/ollama"
        system.install_ollama()
        mock_run.assert_not_called()

    @patch("owui.system.subprocess.run")
    @patch("owui.system.shutil.which")
    def test_install_ollama_not_installed(
        self, mock_which: MagicMock, mock_run: MagicMock
    ) -> None:
        """Downloads and runs installer when ollama is not on PATH."""
        # First which returns None (not installed), second returns path
        mock_which.side_effect = [None, "/usr/bin/ollama"]
        mock_run.side_effect = [
            MagicMock(returncode=0, stdout=b"#!/bin/sh\n"),  # curl
            MagicMock(returncode=0, stdout=b"", stderr=b""),  # sh
        ]
        system.install_ollama()
        assert mock_run.call_count == 2
        curl_args = mock_run.call_args_list[0].args[0]
        assert "curl" in curl_args


class TestEnsurePortAvailable:
    """Tests for ensure_port_available()."""

    @patch("owui.system.subprocess.run")
    def test_ensure_port_available_free(self, mock_run: MagicMock) -> None:
        """No kill called when port is free (lsof returns non-zero)."""
        mock_run.return_value = MagicMock(returncode=1, stdout="", stderr="")
        system.ensure_port_available(8080)
        mock_run.assert_called_once()

    @patch("owui.system.run_with_retry")
    @patch("owui.system.subprocess.run")
    def test_ensure_port_available_occupied(
        self,
        mock_run: MagicMock,
        mock_retry: MagicMock,
    ) -> None:
        """Calls run_with_retry with kill when lsof returns a PID."""
        mock_run.return_value = MagicMock(returncode=0, stdout="12345\n", stderr="")
        system.ensure_port_available(8080)
        mock_run.assert_called_once()
        mock_retry.assert_called_once_with(["sudo", "kill", "-9", "12345"])


class TestVerifyDockerEnvironment:
    """Tests for verify_docker_environment()."""

    @patch("owui.system.run_with_retry")
    @patch("owui.system.subprocess.run")
    def test_verify_docker_environment(
        self,
        mock_run: MagicMock,
        mock_retry: MagicMock,
    ) -> None:
        """Checks docker group, service status, then docker --version
        and docker context ls."""
        mock_run.side_effect = [
            MagicMock(returncode=0, stdout="user docker\n", stderr=""),  # id -nG
            MagicMock(returncode=0, stdout="", stderr=""),  # systemctl
        ]
        system.verify_docker_environment()
        assert mock_run.call_count == 2
        # 2 run_with_retry: docker --version, docker context ls
        assert mock_retry.call_count == 2


class TestVerifyNvidiaEnvironment:
    """Tests for verify_nvidia_environment()."""

    @patch("owui.system.run_with_retry")
    def test_verify_nvidia_environment(self, mock_retry: MagicMock) -> None:
        """Calls run_with_retry with nvidia-smi."""
        system.verify_nvidia_environment()
        mock_retry.assert_called_once_with(["nvidia-smi"])
