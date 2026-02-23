# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code)
when working with code in this repository.

## Project Overview

Automation toolkit for setting up OpenWebUI + Ollama in WSL2
on Windows. Python package (`owui`) handles all business
logic. PowerShell manages Windows host setup. Docker
containers run with NVIDIA GPU support via `network=host`.

## Build and Test

**Linux/macOS (Bash):**

```bash
bash build_and_test.sh            # all checks
bash build_and_test.sh --install  # install tools + owui
bash build_and_test.sh --fix      # auto-fix (markdownlint)
bash build_and_test.sh --shell    # shellcheck bootstrap.sh
bash build_and_test.sh --markdown # markdownlint only
bash build_and_test.sh --powershell # pwsh parse + analyzer
bash build_and_test.sh --test     # pytest only
```

Stages: shellcheck (`bootstrap.sh` only), markdownlint,
PowerShell parse + PSScriptAnalyzer, pytest. Skips tools
not installed; use `--install` to bootstrap.

`--install` installs: `shellcheck-py` (pip in `.venv/`),
`markdownlint-cli` (brew/npm), `pwsh` (brew cask/apt),
`PSScriptAnalyzer` (pwsh module), and the `owui` package
in editable mode. Always uses a Python venv at `.venv/`.

**Windows (cmd):**

```cmd
build_and_test.cmd
```

Stages: PowerShell parse, PSScriptAnalyzer. Requires `pwsh`
(PowerShell 7+).

**pytest only:**

```bash
pytest tests/
```

70 tests covering all `owui` modules. Run with `-v` for
verbose output.

## Entry Points

- Windows: `RUNME.cmd` -> `RUNME.ps1` (requires admin,
  sets up WSL2 + Ubuntu, then runs `bootstrap.sh`)
- WSL/Linux: `./bootstrap.sh` (creates venv, installs
  `owui`, runs `owui setup`)
- Config: `config.toml` (ports, container names, volumes,
  default models)
- CLI: `owui` with subcommands after venv activation

## Architecture

### Setup Flow

`RUNME.ps1` (Windows) -> WSL2/Ubuntu install ->
`bootstrap.sh` -> `owui setup` -> system packages ->
Docker + NVIDIA toolkit -> Ollama container (port 11434)
-> OpenWebUI container (port 3000, connects to Ollama via
OLLAMA_BASE_URL) -> Windows port proxy for host access.

### Package Structure

The `owui` Python package contains all business logic:

- `owui/cli.py` - CLI entry point, argparse subcommands
- `owui/config.py` - TOML config loading (`config.toml`)
- `owui/docker.py` - Container management (check state,
  stop, remove, pull, run)
- `owui/retry.py` - Fibonacci retry with `subprocess.run`
- `owui/system.py` - System package and Docker install
- `owui/ollama.py` - Ollama container orchestration
- `owui/openwebui.py` - OpenWebUI container orchestration
- `owui/diagnostics.py` - Health checks and diagnostics
- `owui/log.py` - Leveled logging (ERROR through DEBUG)

Supporting files:

- `bootstrap.sh` - Thin Bash wrapper (~20 lines): ensures
  python3, creates venv, installs `owui`, runs
  `owui setup`
- `config.toml` - Single config source (TOML format)
- `pyproject.toml` - Python package definition
- `powershell/CommonLibrary.psm1` - PowerShell library
  (WSL management, port proxy, path conversion)

### CLI Subcommands

```bash
owui setup              # Full setup flow
owui diagnose           # Run all diagnostics
owui diagnose ollama    # Ollama diagnostics only
owui diagnose openwebui # OpenWebUI diagnostics only
owui models pull        # Pull configured models
owui models list        # List installed models
owui run                # Interactive model selection
owui run --model NAME   # Run specific model
owui config show        # Print full config
owui config get KEY     # Get value by dot-notation key
```

### Retry Strategy

Fibonacci backoff (10, 10, 20, 30, 50, 80 seconds), max 5
attempts. Applied to package updates, Docker commands,
container startup, and model pulls. Implemented in
`owui/retry.py`.

### Container Management Pattern

Functions check state before acting (idempotent):
`container_exists()`, `container_is_running()`, then
`stop_and_remove()`, `pull_image()`, run new container.
Both Ollama and OpenWebUI follow this pattern. Implemented
in `owui/docker.py`, called by `owui/ollama.py` and
`owui/openwebui.py`.

## Linter Configuration

- `.shellcheckrc` - Shellcheck suppressions: SC1090
  (non-constant source), SC2154 (variable from sourced
  file), SC2034 (appears unused, exported via source).
  Only applies to `bootstrap.sh` now.
- `.PSScriptAnalyzerSettings.psd1` - Excludes intentional
  patterns: Write-Host, Invoke-Expression, ShouldProcess,
  plural nouns, unused params, Write-Log override
- `.gitattributes` - Line endings: CRLF for `.ps1`,
  `.psm1`, `.psd1`, `.cmd`; LF for `.sh`, `.md`.
  Mixed-OS repo, so line endings matter.
- Markdown lines must stay under 80 characters
  (markdownlint MD013 default)

## Python Conventions

- Python 3.11+ required (`tomllib` is stdlib)
- No runtime dependencies (stdlib only)
- Dev dependency: `pytest>=7.0`
- Package installed in editable mode (`pip install -e .`)
- CLI entry point registered in `pyproject.toml`:
  `owui = "owui.cli:main"`
- Leveled logging via `owui/log.py` with verbosity
  control (`-v`, `-q` flags)
- Config loaded from `config.toml` via `tomllib`,
  accessed by dot-notation keys through `owui config get`

## Shell Script Conventions

Only `bootstrap.sh` and `build_and_test.sh` remain as
shell scripts.

- `bootstrap.sh`: strict mode (`set -euo pipefail`) with
  ERR trap logging file, line, and command
- `build_and_test.sh`: uses `set -u` (not
  `set -euo pipefail`) to track pass/fail counts without
  exiting early

### Bash Pitfalls in This Repo

- `(( x++ ))` returns exit code 1 when x=0 under
  `set -e`. Use `x=$(( x + 1 ))` instead.
- `build_and_test.sh` uses `set -u` specifically so it
  can accumulate results across stages.

## PowerShell Conventions

- `Write-Log` for leveled logging
- `Start-CommandWithRetry` for Fibonacci retry
- `Convert-ToPath` handles Windows-to-WSL path translation
  (`C:\foo` -> `/mnt/c/foo`)
- Config values read via `owui config get` (replaces the
  old `ParseBashConfig` approach)
- Health checks: `Test-OllamaHealth.ps1`,
  `Test-OpenWebUIHealth.ps1`
