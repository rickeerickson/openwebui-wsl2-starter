"""Tests for owui.cli module."""

from __future__ import annotations

from typing import Any
from unittest.mock import MagicMock, patch

from owui import cli


class TestCliConfigGet:
    """Tests for 'owui config get' subcommand."""

    @patch("owui.cli.config_mod.load_config")
    def test_cli_config_get(
        self,
        mock_load: MagicMock,
        sample_config: dict[str, Any],
        capsys: Any,
    ) -> None:
        """'owui config get ollama.port' prints 11434."""
        mock_load.return_value = sample_config
        with patch("sys.argv", ["owui", "config", "get", "ollama.port"]):
            cli.main()
        captured = capsys.readouterr()
        assert "11434" in captured.out


class TestCliConfigShow:
    """Tests for 'owui config show' subcommand."""

    @patch("owui.cli.config_mod.load_config")
    def test_cli_config_show(
        self,
        mock_load: MagicMock,
        sample_config: dict[str, Any],
        capsys: Any,
    ) -> None:
        """'owui config show' prints config keys."""
        mock_load.return_value = sample_config
        with patch("sys.argv", ["owui", "config", "show"]):
            cli.main()
        captured = capsys.readouterr()
        assert "ollama" in captured.out
        assert "port" in captured.out


class TestCliSetup:
    """Tests for 'owui setup' subcommand."""

    @patch("owui.cli.openwebui.verify_setup")
    @patch("owui.cli.openwebui.stop_remove_run")
    @patch("owui.cli.ollama.pull_models")
    @patch("owui.cli.ollama.verify_setup")
    @patch("owui.cli.ollama.stop_remove_run")
    @patch("owui.cli.pull_image")
    @patch("owui.cli.system.ensure_port_available")
    @patch("owui.cli.system.verify_docker_environment")
    @patch("owui.cli.system.install_ollama")
    @patch("owui.cli.system.install_docker")
    @patch("owui.cli.system.install_nvidia_toolkit")
    @patch("owui.cli.system.setup_docker_keyring")
    @patch("owui.cli.system.update_system_packages")
    @patch("owui.cli.config_mod.load_config")
    def test_cli_setup(
        self,
        mock_load: MagicMock,
        mock_update: MagicMock,
        mock_keyring: MagicMock,
        mock_nvidia: MagicMock,
        mock_docker: MagicMock,
        mock_ollama_install: MagicMock,
        mock_verify_docker: MagicMock,
        mock_port: MagicMock,
        mock_pull_image: MagicMock,
        mock_ollama_run: MagicMock,
        mock_ollama_verify: MagicMock,
        mock_pull_models: MagicMock,
        mock_webui_run: MagicMock,
        mock_webui_verify: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """Setup subcommand calls all setup functions."""
        mock_load.return_value = sample_config
        with patch("sys.argv", ["owui", "setup"]):
            cli.main()
        mock_update.assert_called_once()
        mock_keyring.assert_called_once()
        mock_nvidia.assert_called_once()
        mock_docker.assert_called_once()
        mock_ollama_install.assert_called_once()
        mock_verify_docker.assert_called_once()
        # pull_image called twice (ollama + openwebui)
        assert mock_pull_image.call_count == 2
        # ensure_port_available called twice
        assert mock_port.call_count == 2
        mock_ollama_run.assert_called_once()
        mock_ollama_verify.assert_called_once()
        mock_pull_models.assert_called_once()
        mock_webui_run.assert_called_once()
        mock_webui_verify.assert_called_once()


class TestCliDiagnose:
    """Tests for 'owui diagnose' subcommand."""

    @patch("owui.cli.diagnostics.run_all")
    @patch("owui.cli.config_mod.load_config")
    def test_cli_diagnose(
        self,
        mock_load: MagicMock,
        mock_diag: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """Diagnose subcommand calls diagnostics.run_all with config."""
        mock_load.return_value = sample_config
        with patch("sys.argv", ["owui", "diagnose"]):
            cli.main()
        mock_diag.assert_called_once_with(sample_config)


class TestCliModelsList:
    """Tests for 'owui models list' subcommand."""

    @patch("owui.cli.subprocess.run")
    @patch("owui.cli.config_mod.load_config")
    def test_cli_models_list(
        self,
        mock_load: MagicMock,
        mock_run: MagicMock,
        sample_config: dict[str, Any],
        capsys: Any,
    ) -> None:
        """'owui models list' calls ollama list and prints output."""
        mock_load.return_value = sample_config
        mock_run.return_value = MagicMock(
            returncode=0,
            stdout="NAME\tSIZE\nllama3.2:1b\t1.3GB\n",
            stderr="",
        )
        with patch("sys.argv", ["owui", "models", "list"]):
            cli.main()
        captured = capsys.readouterr()
        assert "llama3.2:1b" in captured.out


class TestCliVerbose:
    """Tests for -v flag."""

    @patch("owui.cli.setup_logger")
    @patch("owui.cli.config_mod.load_config")
    def test_cli_verbose(
        self,
        mock_load: MagicMock,
        mock_setup_logger: MagicMock,
        sample_config: dict[str, Any],
    ) -> None:
        """The -v flag increases verbosity from 2 to 3.

        setup_logger is called once for "owui" and once for each
        module logger. The first call sets verbosity=3.
        """
        mock_load.return_value = sample_config
        mock_setup_logger.return_value = MagicMock()
        with patch("sys.argv", ["owui", "-v", "config", "show"]):
            cli.main()
        # First call is setup_logger("owui", 3)
        first_call = mock_setup_logger.call_args_list[0]
        assert first_call.args == ("owui", 3)
