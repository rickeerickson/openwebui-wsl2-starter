# OpenWebUI WSL2 Setup

This repository provides an automated setup script to install **OpenWebUI** and **Ollama** inside a WSL2 Ubuntu environment. It configures Docker, NVIDIA container tools, and ensures OpenWebUI runs successfully.

## Features

- Sets up **WSL2** with Ubuntu.
- Installs and configures **Docker** and **NVIDIA container toolkit**.
- Deploys **Ollama** and **OpenWebUI** Docker containers.
- Includes customizable port and container configurations.

## Prerequisites

- **Windows 10/11** with WSL2 enabled.
- Administrative privileges to run the setup script.

## Configuration

The script reads configuration values from a `config.env` file. You can customize the following settings:

### `config.sh`
```bash
# Ollama Configuration
OLLAMA_HOST_PORT=11434
OLLAMA_HOST="localhost"
OLLAMA_CONTAINER_TAG="latest"

# Open WebUI Configuration
OPEN_WEBUI_HOST_PORT=3000
OPEN_WEBUI_HOST="localhost"
OPEN_WEBUI_CONTAINER_TAG="latest"
```

- Edit the `config.sh` file to suit your network or container requirements.

## Usage

1. **Clone this repository**:
   ```bash
   git clone git@github.com:<user_name>/openwebui-wsl2-starter.git
   cd openwebui-wsl2-starter
   ```

2. **Run the setup script** in PowerShell:
   ```powershell
   .\Setup-OpenWebUI.ps1
   ```

   - The script will:
     1. Ensure WSL2 and Ubuntu are set up.
     2. Configure Docker and NVIDIA container tools.
     3. Deploy **Ollama** and **OpenWebUI** using the settings in `config.env`.

3. **Verify**:
   - The script will launch WSL interactively and display `docker ps` to confirm that containers are running.

## Customization

- Update the `config.env` file to change ports or container tags.
- Restart the setup to apply changes:
   ```powershell
   .\Setup-OpenWebUI.ps1
   ```

## Output

At the end of the script:
- OpenWebUI will be accessible at:
   ```
   http://localhost:3000
   ```
- Ollama will be running on:
   ```
   http://localhost:11434
   ```

## Logs

The setup logs are stored in a file adjacent to the running script:
```plaintext
<ScriptDirectory>/update_open-webui.sh.log
```

## Troubleshooting

1. **Port Conflicts**:
   - Ensure the ports configured in `config.env` are available.
   - Use the following to identify processes using a port:
     ```bash
     sudo lsof -i :<PORT>
     ```

2. **Docker Issues**:
   - Restart Docker if necessary:
     ```bash
     sudo systemctl restart docker
     ```

3. **NVIDIA Toolkit**:
   - Verify the toolkit installation:
     ```bash
     nvidia-smi
     ```

## Contributions

Contributions are welcome! Please submit a pull request or open an issue.

## License

This project is licensed under the MIT License.