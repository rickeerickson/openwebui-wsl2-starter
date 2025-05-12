#!/bin/bash
set -euo pipefail
trap 'echo "Error occurred in script: ${BASH_SOURCE[0]}, line: ${LINENO}, command: ${BASH_COMMAND}" >&2' ERR

source_required_file() {
    local filepath="$1"

    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo "$(date '+%Y.%m.%d:%H:%M:%S') - DEBUG: ${log_file:-${0##*/}}: ${0}: ${BASH_SOURCE[1]}::${FUNCNAME[1]}::${BASH_LINENO[1]} - ${BASH_SOURCE[0]}::${FUNCNAME[0]}::${BASH_LINENO[0]} -> sourcing ${filepath}" >&2
    fi

    if [[ -f "$filepath" ]]; then
        source "$filepath"
    else
        echo "$(date '+%Y.%m.%d:%H:%M:%S') - ERROR: ${log_file:-${0##*/}}: ${0}: ${BASH_SOURCE[1]}::${FUNCNAME[1]}::${BASH_LINENO[1]} - ${BASH_SOURCE[0]}::${FUNCNAME[0]}::${BASH_LINENO[0]} -> required file ${filepath} not found." >&2
        exit 1
    fi
}

project_root="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
project_env_sh="${project_root}/common/com_env.sh"
source_required_file "${project_env_sh}"

config_file="${repo_root}/update_open-webui.config.sh"
source_required_file "${config_file}"

script_dir="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
script_name="$(basename "$(realpath "${BASH_SOURCE[0]}")")"
log_file="${script_dir}/${script_name}.log"


# Ollama Models
# -------------------------------
# Full (FP16) models
OLLAMA_MODELS=(
    # Coding
    "codegemma:7b"
    "codellama:7b"
    "codellama:13b"
    "codellama:34b"
    "codellama:70b"

    # General-purpose LLMs
    "llama2:7b"
    "mistral:7b"
    "gemma2:9b"
    "gemma2:27b"
    "llama3.2:3b"
    "llama3.3:70b"
    "llama4:latest"
    "mistral-small3.1:24b"

    # Reasoning & Misc
    "deepseek-r1:8b"
    "deepseek-r1:14b"
    "deepseek-r1:32b"

    # Tiny & Edge
    "phi4-mini:3.8b"
    "phi4:14b"
    "qwen3:32b"
    "qwq:32b"

    # Coding-focused specialist
    "deepseek-coder-v2:16b"
)

# Optimized (quantized) variants for 32 GB VRAM
OLLAMA_MODELS+=(
    # Coding speed vs. accuracy
    "codegemma:7b-code-q4_K_M"
    "codegemma:7b-instruct-q8_0"
    "codellama:70b-instruct-q4_K_M"

    # LLMs speed vs. accuracy
    "llama2:7b-chat-q4_K_M"
    "mistral:7b-instruct-q8_0"
    "gemma2:9b-instruct-q5_1"
    "gemma2:27b-instruct-q4_K_M"
    "llama3.2:3b-instruct-q4_K_M"
    "llama3.3:70b-instruct-q4_K_M"

    # Reasoning & Misc quantized
    "deepseek-r1:8b-llama-distill-q4_K_M"
)

# Pull all models
echo "Pulling Ollama models:"
pull_ollama_models "${OLLAMA_MODELS[*]}"
