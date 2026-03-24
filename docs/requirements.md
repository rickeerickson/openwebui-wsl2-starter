# Requirements

## Purpose

Automate the installation and configuration of OpenWebUI and Ollama
on a Windows 11 host running WSL2, with NVIDIA GPU acceleration.
The result is a locally hosted LLM chat interface accessible from
the Windows host and optionally from other LAN devices.

## Scope

- Windows 11 host with an NVIDIA GPU
- WSL2 with Ubuntu as the Linux environment
- Docker containers for Ollama (model server) and OpenWebUI
  (chat interface)
- NVIDIA GPU passthrough from Windows to WSL2 containers

Out of scope: cloud deployment, non-NVIDIA GPUs, Kubernetes,
non-Ubuntu WSL distributions.

---

## Functional Requirements

### Windows Host Setup

- FR-WIN-1: Detect and enable required Windows features
  (WSL, VirtualMachinePlatform).
- FR-WIN-2: Install WSL2 and set it as the default version.
- FR-WIN-3: Install Ubuntu as the WSL2 distribution,
  prompting the user to create a Linux username and password.
- FR-WIN-4: Run the Linux setup script inside WSL from Windows
  without manual intervention after initial user prompts.
- FR-WIN-5: Set up a Windows port proxy so that OpenWebUI
  on WSL2 is reachable from the Windows host and LAN clients.
- FR-WIN-6: Create a Windows Firewall inbound rule for the
  OpenWebUI port when the port proxy is created.
- FR-WIN-7: All Windows operations that modify system state
  must run with administrator privileges.

### Linux (WSL2) Setup

- FR-LIN-1: Update system packages (apt-get update, upgrade,
  dist-upgrade, autoremove, autoclean).
- FR-LIN-2: Install and configure Docker CE with the NVIDIA
  Container Toolkit.
- FR-LIN-3: Add the current user to the `docker` group so
  containers can run without sudo.
- FR-LIN-4: Install the Ollama binary using the official
  install script.
- FR-LIN-5: Pull the Ollama Docker image and start the Ollama
  container with NVIDIA GPU access.
- FR-LIN-6: Pull the OpenWebUI Docker image and start the
  OpenWebUI container connected to Ollama.
- FR-LIN-7: Pull default Ollama models after Ollama is running.
- FR-LIN-8: Verify both containers are running and responding
  on their configured ports before completing setup.

### Configuration

- FR-CFG-1: All ports, hostnames, container names, image tags,
  and default models are defined in a single config file
  (`update_open-webui.config.sh`).
- FR-CFG-2: The config file must be readable by both Bash
  scripts and PowerShell scripts without duplication.
- FR-CFG-3: Default models are configurable as a Bash array
  (`DEFAULT_OLLAMA_MODELS`).

### Idempotency

- FR-IDP-1: Re-running setup must not fail if components are
  already installed or containers already exist.
- FR-IDP-2: Container management must check current state
  before stop/remove/start operations.
- FR-IDP-3: Port proxy rules must be checked before creation;
  conflicting rules must be replaced, not duplicated.
- FR-IDP-4: Docker GPG keys and apt repositories must be
  checked before re-adding.

### Diagnostics and Health Checks

- FR-DGN-1: Provide Bash scripts to diagnose Ollama and
  OpenWebUI from inside WSL.
- FR-DGN-2: Provide PowerShell scripts to diagnose Ollama and
  OpenWebUI from the Windows host.
- FR-DGN-3: Diagnostics must report: system info, network
  interfaces, listening ports, TCP connectivity, HTTP response
  codes, Docker status, and container logs.

### Interactive Utilities

- FR-UTL-1: Provide an interactive Ollama model runner that
  lists installed models and accepts a model selection.
- FR-UTL-2: Provide a bulk model pull script with a curated
  list of models organized by category.

---

## Non-Functional Requirements

### Reliability

- NFR-REL-1: All external commands (apt, docker, curl) must
  use Fibonacci backoff retry with a maximum of 5 attempts
  and delays of 10, 10, 20, 30, 50 seconds.
- NFR-REL-2: Containers must be configured with
  `--restart always` so they survive WSL restarts.
- NFR-REL-3: Container data and Ollama models must persist
  in Docker named volumes across container recreations.

### Observability

- NFR-OBS-1: All scripts must write timestamped, leveled log
  output to both stderr and a `.log` file adjacent to the
  running script.
- NFR-OBS-2: Log levels must be: ERROR (0), WARNING (1),
  INFO (2), DEBUG_1 (3), DEBUG_2 (4). Default is INFO.
- NFR-OBS-3: Verbosity must be controllable via the
  `VERBOSITY` environment variable.
- NFR-OBS-4: Errors must include the file name, line number,
  and failing command.

### Correctness

- NFR-COR-1: All Bash scripts must use strict mode:
  `set -euo pipefail` with an ERR trap.
- NFR-COR-2: All PowerShell scripts must use
  `Set-StrictMode -Version Latest` and
  `$ErrorActionPreference = 'Stop'`.
- NFR-COR-3: Shell scripts must pass shellcheck with
  suppressions documented in `.shellcheckrc`.
- NFR-COR-4: PowerShell scripts must pass PSScriptAnalyzer
  with suppressions documented in `.PSScriptAnalyzerSettings.psd1`.
- NFR-COR-5: Markdown files must pass markdownlint with lines
  under 80 characters.

### Maintainability

- NFR-MNT-1: Common logic must live in shared library files
  (`repo_lib.sh`, `CommonLibrary.psm1`); component scripts
  must not duplicate it.
- NFR-MNT-2: Each component (Ollama, OpenWebUI) must have
  its own `common/` directory that sources the base library.
- NFR-MNT-3: Safe file sourcing must use `source_required_file()`
  to validate existence before sourcing.
- NFR-MNT-4: A build-and-test script must check syntax, lint,
  and run unit tests for all scripts.

### Cross-Platform

- NFR-XPL-1: Line endings must be enforced via `.gitattributes`:
  CRLF for `.ps1`, `.psm1`, `.psd1`, `.cmd`; LF for `.sh`,
  `.md`.
- NFR-XPL-2: Windows-to-WSL path conversion (`C:\path` to
  `/mnt/c/path`) must be handled by a shared utility function.

### Performance

- NFR-PRF-1: Containers must run with `--gpus all` and
  `--network=host` to give full GPU access and minimize
  network overhead.
- NFR-PRF-2: Model pulls must skip already-installed models
  to avoid redundant downloads.

---

## Constraints

- Requires Windows 11 with Hyper-V and WSL2 support.
- Requires an NVIDIA GPU with drivers that support
  NVIDIA Container Toolkit.
- Requires internet access to pull Docker images, apt packages,
  and Ollama models.
- Requires administrator privileges on the Windows host for
  WSL setup and port proxy configuration.
- Requires the user to interactively set a Linux username and
  password during initial Ubuntu installation.
