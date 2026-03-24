# Design

## System Overview

The system is a two-phase automation toolkit. The Windows phase
installs WSL2 and Ubuntu. The Linux phase installs Docker, NVIDIA
tooling, and the Ollama and OpenWebUI containers inside WSL2.
A port proxy bridges the WSL2 network to the Windows host so
the chat interface is accessible from Windows and the LAN.

---

## Execution Flow

```text
RUNME.cmd (Windows batch wrapper)
  └─ RUNME.ps1 (PowerShell, requires admin)
      ├─ Enable Windows features (WSL, VirtualMachinePlatform)
      ├─ Install or update WSL2
      ├─ Set WSL default version to 2
      ├─ Install Ubuntu interactively (user sets password)
      ├─ Shut down WSL
      ├─ Run update_open-webui.sh in WSL
      │   ├─ update_system_packages()
      │   ├─ setup_docker_keyring()
      │   ├─ install_nvidia_container_toolkit()
      │   ├─ install_and_configure_docker()
      │   ├─ install_ollama()
      │   ├─ verify_docker_environment()
      │   ├─ pull_docker_image("ollama/ollama")
      │   ├─ ensure_port_available(OLLAMA_PORT)
      │   ├─ stop_remove_run_ollama_container()
      │   ├─ verify_ollama_setup()
      │   ├─ pull_ollama_models(DEFAULT_OLLAMA_MODELS)
      │   ├─ pull_docker_image("ghcr.io/open-webui/open-webui")
      │   ├─ ensure_port_available(OPEN_WEBUI_PORT)
      │   ├─ stop_remove_run_open_webui_container()
      │   └─ verify_open_webui_setup()
      ├─ Enable-OpenWebUIPortProxyIfNeeded()
      │   ├─ netsh portproxy add v4tov4 (0.0.0.0:3000 -> 127.0.0.1:3000)
      │   └─ New-NetFirewallRule (inbound port 3000)
      └─ Launch interactive WSL bash session
```

---

## Component Map

```text
openwebui-wsl2-starter/
├── RUNME.cmd                         Windows entry point (batch)
├── RUNME.ps1                         Windows entry point (PowerShell)
├── RUNME.sh                          Linux help display
├── update_open-webui.sh              Linux orchestration
├── update_open-webui.config.sh       Shared configuration
├── build_and_test.sh                 Lint and test runner (Linux/macOS)
├── build_and_test.cmd                Lint and test runner (Windows)
├── bash/
│   ├── common/
│   │   ├── repo_env.sh               Logging constants, REPO_ROOT, ERR trap
│   │   └── repo_lib.sh               All shared Bash functions (~850 lines)
│   └── tests/
│       └── test_run_command.sh       Unit tests for run_command()
├── ollama/
│   └── scripts/
│       ├── common/
│       │   ├── com_env.sh            Ollama env init (sources repo_env.sh)
│       │   └── com_lib.sh            Ollama-specific functions (placeholder)
│       ├── ollama_run.sh             Interactive model runner
│       ├── get_models.sh             Bulk model pull (85+ models)
│       ├── diagnose_ollama.sh        Bash diagnostics for Ollama
│       └── Test-OllamaHealth.ps1     Windows health check for Ollama
├── open-webui/
│   └── scripts/
│       ├── common/
│       │   ├── com_env.sh            OpenWebUI env init (sources repo_env.sh)
│       │   └── com_lib.sh            OpenWebUI-specific functions (placeholder)
│       ├── diagnose_open-webui.sh    Bash diagnostics for OpenWebUI
│       └── Test-OpenWebUIHealth.ps1  Windows health check for OpenWebUI
└── powershell/
    ├── CommonLibrary.psm1            All shared PowerShell functions (~450 lines)
    ├── Install-Ubuntu.ps1            Standalone Ubuntu installer
    ├── Update-Wsl2.ps1               WSL update utility
    ├── Remove-Ubuntu.ps1             WSL Ubuntu removal
    └── Remove-NetShBindings.ps1      Port proxy cleanup
```

---

## Library Hierarchy

Each script sources its environment through a chain:

```text
Script (e.g., ollama_run.sh)
  └─ sources com_env.sh
      ├─ sources com_lib.sh          (component-specific functions)
      └─ sources repo_env.sh
          ├─ sources repo_lib.sh     (all shared functions)
          ├─ defines REPO_ROOT
          ├─ defines LEVEL_* constants
          └─ sets up ERR trap + log file path
```

Sourcing uses `source_required_file()` at each level to validate
the file exists before sourcing. This prevents silent failures.

`update_open-webui.sh` sources `repo_env.sh` and
`update_open-webui.config.sh` directly, since it is the
top-level orchestrator and does not need a component `com_env.sh`.

---

## Configuration Design

All tunable values live in `update_open-webui.config.sh`:

| Variable                    | Default           | Purpose                    |
|-----------------------------|-------------------|----------------------------|
| `OLLAMA_PORT`               | `11434`           | Ollama API port            |
| `OLLAMA_HOST`               | `localhost`       | Ollama bind address        |
| `OLLAMA_CONTAINER_TAG`      | `latest`          | Docker image tag           |
| `OLLAMA_CONTAINER_NAME`     | `ollama`          | Docker container name      |
| `OLLAMA_VOLUME_NAME`        | `ollama`          | Docker volume for models   |
| `OPEN_WEBUI_PORT`           | `3000`            | OpenWebUI HTTP port        |
| `OPEN_WEBUI_HOST`           | `localhost`       | OpenWebUI bind address     |
| `OPEN_WEBUI_CONTAINER_TAG`  | `latest`          | Docker image tag           |
| `OPEN_WEBUI_CONTAINER_NAME` | `open-webui`      | Docker container name      |
| `OPEN_WEBUI_VOLUME_NAME`    | `open-webui`      | Docker volume for app data |
| `DEFAULT_OLLAMA_MODELS`     | `("llama3.2:1b")` | Models pulled on setup     |

PowerShell reads this file via `ParseBashConfig()`, which parses
`VAR=value` lines into a hashtable. Both languages share a single
source of truth.

---

## Container Design

### Ollama Container

```bash
docker run -d \
  --gpus all \
  --network=host \
  --volume ollama:/root/.ollama \
  --env OLLAMA_HOST="localhost" \
  --restart always \
  --name ollama \
  ollama/ollama:latest
```

- `--network=host` is required for NVIDIA GPU access from the
  container in WSL2.
- `--gpus all` passes all NVIDIA GPUs.
- The volume persists downloaded models across container recreations.
- `--restart always` ensures the container restarts after WSL
  or Docker restarts.

### OpenWebUI Container

```bash
docker run -d \
  --gpus all \
  --network=host \
  --volume open-webui:/app/backend/data \
  --env OLLAMA_BASE_URL="http://localhost:11434" \
  --env PORT="3000" \
  --restart always \
  --name open-webui \
  ghcr.io/open-webui/open-webui:latest
```

- Connects to Ollama via `OLLAMA_BASE_URL` using `localhost`
  because both containers share the host network stack.
- The volume persists user data, settings, and chat history.

---

## Networking Model

```text
LAN client
    |
    | TCP :3000
    v
Windows host (0.0.0.0:3000)
    |
    | netsh portproxy v4tov4
    v
WSL2 loopback (127.0.0.1:3000)
    |
    | host network stack
    v
OpenWebUI container (:3000)
    |
    | http://localhost:11434
    v
Ollama container (:11434)
    |
    | GPU inference
    v
NVIDIA GPU (via --gpus all + --network=host)
```

The port proxy (managed by `Enable-OpenWebUIPortProxyIfNeeded`)
forwards any-address port 3000 on Windows to the WSL2 loopback.
A Windows Firewall inbound rule is created to allow the traffic.

---

## Retry Strategy

All external commands use Fibonacci backoff via
`run_command_with_retry()`.

| Parameter       | Value                           |
|-----------------|---------------------------------|
| Max attempts    | 5                               |
| Delay sequence  | 10, 10, 20, 30, 50 seconds      |
| Applied to      | apt, docker pull, docker run,   |
|                 | container status checks,        |
|                 | ollama pull, HTTP verifications |

The helper `retry_logic()` is a pure function that takes current
retry state and returns the next state as a string
(`retry_count fib1 fib2`). The caller reads it via `read`.
This design keeps the retry logic testable and free of side effects.

PowerShell uses `Start-CommandWithRetry()` with the same strategy.

---

## Logging System

### Bash Logging

- Function: `log_message(message, level)`
- Format: `YYYY.MM.DD:HH:MM:SS - [LEVEL]: message`
- Output: stderr and `${script_dir}/${script_name}.log`
- Levels controlled by `VERBOSITY` environment variable

```text
LEVEL_ERROR   = 0
LEVEL_WARNING = 1
LEVEL_INFO    = 2   (default)
LEVEL_DEBUG_1 = 3
LEVEL_DEBUG_2 = 4
```

### PowerShell Logging

- Function: `Write-Log(Message, Level)`
- Same format and level constants as Bash (offset by 1)
- Output: Tee-Object to host and `${script_name}.log`

---

## Error Handling

### Bash Error Handling

- `set -euo pipefail` in all scripts
- ERR trap logs file name, line number, and failing command
- `run_command()` handles optional `ignore_exit_status` and
  `should_fail` flags for expected failures (e.g., grep returning 1)
- `source_required_file()` exits with a clear message if a
  required file is missing

### PowerShell Error Handling

- `Set-StrictMode -Version Latest`
- `$ErrorActionPreference = 'Stop'`
- `Start-CommandWithRetry()` handles retry and exit status flags

---

## Idempotency Patterns

### Container Lifecycle

Before any container operation, the script checks current state:

```text
container_exists(name)?
  no  -> skip stop/remove, proceed to run
  yes -> container_is_running(name)?
             yes -> try_stop_container(); wait_for_container_stop()
             no  -> skip stop
         remove_container(name)
         proceed to run
```

Functions: `container_exists()`, `container_is_running()`,
`container_is_stopped()`, `stop_and_remove_container()`,
`wait_for_container_status_up()`, `wait_for_container_stop()`.

### Port Proxy

`Enable-OpenWebUIPortProxyIfNeeded()` reads existing netsh rules,
checks if the exact rule already exists, removes conflicting rules
(same listen addr:port but different connect target), then creates
the rule only if needed.

### System Packages and Keys

`setup_docker_keyring()` checks for existing Docker apt sources
before adding. `install_ollama()` checks if Ollama is already
installed before running the install script.

---

## Testing

### Unit Tests

`bash/tests/test_run_command.sh` tests `run_command()` with
11 scenarios covering:

- Success and failure paths
- `ignore_exit_status` flag
- `should_fail` flag (expects non-zero exit)
- Combinations of flags
- Empty commands

Enable verbose output with `DEBUG=true`.

### Build and Test Pipeline

`build_and_test.sh` runs in stages and skips unavailable tools:

| Stage              | Tool             | Scope               |
|--------------------|------------------|---------------------|
| Bash syntax        | `bash -n`        | All `.sh` files     |
| Shell lint         | shellcheck       | All `.sh` files     |
| Markdown lint      | markdownlint     | All `.md` files     |
| PowerShell syntax  | `pwsh` parse     | All `.ps1/.psm1`    |
| PowerShell lint    | PSScriptAnalyzer | All `.ps1/.psm1`    |
| Unit tests         | bash             | `bash/tests/test_*` |

Flags: `--install` bootstraps missing tools, `--fix` auto-fixes
markdownlint issues, `--shell/--markdown/--powershell/--test` run
individual stages.

### Health Checks

Diagnostic scripts run from both contexts:

| Script                          | Context   | Checks                    |
|---------------------------------|-----------|---------------------------|
| `diagnose_ollama.sh`            | WSL2 bash | System, network, Ollama   |
| `diagnose_open-webui.sh`        | WSL2 bash | System, network, WebUI    |
| `Test-OllamaHealth.ps1`         | Windows   | WSL diag + port proxy     |
| `Test-OpenWebUIHealth.ps1`      | Windows   | WSL diag + firewall + TCP |

---

## Key Design Decisions

### Single config file read by both Bash and PowerShell

Rationale: prevents config drift between platforms. The format
is valid Bash; PowerShell parses it with a simple regex. Only
simple `VAR=value` lines are supported (no arrays in PowerShell
parsing, arrays handled natively in Bash).

### `--network=host` for containers

Rationale: required for NVIDIA GPU passthrough in WSL2. The GPU
is accessible via the host network stack; bridge networking
cannot forward the GPU device.

### Fibonacci backoff over linear or exponential

Rationale: Fibonacci grows slower than exponential, avoiding
excessively long waits for transient failures (e.g., apt lock),
while still backing off enough to let services recover.

### `retry_logic()` as a pure function

Rationale: separating state calculation from execution makes
the retry logic independently testable and avoids shared mutable
state in the retry loop.

### Component `com_lib.sh` files as placeholders

Rationale: establishes the extension point for component-specific
functions without adding complexity prematurely. All current
shared logic remains in `repo_lib.sh`.
