# CLI Design

## Overview

Two Go CLI binaries replace the shell and PowerShell scripts.
Both build from a shared Go module. The Windows binary shells
out to `wsl.exe`, `netsh`, and `powershell.exe`. The Linux
binary shells out to `docker`, `apt-get`, `ollama`, and `curl`.

The goal: a user downloads `ow.exe` on a clean Windows 11
desktop and runs `ow setup`. Everything else is automated.

---

## Binaries

| Binary     | Target          | Purpose                     |
|------------|-----------------|-----------------------------|
| `ow.exe`   | `windows/amd64` | WSL, port proxy, host mgmt  |
| `ow`       | `linux/amd64`   | Docker, containers, models  |

Same name, different builds. `GOOS` and `GOARCH` at build time
determine which code compiles. Platform-specific files use Go
build tags or `_windows.go` / `_linux.go` suffixes.

---

## Subcommands

### Windows (`ow.exe`)

```text
ow setup             Full setup (WSL + Ubuntu + Linux setup + port proxy)
ow wsl install       Install/update WSL2 and Ubuntu
ow wsl remove        Unregister Ubuntu from WSL
ow wsl stop          Shut down WSL
ow proxy enable      Create netsh port proxy + firewall rule
ow proxy remove      Remove netsh port proxy rules
ow proxy show        Show current port proxy rules
ow diagnose          Run health checks from Windows side
ow config show       Print resolved configuration
ow version           Print version and build info
```

`ow setup` is the main entry point. It runs the full sequence:

1. Check admin privileges (exit with message if not elevated)
2. Enable Windows features (WSL, VirtualMachinePlatform)
3. Install/update WSL2, set default version to 2
4. Install Ubuntu (interactive: user sets password)
5. Copy `ow` Linux binary into WSL
6. Run `wsl ow setup` inside WSL
7. Enable port proxy and firewall rule
8. Print access URL

### Linux (`ow`)

```text
ow setup             Full setup (packages, Docker, containers, models)
ow containers up     Pull images, start Ollama + OpenWebUI containers
ow containers down   Stop and remove containers
ow containers status Show container state
ow ollama pull       Pull configured default models
ow ollama run        Interactive model runner (list + select)
ow ollama models     List installed models
ow diagnose          Run health checks from Linux side
ow config show       Print resolved configuration
ow version           Print version and build info
```

`ow setup` is the main entry point. It runs the full sequence:

1. Update system packages (apt)
2. Install Docker CE + NVIDIA Container Toolkit
3. Configure Docker (NVIDIA runtime, docker group)
4. Install Ollama
5. Verify Docker environment
6. Pull Ollama image, start container, verify
7. Pull default models
8. Pull OpenWebUI image, start container, verify

---

## Project Structure

```text
cmd/
  ow/
    main.go                CLI entry point (shared)
    root.go                Root cobra command, version
    setup.go               Platform-specific setup command
    setup_windows.go       Windows setup implementation
    setup_linux.go         Linux setup implementation
    diagnose.go            Platform-specific diagnose command
    diagnose_windows.go    Windows diagnose implementation
    diagnose_linux.go      Linux diagnose implementation
    config_cmd.go          Config show command
    wsl.go                 WSL subcommands (build tag: windows)
    proxy.go               Proxy subcommands (build tag: windows)
    containers.go          Container subcommands (build tag: linux)
    ollama_cmd.go          Ollama subcommands (build tag: linux)

internal/
  config/
    config.go              Config struct, loading, defaults, validation
    config_test.go

  exec/
    runner.go              Command execution with logging
    runner_test.go
    retry.go               Fibonacci backoff retry
    retry_test.go

  logging/
    logger.go              Leveled logger (file + stderr)
    logger_test.go

  docker/                  (build tag: linux)
    client.go              Container lifecycle (exists, running, stop,
                           remove, run, pull)
    client_test.go
    container.go           Container config structs
    container_test.go
    health.go              Container health checks (HTTP, port)
    health_test.go

  apt/                     (build tag: linux)
    packages.go            System package management
    packages_test.go
    keyring.go             GPG key + apt repo setup
    keyring_test.go

  nvidia/                  (build tag: linux)
    toolkit.go             NVIDIA Container Toolkit install
    toolkit_test.go

  ollama/                  (build tag: linux)
    install.go             Ollama binary install
    install_test.go
    models.go              Model pull, list, run
    models_test.go

  wsl/                     (build tag: windows)
    wsl.go                 WSL install, update, stop, remove
    wsl_test.go
    distro.go              Distro management
    distro_test.go

  netsh/                   (build tag: windows)
    proxy.go               Port proxy CRUD
    proxy_test.go
    firewall.go            Firewall rule management
    firewall_test.go

  winfeature/              (build tag: windows)
    features.go            Enable Windows optional features
    features_test.go

  diagnose/
    diagnose.go            Shared diagnostic types
    diagnose_windows.go    Windows checks
    diagnose_linux.go      Linux checks
    diagnose_test.go
```

---

## Configuration

Config moves from `update_open-webui.config.sh` to YAML.
YAML is human-readable, supports lists natively, and Go has
mature parsing via `gopkg.in/yaml.v3`.

### Config file: `ow.yaml`

```yaml
ollama:
  port: 11434
  host: localhost
  image: ollama/ollama
  tag: latest
  container: ollama
  volume: ollama
  models:
    - llama3.2:1b

openwebui:
  port: 3000
  host: localhost
  image: ghcr.io/open-webui/open-webui
  tag: latest
  container: open-webui
  volume: open-webui

wsl:
  distro: Ubuntu

proxy:
  listen_address: 0.0.0.0
  listen_port: 3000
  connect_address: 127.0.0.1
  connect_port: 3000
```

### Config resolution order

1. Built-in defaults (compiled into the binary)
2. `ow.yaml` in the working directory
3. `~/.config/ow/ow.yaml` (user-level)
4. Environment variables (`OW_OLLAMA_PORT`, etc.)
5. CLI flags (`--ollama-port`, etc.)

Later sources override earlier ones. `ow config show` prints
the fully resolved config so the user can verify.

### Config struct

```go
type Config struct {
    Ollama   OllamaConfig   `yaml:"ollama"`
    OpenWebUI OpenWebUIConfig `yaml:"openwebui"`
    WSL      WSLConfig       `yaml:"wsl"`
    Proxy    ProxyConfig     `yaml:"proxy"`
}

type OllamaConfig struct {
    Port      int      `yaml:"port"`
    Host      string   `yaml:"host"`
    Image     string   `yaml:"image"`
    Tag       string   `yaml:"tag"`
    Container string   `yaml:"container"`
    Volume    string   `yaml:"volume"`
    Models    []string `yaml:"models"`
}
```

---

## Shared Packages

These packages compile on both platforms:

### `internal/exec`

Wraps `os/exec.Command` with logging and error handling.
Replaces `run_command()` from `repo_lib.sh`.

```go
type Runner struct {
    Logger *logging.Logger
}

func (r *Runner) Run(
    ctx context.Context,
    name string,
    args ...string,
) (string, error)

func (r *Runner) RunWithRetry(
    ctx context.Context,
    opts RetryOpts,
    name string,
    args ...string,
) (string, error)
```

Testable: inject a mock `Runner` in tests to avoid real
shell-outs. Production code uses the real `os/exec` runner.

### `internal/exec/retry`

Fibonacci backoff retry. Pure function, no side effects.
Replaces `retry_logic()` from `repo_lib.sh`.

```go
type RetryOpts struct {
    MaxAttempts int
    InitialA    time.Duration
    InitialB    time.Duration
}

func DefaultRetryOpts() RetryOpts {
    return RetryOpts{
        MaxAttempts: 5,
        InitialA:    10 * time.Second,
        InitialB:    10 * time.Second,
    }
}

// NextDelay returns the delay and the next Fibonacci pair.
func NextDelay(a, b time.Duration) (time.Duration, time.Duration) {
    return a, a + b
}
```

### `internal/logging`

Leveled logger writing to stderr and a log file.
Replaces `log_message()` from `repo_lib.sh`.

```go
type Level int

const (
    Error   Level = 0
    Warning Level = 1
    Info    Level = 2
    Debug1  Level = 3
    Debug2  Level = 4
)
```

Verbosity controlled by `--verbosity` flag or `OW_VERBOSITY`
environment variable.

---

## Security

This CLI runs as administrator on Windows and executes
privileged operations in WSL. A bug here can compromise
the host machine. Security is not optional.

### Threat Surface

- **Command injection via config**: config values
  (container names, image tags, hostnames) are interpolated
  into shell commands. A malicious `ow.yaml` could inject
  arbitrary commands.
- **Privilege escalation**: the Windows CLI requires admin.
  The Linux CLI runs `apt-get` and modifies Docker config.
  Bugs in command construction could be exploited.
- **Supply chain**: the CLI downloads Docker images, GPG
  keys, and the Ollama install script from the internet.
  Compromised upstream sources affect the user.
- **Untrusted input in exec args**: all values passed to
  `os/exec.Command` must be arguments, never concatenated
  into a shell string.

### Mitigations

- **No shell interpretation**: always use `exec.Command`
  with explicit argument lists. Never pass user-controlled
  values through `sh -c` or `powershell.exe -Command`
  with string interpolation. When PowerShell cmdlets are
  required, pass arguments via `-ArgumentList`, not string
  concatenation.
- **Config validation**: validate all config values at load
  time. Ports must be integers in range 1-65535. Container
  names, volume names, and image tags must match
  `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`. Hostnames must match
  a strict pattern. Reject anything else before it reaches
  `exec.Command`.
- **Allowlisted executables**: the `Runner` only executes
  from a hardcoded list of allowed binaries (`docker`,
  `apt-get`, `wsl.exe`, `netsh`, `powershell.exe`,
  `ollama`, `curl`, `lsof`, `nvidia-smi`). Any attempt
  to run an unlisted binary is an error.
- **Pin upstream sources**: Docker GPG key fingerprints
  are hardcoded and verified. Ollama install script URL
  is pinned. Docker image digests can be pinned in config
  for production use.
- **Least privilege**: document which commands need admin
  and which do not. Future: split the Windows CLI so only
  the WSL/netsh steps require elevation.
- **No secrets in config**: `ow.yaml` contains no
  credentials. If future features need secrets, use OS
  credential stores (Windows Credential Manager, Linux
  keyring), not config files.
- **Audit logging**: all `exec.Command` invocations are
  logged with the full argument list so the user can
  review what ran. Log files are written with 0600
  permissions.

### Static Analysis

- `gosec` in CI to catch common Go security issues
- `golangci-lint` with security-related linters enabled
  (`gosec`, `gocritic`, `bodyclose`, `noctx`)
- Dependabot or `govulncheck` for dependency CVEs

---

## Testing Strategy

### Unit Tests

Every package has `_test.go` files. Run with `go test ./...`.

Key areas:

- **Config**: loading, defaults, override precedence,
  validation, rejection of invalid/malicious values
- **Retry**: delay sequence, max attempts, edge cases
- **Runner**: mock exec to verify logging, retry, error
  handling, and allowlist enforcement without real
  shell-outs
- **Docker client**: mock `docker` CLI output to test
  container state parsing
- **Port proxy**: mock `netsh` output to test rule parsing
  and idempotency logic
- **Input validation**: fuzz tests for config fields to
  verify injection attempts are rejected

### Integration Tests

Build-tagged tests (`//go:build integration`) that run real
commands. Gated behind `go test -tags integration ./...`.
Useful for CI with Docker available, not required locally.

### Mutation Testing

Run `go-mutesting` or `gremlins` at the end of each
milestone to verify test suite quality. Mutation testing
modifies the source (e.g., flips conditionals, removes
statements) and checks that tests catch the change.

Targets:

- `internal/exec` (retry logic, allowlist enforcement)
- `internal/config` (validation, override precedence)
- `internal/docker` (state machine transitions)
- `internal/netsh` (rule parsing, idempotency)

A mutation score below 80% on these packages blocks the
milestone PR from merging.

### Test Coverage Target

Aim for high coverage on `internal/` packages. The `cmd/`
layer is thin (just wiring), so coverage there matters less.

---

## Build

### Makefile targets

```text
make build           Build for current platform
make build-windows   GOOS=windows GOARCH=amd64
make build-linux     GOOS=linux GOARCH=amd64
make build-all       Both platforms
make test            go test ./...
make lint            golangci-lint run
make clean           Remove build artifacts
```

### Version injection

```text
go build -ldflags "-X main.version=1.0.0
  -X main.commit=$(git rev-parse --short HEAD)
  -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

`ow version` prints version, commit, date, GOOS, GOARCH.

### Release artifacts

```text
dist/
  ow-windows-amd64.exe
  ow-linux-amd64
  ow.yaml.example
```

---

## CI Pipeline

GitHub Actions workflow with two jobs:

### `build-and-test` (runs on every push and PR)

1. `go vet ./...`
2. `golangci-lint run` (includes `gosec`)
3. `go test -race -coverprofile=coverage.out ./...`
4. `go build` for `linux/amd64`
5. `go build` for `windows/amd64`
6. Upload coverage to Codecov (optional)
7. `govulncheck ./...` for dependency CVEs

### `mutation-test` (runs on milestone PRs only)

Triggered by PR label `milestone` or manually via
`workflow_dispatch`.

1. Install `gremlins` (or `go-mutesting`)
2. Run mutation testing on `internal/` packages
3. Fail if mutation score < 80%
4. Post mutation report as PR comment

### `release` (runs on tag push)

1. `go build` both platforms with `-ldflags` version info
2. Create GitHub release with artifacts
3. Generate checksums (`sha256sum`)

### Linting configuration

`golangci-lint` config (`.golangci.yml`) enables:

- `gosec` for security
- `gocritic` for code quality
- `bodyclose` for unclosed HTTP bodies
- `noctx` for missing context in HTTP requests
- `errcheck` for unchecked errors
- `govet` for suspicious constructs
- `staticcheck` for general correctness

Markdownlint continues to run for `docs/*.md` files via
the existing `build_and_test.sh --markdown` path.

---

## User Experience

### Clean Windows 11 desktop, zero dependencies

The user downloads `ow.exe` and `ow.yaml` (optional, for
custom config). They right-click `ow.exe` and select "Run as
administrator", or open an admin terminal and run:

```text
ow.exe setup
```

The CLI:

1. Prints what it will do
2. Enables WSL and VirtualMachinePlatform (may require reboot)
3. Installs WSL2 and Ubuntu
4. Prompts user to set Linux username and password
5. Copies the Linux `ow` binary into WSL
6. Runs `ow setup` inside WSL (visible output streamed)
7. Sets up port proxy and firewall rule
8. Prints: "OpenWebUI is running at <http://localhost:3000>"

If a reboot is required (step 2), the CLI prints a message
and exits. The user re-runs `ow.exe setup` after reboot and
it picks up where it left off (idempotent).

### Day-2 operations

From Windows:

```text
ow.exe diagnose            Check health from Windows
ow.exe proxy show          Verify port proxy rules
```

From WSL:

```text
ow containers status       Check container state
ow ollama models           List installed models
ow ollama pull             Pull new default models
ow ollama run              Interactive model selection
ow diagnose                Full Linux-side diagnostics
ow containers down         Stop everything
ow containers up           Start everything
```

---

## Migration Plan

Work is organized into milestones. Each milestone is a set
of PRs that land on `main`. Shell scripts remain functional
until the final cutover. Mutation testing runs at the end
of each milestone before the milestone PR merges.

### Milestone 1: Foundation

Goal: Go module, CI, and shared packages. No CLI yet.

| PR   | Scope                                        |
|------|----------------------------------------------|
| 1.1  | `go mod init`, project structure, Makefile   |
| 1.2  | CI workflow (build, lint, test, govulncheck) |
| 1.3  | `internal/logging` with tests                |
| 1.4  | `internal/exec` runner + allowlist + tests   |
| 1.5  | `internal/exec` retry + tests                |
| 1.6  | `internal/config` struct, YAML, validation   |
| 1.7  | `internal/config` env + flag override, tests |
| 1.8  | `.golangci.yml` with security linters        |
| 1.9  | Mutation test run, fix gaps, merge milestone |

### Milestone 2: Linux CLI

Goal: `ow` binary handles full Linux setup and day-2 ops.

| PR   | Scope                                        |
|------|----------------------------------------------|
| 2.1  | `cmd/ow` scaffold with cobra, root + version |
| 2.2  | `internal/docker` container lifecycle, tests |
| 2.3  | `internal/docker` health checks + tests      |
| 2.4  | `internal/apt` packages + keyring + tests    |
| 2.5  | `internal/nvidia` toolkit install + tests    |
| 2.6  | `internal/ollama` install + models + tests   |
| 2.7  | `ow setup` Linux command (wires everything)  |
| 2.8  | `ow containers` subcommands (up/down/status) |
| 2.9  | `ow ollama` subcommands (pull/run/models)    |
| 2.10 | `ow diagnose` Linux implementation           |
| 2.11 | `ow config show` command                     |
| 2.12 | Integration test against WSL2                |
| 2.13 | Mutation test run, fix gaps, merge milestone |

### Milestone 3: Windows CLI

Goal: `ow.exe` binary handles full Windows setup and
day-2 ops.

| PR   | Scope                                        |
|------|----------------------------------------------|
| 3.1  | `internal/wsl` install, update, stop + tests |
| 3.2  | `internal/wsl` distro management + tests     |
| 3.3  | `internal/netsh` port proxy CRUD + tests     |
| 3.4  | `internal/netsh` firewall rules + tests      |
| 3.5  | `internal/winfeature` feature enable + tests |
| 3.6  | `ow setup` Windows (wires all steps)         |
| 3.7  | `ow wsl` subcommands (install/remove/stop)   |
| 3.8  | `ow proxy` subcommands (enable/remove/show)  |
| 3.9  | `ow diagnose` Windows implementation         |
| 3.10 | Integration test on clean Windows 11 VM      |
| 3.11 | Mutation test run, fix gaps, merge milestone |

### Milestone 4: Cutover

Goal: remove shell scripts, ship v1.0.0.

| PR   | Scope                                        |
|------|----------------------------------------------|
| 4.1  | Remove bash scripts, PowerShell scripts      |
| 4.2  | Update README with Go CLI instructions       |
| 4.3  | Add `ow.yaml.example` to repo root           |
| 4.4  | Release workflow (tag, GitHub release)       |
| 4.5  | Final mutation test, tag v1.0.0              |

---

## Dependencies

External dependencies, kept minimal:

- `github.com/spf13/cobra` for CLI subcommands and help
- `github.com/spf13/viper` for config file search, env var
  binding, and flag override. Handles the config resolution
  chain so our code stays simple.
- `gopkg.in/yaml.v3` (pulled in by viper)
- Standard library for everything else (`os/exec`, `net/http`,
  `log/slog`, `testing`)

Dev/CI dependencies:

- `github.com/golangci/golangci-lint` for linting
- `github.com/securego/gosec` (via golangci-lint)
- `golang.org/x/vuln/cmd/govulncheck` for CVE scanning
- `github.com/go-gremlins/gremlins` for mutation testing

No Docker SDK. The CLI shells out to the `docker` CLI, same
as the current scripts. This avoids a large dependency and
matches the existing behavior.

---

## Key Design Decisions

### Shell out to `docker` CLI instead of Docker SDK

Rationale: the Docker Go SDK is large and complex. The current
scripts already shell out to `docker`. Keeping that pattern
means the Go code is a thin wrapper with logging and retry.
The `docker` CLI must be installed anyway for GPU support.

### YAML config instead of bash config

Rationale: Go can parse YAML natively. The bash config format
required a custom parser in PowerShell and would need another
custom parser in Go. YAML supports lists (for models) without
workarounds. The tradeoff is that existing users need to
migrate their config once.

### Single binary name (`ow`) for both platforms

Rationale: simpler for documentation and user muscle memory.
`ow setup` works the same way regardless of platform. The
binary is platform-specific at build time, not runtime.

### `internal/` packages, not `pkg/`

Rationale: these packages are not intended for external
import. `internal/` enforces that at the compiler level.

### cobra + viper for CLI and config

Rationale: cobra handles subcommand routing, help text, flag
parsing, and shell completions. Viper handles config file
discovery, environment variable binding, and the override
chain (defaults, file, env, flags) so our config code is
a struct definition and a few `viper.BindPFlag` calls instead
of hand-rolled merge logic. Both are well-audited, widely
used, and maintained by the same team.

### Allowlisted executables in the runner

Rationale: this CLI runs as administrator and constructs
shell commands from config values. An open `exec.Command`
is a command injection risk. The allowlist constrains
what binaries the runner can invoke, limiting blast radius
if a config value is malicious or a bug constructs a bad
command. The list is small and stable (docker, apt-get,
wsl.exe, netsh, powershell.exe, ollama, curl, lsof,
nvidia-smi).
