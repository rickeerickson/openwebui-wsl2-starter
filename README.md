# OpenWebUI WSL2 Setup

Automated setup for **OpenWebUI** and **Ollama** inside a
WSL2 Ubuntu environment, with Docker, NVIDIA container
tools, and customizable configuration.

## Prerequisites

- Windows 11
- Administrative privileges
- Python 3.11+ (in WSL2; bootstrap.sh checks for it)

## Configuration

Edit `config.toml` before running setup to customize ports,
container names, volumes, and default models.

```toml
[ollama]
host = "localhost"
port = 11434
container_tag = "latest"
container_name = "ollama"
volume_name = "ollama"
image = "ollama/ollama"

[openwebui]
host = "localhost"
port = 3000
container_tag = "latest"
container_name = "open-webui"
volume_name = "open-webui"
image = "ghcr.io/open-webui/open-webui"

[models]
default = [
    "llama3.2:1b",
]
```

## Phase 1: Windows Setup (WSL2 + Ubuntu)

Enables WSL2, installs Ubuntu, and configures the Windows
host.

```cmd
RUNME.cmd
```

This runs `RUNME.ps1`, which:

- Enables WSL and VirtualMachinePlatform Windows features
- Installs or updates WSL2 and sets it as default
- Installs the Ubuntu distribution
- Configures Windows port proxy so the host can reach
  WSL containers

Once Windows setup completes, it automatically hands off
to Phase 2.

**Standalone scripts** (in `powershell/`):

- `Install-Ubuntu.ps1` - Install Ubuntu into WSL2
- `Update-Wsl2.ps1` - Update WSL2
- `Remove-Ubuntu.ps1` - Remove the Ubuntu distribution

## Phase 2: Ubuntu Setup (Docker, Containers, Models)

Runs inside WSL2 via `bootstrap.sh`, which creates a
Python venv, installs the `owui` package, and runs
`owui setup`. The setup flow:

1. Updates system packages
2. Installs Docker and the NVIDIA container toolkit
3. Deploys the Ollama container (port 11434)
4. Deploys the OpenWebUI container (port 3000),
   connected to Ollama via `OLLAMA_BASE_URL`
5. Pulls configured models
6. Verifies both containers are running

To re-run setup after changing configuration:

```cmd
RUNME.cmd
```

Or directly in WSL2:

```bash
./bootstrap.sh
```

## Phase 3: Usage

After setup, OpenWebUI and Ollama are running as Docker
containers with `--restart always`.

**OpenWebUI (browser):**

```text
http://localhost:3000
```

**Ollama API:**

```text
http://localhost:11434
```

**CLI commands** (activate the venv first, or use
`bootstrap.sh` which handles this):

```bash
# Interactive model selection
owui run

# Run a specific model
owui run --model llama3.2:1b

# Pull configured models
owui models pull

# List installed models
owui models list

# Show full config
owui config show

# Get a specific config value
owui config get openwebui.port
```

**Diagnostics:**

```bash
owui diagnose              # all checks
owui diagnose ollama       # Ollama only
owui diagnose openwebui    # OpenWebUI only
```

PowerShell health checks are also available:

```powershell
.\ollama\scripts\Test-OllamaHealth.ps1
.\open-webui\scripts\Test-OpenWebUIHealth.ps1
```

## Logs

Setup logs are written by the `owui` logger. Increase
verbosity with `-v` (repeat for more) or suppress
non-error output with `-q`:

```bash
owui -v setup
owui -vv diagnose
```

## Troubleshooting

**Port conflicts:**

```bash
sudo lsof -i :<PORT>
```

**Docker issues:**

```bash
sudo systemctl restart docker
```

**NVIDIA toolkit:**

```bash
nvidia-smi
```

## License

MIT License.
