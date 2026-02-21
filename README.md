# OpenWebUI WSL2 Setup

Automated setup for **OpenWebUI** and **Ollama** inside a WSL2
Ubuntu environment, with Docker, NVIDIA container tools, and
customizable configuration.

## Prerequisites

- Windows 11
- Administrative privileges

## Configuration

Edit `update_open-webui.config.sh` before running setup to
customize ports, container names, volumes, and default models.

```bash
# Ollama Configuration
OLLAMA_PORT=11434
OLLAMA_HOST="localhost"
OLLAMA_CONTAINER_TAG="latest"
OLLAMA_CONTAINER_NAME="ollama"
OLLAMA_VOLUME_NAME="ollama"

# Open WebUI Configuration
OPEN_WEBUI_PORT=3000
OPEN_WEBUI_HOST="localhost"
OPEN_WEBUI_CONTAINER_TAG="latest"
OPEN_WEBUI_CONTAINER_NAME="open-webui"
OPEN_WEBUI_VOLUME_NAME="open-webui"

# Default Ollama Models
DEFAULT_OLLAMA_MODELS=(
    "llama3.2:1b"
)
```

## Phase 1: Windows Setup (WSL2 + Ubuntu)

Enables WSL2, installs Ubuntu, and configures the Windows host.

```cmd
RUNME.cmd
```

This runs `RUNME.ps1`, which:

- Enables WSL and VirtualMachinePlatform Windows features
- Installs or updates WSL2 and sets it as the default version
- Installs the Ubuntu distribution
- Configures Windows port proxy so the host can reach
  WSL containers

Once Windows setup completes, it automatically hands off to
Phase 2.

**Standalone scripts** (in `powershell/`):

- `Install-Ubuntu.ps1` - Install Ubuntu into WSL2
- `Update-Wsl2.ps1` - Update WSL2
- `Remove-Ubuntu.ps1` - Remove the Ubuntu distribution

## Phase 2: Ubuntu Setup (Docker, Containers, Models)

Runs inside WSL2 via `update_open-webui.sh`. Installs the full
stack:

1. Updates system packages
2. Installs Docker and the NVIDIA container toolkit
3. Installs Ollama
4. Deploys the Ollama container (port 11434)
5. Deploys the OpenWebUI container (port 3000), connected
   to Ollama via `OLLAMA_BASE_URL`
6. Verifies both containers are running

To re-run setup after changing configuration:

```cmd
RUNME.cmd
```

**Downloading models:**

The default model (`llama3.2:1b`) is configured in
`update_open-webui.config.sh`. To pull a broader set of models
(coding, general, quantized variants for 32GB VRAM):

```bash
bash ollama/scripts/get_models.sh
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

**Ollama CLI (interactive):**

```bash
# Interactive model selection
bash ollama/scripts/ollama_run.sh

# Specify a model directly
bash ollama/scripts/ollama_run.sh --model llama3.2:1b
```

**Diagnostics:**

```bash
bash ollama/scripts/diagnose_ollama.sh
bash open-webui/scripts/diagnose_open-webui.sh
```

PowerShell health checks are also available:

```powershell
.\ollama\scripts\Test-OllamaHealth.ps1
.\open-webui\scripts\Test-OpenWebUIHealth.ps1
```

## Logs

Setup logs are written adjacent to the running script:

```text
<ScriptDirectory>/update_open-webui.sh.log
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
