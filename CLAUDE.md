# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when
working with code in this repository.

## Project Overview

Automation toolkit for setting up OpenWebUI + Ollama in WSL2 on
Windows. Cross-platform: PowerShell for Windows host setup, Bash
for WSL2 container orchestration. Docker containers run with
NVIDIA GPU support via `network=host`.

## Build and Test

**Linux/macOS (Bash):**

```bash
bash build_and_test.sh            # run all checks
bash build_and_test.sh --install  # install missing tools
bash build_and_test.sh --fix      # auto-fix (markdownlint)
bash build_and_test.sh --shell    # shellcheck + bash -n only
bash build_and_test.sh --markdown # markdownlint only
bash build_and_test.sh --powershell # pwsh parse + analyzer
bash build_and_test.sh --test     # unit tests only
```

Stages: `bash -n` syntax, PowerShell parse, shellcheck,
markdownlint, PSScriptAnalyzer, unit tests. Skips linters
gracefully if not installed; use `--install` to bootstrap.

`--install` installs: `shellcheck-py` (pip in `.venv/`),
`markdownlint-cli` (brew/npm), `pwsh` (brew cask/apt),
`PSScriptAnalyzer` (pwsh module). Always uses a Python venv
at `.venv/` for pip-installed tools.

**Windows (cmd):**

```cmd
build_and_test.cmd
```

Stages: PowerShell parse, PSScriptAnalyzer. Requires `pwsh`
(PowerShell 7+).

**Unit tests only:**

```bash
bash bash/tests/test_run_command.sh
```

Tests cover `run_command()` (11 scenarios). Enable verbose
output with `DEBUG=true`.

## Entry Points

- Windows: `RUNME.cmd` or `RUNME.ps1` (requires admin,
  sets up WSL2 + Ubuntu, then runs Bash setup)
- WSL/Linux: `update_open-webui.sh` (main orchestration)
- Config: `update_open-webui.config.sh` (ports, container
  names, volumes, default models)

## Architecture

### Setup Flow

`RUNME.ps1` (Windows) -> WSL2/Ubuntu install ->
`update_open-webui.sh` (Bash) -> system packages -> Docker +
NVIDIA toolkit -> Ollama container (port 11434) -> OpenWebUI
container (port 3000, connects to Ollama via OLLAMA_BASE_URL)
-> Windows port proxy for host access.

### Library Hierarchy

Each component has its own `common/` directory with
`com_env.sh` (environment/constants) and `com_lib.sh`
(functions). They all build on the base layer:

- `bash/common/repo_lib.sh` - Core library (logging,
  `run_command`, retry logic, Docker/container management,
  system setup)
- `bash/common/repo_env.sh` - Sources `repo_lib.sh`, defines
  log levels and constants (`LEVEL_ERROR`=0 through
  `LEVEL_DEBUG_2`=4)
- `ollama/scripts/common/com_env.sh` + `com_lib.sh` -
  Ollama-specific functions
- `open-webui/scripts/common/com_env.sh` + `com_lib.sh` -
  OpenWebUI-specific functions
- `powershell/CommonLibrary.psm1` - PowerShell equivalent
  (WSL management, port proxy, path conversion)

Scripts source their environment via `source_required_file()`,
which validates file existence before sourcing.

### Retry Strategy

Fibonacci backoff (10, 10, 20, 30, 50, 80 seconds), max 5
attempts. Applied to package updates, Docker commands,
container startup, and model pulls. Implemented in
`run_command_with_retry()` and `retry_logic()`.

### Container Management Pattern

Functions check state before acting (idempotent):
`container_exists()`, `container_is_running()`, then
`stop_and_remove_container()`, `pull_docker_image()`, run
new container. Both Ollama and OpenWebUI follow this pattern
via `stop_remove_run_*_container()` orchestrators.

## Linter Configuration

- `.shellcheckrc` - Global shellcheck suppressions: SC1090
  (non-constant source), SC2154 (variable from sourced file),
  SC2034 (appears unused, exported via source)
- `.PSScriptAnalyzerSettings.psd1` - Excludes intentional
  patterns: Write-Host, Invoke-Expression, ShouldProcess,
  plural nouns, unused params, Write-Log override
- `.gitattributes` - Line endings: CRLF for `.ps1`, `.psm1`,
  `.psd1`, `.cmd`; LF for `.sh`, `.md`. Mixed-OS repo, so
  line endings matter.
- Markdown lines must stay under 80 characters (markdownlint
  MD013 default)

## Shell Script Conventions

- Strict mode everywhere: `set -euo pipefail` with ERR trap
  logging file, line, and command
- `source_required_file()` pattern for safe sourcing with
  existence checks. This pattern causes shellcheck false
  positives (SC1090, SC2154, SC2034), handled by
  `.shellcheckrc`.
- Leveled logging via `log_message()` with `VERBOSITY` control
- `run_command()` wraps execution with logging,
  `ignore_exit_status`, and `should_fail` flags
- Log files written adjacent to the running script
  (`${script_dir}/${script_name}.log`)

### Bash Pitfalls in This Repo

- `(( x++ ))` returns exit code 1 when x=0 under `set -e`.
  Use `x=$(( x + 1 ))` instead.
- SC2155: `local FOO=$(cmd)` masks return values. Always
  declare and assign separately: `local foo; foo=$(cmd)`.
- SC2076: Quoted RHS in `[[ $x =~ "pattern" ]]` does literal
  match, not regex. Extract to a variable:
  `local p="pattern"; [[ $x =~ ${p} ]]`.
- `build_and_test.sh` uses `set -u` (not `set -euo pipefail`)
  so it can track pass/fail counts without exiting early.

## PowerShell Conventions

- `Write-Log` for leveled logging (mirrors Bash `log_message`)
- `Start-CommandWithRetry` for Fibonacci retry (mirrors Bash
  `run_command_with_retry`)
- `ParseBashConfig` reads `update_open-webui.config.sh` into
  a PowerShell hashtable, keeping config in one place
- `Convert-ToPath` handles Windows-to-WSL path translation
  (`C:\foo` -> `/mnt/c/foo`)
