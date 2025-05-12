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

# Default model name
DEFAULT_MODEL="${DEFAULT_OLLAMA_MODELS[0]}"
selected_model=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        -m|--model)
            if [[ -n "${2:-}" ]]; then
                selected_model="$2"
                shift 2
            else
                echo "Error: --model requires a value." >&2
                exit 1
            fi
            ;;
        -h|--help)
            echo "Usage: $0 [-m|--model MODEL_NAME]"
            echo "       $0 [-h|--help]"
            echo
            echo "Options:"
            echo "  -m, --model MODEL_NAME   Specify the model name to use directly."
            echo "  -h, --help               Show this help message and exit."
            exit 0
            ;;
        *)
            echo "Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

# List available models
list_models() {
    echo "Available models..."
    ollama list
    echo "Enter the model name to use (default: ${DEFAULT_MODEL}):"
}

# Prompt user to select a model
select_model() {
    read -r selected_model

    if [[ -z "${selected_model}" ]]; then
        selected_model="${DEFAULT_MODEL}"
    fi

    echo "${selected_model}"
}

ensure_ollama_running

if [[ -z "${selected_model}" ]]; then
    list_models
    selected_model=$(select_model)
fi

echo "Selected model: ${selected_model:-$DEFAULT_MODEL}"
echo "Running: ollama run \"${selected_model:-$DEFAULT_MODEL}\" --verbose"
ollama run "${selected_model:-$DEFAULT_MODEL}" --verbose
