"""CLI entry point with subcommands.

Provides setup, diagnose, models, run, and config subcommands
for managing the OpenWebUI + Ollama stack.
"""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path
from typing import Any

from owui import config as config_mod
from owui import diagnostics, ollama, openwebui, system
from owui.docker import pull_image
from owui.log import LEVEL_ERROR, LEVEL_INFO, get_logger, setup_logger

logger = get_logger(__name__)


def _build_parser() -> argparse.ArgumentParser:
    """Build the argument parser with all subcommands."""
    parser = argparse.ArgumentParser(
        prog="owui",
        description="Manage OpenWebUI + Ollama stack",
    )
    parser.add_argument(
        "--config",
        type=Path,
        default=None,
        help="Path to config.toml",
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="count",
        default=0,
        help="Increase verbosity (repeat for more)",
    )
    parser.add_argument(
        "-q",
        "--quiet",
        action="store_true",
        help="Suppress non-error output",
    )

    subparsers = parser.add_subparsers(dest="command")

    # setup
    subparsers.add_parser("setup", help="Full setup flow")

    # diagnose
    diagnose_parser = subparsers.add_parser(
        "diagnose",
        help="Run diagnostics",
    )
    diagnose_parser.add_argument(
        "target",
        nargs="?",
        choices=["ollama", "openwebui", "all"],
        default="all",
        help="Diagnostic target (default: all)",
    )

    # models
    models_parser = subparsers.add_parser("models", help="Manage models")
    models_sub = models_parser.add_subparsers(dest="models_command")
    models_sub.add_parser("pull", help="Pull configured models")
    models_sub.add_parser("list", help="List installed models")

    # run
    run_parser = subparsers.add_parser(
        "run",
        help="Run an Ollama model interactively",
    )
    run_parser.add_argument(
        "--model",
        "-m",
        default=None,
        help="Model name to run",
    )

    # config
    config_parser = subparsers.add_parser("config", help="View configuration")
    config_sub = config_parser.add_subparsers(dest="config_command")
    get_parser = config_sub.add_parser(
        "get",
        help="Get a config value by dot-notation key",
    )
    get_parser.add_argument("key", help="Config key (e.g. ollama.port)")
    config_sub.add_parser("show", help="Print full configuration")

    return parser


def _resolve_verbosity(args: argparse.Namespace) -> int:
    """Compute verbosity level from CLI flags.

    Args:
        args: Parsed CLI arguments.

    Returns:
        Bash-compatible verbosity integer (0-4).
    """
    if args.quiet:
        return LEVEL_ERROR
    return LEVEL_INFO + int(args.verbose)


def _cmd_setup(cfg: dict[str, Any]) -> None:
    """Execute the full setup flow."""
    logger.info("Starting full setup...")
    system.update_system_packages()
    system.setup_docker_keyring()
    system.install_nvidia_toolkit()
    system.install_docker()
    system.install_ollama()
    system.verify_docker_environment()

    ollama_cfg = cfg["ollama"]
    openwebui_cfg = cfg["openwebui"]
    models_cfg = cfg.get("models", {})

    # Ollama
    pull_image(
        str(ollama_cfg["image"]),
        str(ollama_cfg["container_tag"]),
    )
    system.ensure_port_available(int(ollama_cfg["port"]))
    ollama.stop_remove_run(ollama_cfg)
    ollama.verify_setup(
        str(ollama_cfg["host"]),
        int(ollama_cfg["port"]),
    )
    model_list = models_cfg.get("default", [])
    if isinstance(model_list, list):
        ollama.pull_models([str(m) for m in model_list])

    # OpenWebUI
    pull_image(
        str(openwebui_cfg["image"]),
        str(openwebui_cfg["container_tag"]),
    )
    system.ensure_port_available(int(openwebui_cfg["port"]))
    openwebui.stop_remove_run(openwebui_cfg, ollama_cfg)
    openwebui.verify_setup(openwebui_cfg)

    logger.info("Setup completed.")


def _cmd_diagnose(cfg: dict[str, Any], target: str) -> None:
    """Run diagnostics for the specified target."""
    if target == "all":
        diagnostics.run_all(cfg)
    elif target == "ollama":
        ollama_cfg = cfg.get("ollama", {})
        diagnostics.test_port(
            str(ollama_cfg.get("host", "localhost")),
            int(ollama_cfg.get("port", 11434)),
        )
        diagnostics.check_container_logs(
            str(ollama_cfg.get("container_name", "ollama")),
        )
    elif target == "openwebui":
        openwebui_cfg = cfg.get("openwebui", {})
        diagnostics.test_port(
            str(openwebui_cfg.get("host", "localhost")),
            int(openwebui_cfg.get("port", 3000)),
        )
        diagnostics.check_container_logs(
            str(openwebui_cfg.get("container_name", "open-webui")),
        )


def _cmd_models(
    cfg: dict[str, Any],
    models_command: str | None,
) -> None:
    """Handle models subcommands."""
    if models_command == "pull":
        models_cfg = cfg.get("models", {})
        model_list = models_cfg.get("default", [])
        if isinstance(model_list, list):
            ollama.pull_models([str(m) for m in model_list])
    elif models_command == "list":
        result = subprocess.run(
            ["ollama", "list"],
            capture_output=True,
            text=True,
            check=False,
        )
        print(result.stdout)
        if result.returncode != 0:
            logger.error("ollama list failed: %s", result.stderr)
    else:
        print("Usage: owui models {pull|list}")


def _cmd_run(cfg: dict[str, Any], model: str | None) -> None:
    """Run an Ollama model interactively."""
    if model is None:
        models_cfg = cfg.get("models", {})
        default_models = models_cfg.get("default", [])
        if isinstance(default_models, list) and default_models:
            model = str(default_models[0])
        else:
            logger.error("No model specified and no default model configured.")
            sys.exit(1)

    logger.info("Running model: %s", model)
    subprocess.run(
        ["ollama", "run", model],
        check=False,
    )


def _cmd_config(
    cfg: dict[str, Any],
    config_command: str | None,
    key: str | None,
) -> None:
    """Handle config subcommands."""
    if config_command == "get":
        if key is None:
            print("Usage: owui config get KEY")
            return
        try:
            value = config_mod.get_config_value(key, cfg)
            print(value)
        except KeyError:
            logger.exception("Config key not found")
            sys.exit(1)
    elif config_command == "show":
        _print_config(cfg)
    else:
        print("Usage: owui config {get KEY|show}")


def _print_config(cfg: dict[str, Any], prefix: str = "") -> None:
    """Recursively print config as dot-notation key-value pairs."""
    for k, v in cfg.items():
        full_key = f"{prefix}{k}" if not prefix else f"{prefix}.{k}"
        if isinstance(v, dict):
            _print_config(v, full_key)
        else:
            print(f"{full_key} = {v}")


def main() -> None:
    """CLI entry point."""
    parser = _build_parser()
    args = parser.parse_args()

    verbosity = _resolve_verbosity(args)
    setup_logger("owui", verbosity)
    # Re-setup module loggers to inherit verbosity.
    for mod_name in [
        "owui.config",
        "owui.retry",
        "owui.docker",
        "owui.system",
        "owui.ollama",
        "owui.openwebui",
        "owui.diagnostics",
    ]:
        setup_logger(mod_name, verbosity)

    cfg = config_mod.load_config(args.config)

    if args.command is None:
        parser.print_help()
        sys.exit(0)

    if args.command == "setup":
        _cmd_setup(cfg)
    elif args.command == "diagnose":
        _cmd_diagnose(cfg, args.target)
    elif args.command == "models":
        _cmd_models(cfg, args.models_command)
    elif args.command == "run":
        _cmd_run(cfg, args.model)
    elif args.command == "config":
        _cmd_config(cfg, args.config_command, getattr(args, "key", None))


if __name__ == "__main__":
    main()
