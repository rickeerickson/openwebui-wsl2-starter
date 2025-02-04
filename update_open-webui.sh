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
project_env_sh="${project_root}/bash/common/repo_env.sh"
config_file="${project_root}/update_open-webui.config.sh"

source_required_file "${project_env_sh}"
source_required_file "${config_file}"

script_dir="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
script_name="$(basename "$(realpath "${BASH_SOURCE[0]}")")"
log_file="${script_dir}/${script_name}.log"

update_system_packages
setup_docker_keyring
install_nvidia_container_toolkit
install_and_configure_docker
install_ollama

verify_docker_environment

pull_docker_image "ollama/ollama" "${OLLAMA_CONTAINER_TAG}"
ensure_port_available "${OLLAMA_PORT}"
stop_remove_run_ollama_container "${OLLAMA_HOST}" "${OLLAMA_PORT}" "${OLLAMA_CONTAINER_TAG}"
verify_ollama_setup "${OLLAMA_HOST}" "${OLLAMA_PORT}"
pull_ollama_models

pull_docker_image "ghcr.io/open-webui/open-webui" "${OPEN_WEBUI_CONTAINER_TAG}"
ensure_port_available "${OPEN_WEBUI_PORT}"
stop_remove_run_open_webui_container "${OLLAMA_HOST}" "${OLLAMA_PORT}" "${OPEN_WEBUI_HOST}" "${OPEN_WEBUI_PORT}" "${OPEN_WEBUI_CONTAINER_TAG}"
verify_open_webui_setup "${OPEN_WEBUI_HOST}" "${OPEN_WEBUI_PORT}"

log_message "${script_name} completed successfully."
