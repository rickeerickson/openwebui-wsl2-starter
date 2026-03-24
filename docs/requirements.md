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
- FR-WIN-4: Run the Linux CLI inside WSL from Windows
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
  (`ow.yaml`).
- FR-CFG-2: Config supports override via file, environment
  variables, and CLI flags, resolved in a defined order.
- FR-CFG-3: Default models are configurable as a YAML list.
- FR-CFG-4: All config values must be validated at load time
  before any system commands execute.

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

- FR-DGN-1: Provide a diagnose subcommand from inside WSL
  for Ollama and OpenWebUI.
- FR-DGN-2: Provide a diagnose subcommand from the Windows
  host for Ollama and OpenWebUI.
- FR-DGN-3: Diagnostics must report: system info, network
  interfaces, listening ports, TCP connectivity, HTTP response
  codes, Docker status, and container logs.

### Interactive Utilities

- FR-UTL-1: Provide an interactive Ollama model runner that
  lists installed models and accepts a model selection.
- FR-UTL-2: Provide a model pull subcommand that pulls all
  models listed in config.

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

- NFR-COR-1: Go code must pass `golangci-lint` with security
  linters enabled (`gosec`, `gocritic`, `bodyclose`, `noctx`).
- NFR-COR-2: Go code must pass `go vet` with no warnings.
- NFR-COR-3: Markdown files must pass markdownlint with lines
  under 80 characters.
- NFR-COR-4: Dependencies must pass `govulncheck` with no
  known CVEs.

### Security

- NFR-SEC-1: Command execution must use `os/exec.Command`
  with explicit argument lists. No shell string interpolation.
- NFR-SEC-2: The command runner must enforce an allowlist of
  permitted executables.
- NFR-SEC-3: All config values must be validated against
  strict patterns before use in any system command.
- NFR-SEC-4: All `exec.Command` invocations must be logged
  with the full argument list for audit.
- NFR-SEC-5: Log files must be written with restrictive
  permissions (0600).

### Testability

- NFR-TST-1: All `internal/` packages must have unit tests.
- NFR-TST-2: The command runner must be mockable so tests
  can verify behavior without real shell-outs.
- NFR-TST-3: Mutation testing must run at each milestone.
  `internal/` packages must achieve a mutation score of 80%
  or higher.
- NFR-TST-4: CI must build, lint, and test on every push
  and PR.

### Maintainability

- NFR-MNT-1: Shared logic (config, logging, retry, exec)
  must live in `internal/` packages compiled on both
  platforms.
- NFR-MNT-2: Platform-specific code must use Go build tags
  or `_windows.go` / `_linux.go` suffixes.
- NFR-MNT-3: External dependencies must be minimal. Prefer
  the standard library where practical.

### Cross-Platform

- NFR-XPL-1: Both CLIs build from the same Go module using
  `GOOS=windows` and `GOOS=linux`.
- NFR-XPL-2: The CLI binary name is `ow` on both platforms.
  Subcommands work the same way regardless of OS.

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
