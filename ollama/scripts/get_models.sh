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
OLLAMA_MODELS=(
    "codegemma:7b"
    "codellama:13b"
    "codellama:34b"
    "codellama:70b"
    "codellama:7b"
    "deepseek-coder-v2:16b"
    "deepseek-r1:14b"
    "deepseek-r1:32b"
    "deepseek-r1:8b"
    "gemma2:27b"
    "gemma2:9b"
    "llama2:7b"
    "llama3.2:3b"
    "llama3.3:70b"
    "mistral:7b"
    "phi4-mini:3.8b"
    "phi4:14b"
    "qwq:32b"
)
    
pull_ollama_models "${OLLAMA_MODELS[*]}"