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

# Source project functions and variables
project_root="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
project_lib_sh="${project_root}/scripts/common/com_lib.sh"
source_required_file "${project_lib_sh}"

# Source global functions and variables
repo_env_sh="${repo_root}/bash/common/repo_env.sh"
source_required_file $repo_env_sh

# Local variables
script_dir="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
script_name="$(basename "$(realpath "${BASH_SOURCE[0]}")")"
log_file="${script_dir}/${script_name}.log"

# Source local functions and variables
open_webui_config_sh="${repo_root}/update_open-webui.config.sh"
source_required_file $open_webui_config_sh