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
    "codegemma:7b"
    "gemma2:9b"
)